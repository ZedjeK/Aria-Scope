package main

import (
	"errors"
	"fmt"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"

	"aria-scope/internal/menage"
	"aria-scope/internal/reglages"
	"aria-scope/internal/scan"
	"aria-scope/internal/volumes"
)

// Nombre maximal de lignes renvoyées par dossier déplié ; le reste est
// regroupé dans une ligne « N autres ». Évite de figer l'interface sur un
// dossier comme WinSxS (des dizaines de milliers de sous-dossiers).
const limiteLignes = 300

// ScopeService expose au frontend les actions d'Aria Scope.
type ScopeService struct {
	app      *application.App  // renseigné après application.New
	reglages *reglages.Magasin // nil si le dossier de configuration est introuvable

	mu      sync.Mutex
	scan    *scan.Scan
	gen     int   // numéro du scan, pour ignorer les demandes d'un ancien résultat
	attendu int64 // espace occupé du volume si on scanne une racine de disque

	archivage *menage.Archivage // le dernier lancé, en cours ou fini
}

// Etat est ce que l'interface interroge régulièrement.
type Etat struct {
	Version      string            `json:"version"`
	Gen          int               `json:"gen"`
	Scan         *scan.Progression `json:"scan"` // nil : aucun scan lancé
	Attendu      int64             `json:"attendu"`
	Parallelisme int               `json:"parallelisme"`
	Filtres      *scan.Filtres     `json:"filtres"` // filtres de l'analyse affichée ; nil s'il n'y en a pas
}

// Etat renvoie la version et l'avancement du scan.
func (s *ScopeService) Etat() Etat {
	s.mu.Lock()
	sc, gen, attendu := s.scan, s.gen, s.attendu
	s.mu.Unlock()
	e := Etat{Version: version, Gen: gen, Attendu: attendu, Parallelisme: scan.Parallelisme()}
	if sc != nil {
		p := sc.Progression()
		e.Scan = &p
		if f := sc.Filtres(); f.Actifs() {
			e.Filtres = &f
		}
	}
	return e
}

// Volumes liste les disques de la machine.
func (s *ScopeService) Volumes() []volumes.Volume { return volumes.Lister() }

// ChoisirDossier ouvre le sélecteur de dossier. Renvoie "" si l'utilisateur annule.
func (s *ScopeService) ChoisirDossier() (string, error) {
	if s.app == nil {
		return "", errors.New("application non prête")
	}
	return s.app.Dialog.OpenFile().
		SetTitle("Dossier à analyser").
		CanChooseDirectories(true).
		CanChooseFiles(false).
		CanCreateDirectories(false).
		PromptForSingleSelection()
}

// Lancer démarre l'analyse de chemin en appliquant les filtres (et arrête la
// précédente s'il y en a une).
func (s *ScopeService) Lancer(chemin string, filtres scan.Filtres) error {
	nouveau, err := scan.LancerAvec(chemin, filtres)
	if err != nil {
		return fmt.Errorf("impossible d'analyser %s : %w", chemin, err)
	}
	s.remplacer(nouveau, volumes.Utilise(nouveau.Racine()))
	return nil
}

// Reglages renvoie les préférences enregistrées.
func (s *ScopeService) Reglages() reglages.Reglages {
	if s.reglages == nil {
		return reglages.Defaut()
	}
	return s.reglages.Lire()
}

// SetAnalyseAuto active ou non l'analyse de tout le PC à l'ouverture.
func (s *ScopeService) SetAnalyseAuto(actif bool) (reglages.Reglages, error) {
	if s.reglages == nil {
		return reglages.Defaut(), errors.New("dossier de configuration introuvable : réglage non enregistré")
	}
	return s.reglages.Modifier(func(r *reglages.Reglages) { r.AnalyseAuto = actif })
}

// NomPC est le nom de la racine virtuelle qui réunit tous les disques.
const NomPC = "Ce PC"

// LancerPC analyse tous les disques internes et externes en même temps
// (pas les lecteurs réseau ni optiques), avec les filtres.
func (s *ScopeService) LancerPC(filtres scan.Filtres) error {
	disques := volumes.Locaux()
	var racines []string
	var attendu int64
	for _, v := range disques {
		racines = append(racines, v.Chemin)
		attendu += v.Total - v.Libre
	}
	nouveau, err := scan.LancerPC(NomPC, racines, filtres)
	if err != nil {
		return fmt.Errorf("impossible d'analyser le PC : %w", err)
	}
	s.remplacer(nouveau, attendu)
	return nil
}

// remplacer installe un nouveau scan et arrête le précédent.
func (s *ScopeService) remplacer(nouveau *scan.Scan, attendu int64) {
	s.mu.Lock()
	ancien := s.scan
	s.scan, s.attendu = nouveau, attendu
	s.gen++
	s.mu.Unlock()
	if ancien != nil {
		ancien.Arreter()
	}
}

// Arreter interrompt l'analyse en cours ; le résultat partiel reste affiché.
func (s *ScopeService) Arreter() {
	s.mu.Lock()
	sc := s.scan
	s.mu.Unlock()
	if sc != nil {
		sc.Arreter()
	}
}

// courant renvoie le scan correspondant à gen, ou une erreur s'il a été remplacé.
func (s *ScopeService) courant(gen int) (*scan.Scan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.scan == nil || gen != s.gen {
		return nil, errors.New("ce résultat a été remplacé par une nouvelle analyse")
	}
	return s.scan, nil
}

// Racine renvoie la ligne du dossier analysé.
func (s *ScopeService) Racine(gen int) (scan.Entree, error) {
	sc, err := s.courant(gen)
	if err != nil {
		return scan.Entree{}, err
	}
	return sc.RacineEntree()
}

// Enfants renvoie le contenu d'un dossier. mode choisit la taille mesurée
// ("taille" ou "disque"), tri la colonne et le sens.
func (s *ScopeService) Enfants(gen int, id int32, mode string, tri scan.Tri) ([]scan.Entree, error) {
	sc, err := s.courant(gen)
	if err != nil {
		return nil, err
	}
	return sc.Enfants(id, mode, tri, limiteLignes)
}

// Fichiers renvoie les fichiers posés directement dans un dossier.
func (s *ScopeService) Fichiers(gen int, id int32, mode string, tri scan.Tri) ([]scan.Entree, error) {
	sc, err := s.courant(gen)
	if err != nil {
		return nil, err
	}
	return sc.Fichiers(id, mode, tri, limiteLignes)
}

// GrosFichiers renvoie les 1 000 plus gros fichiers de l'analyse, triés selon tri.
func (s *ScopeService) GrosFichiers(gen int, mode string, tri scan.Tri) ([]scan.GrosFichier, error) {
	sc, err := s.courant(gen)
	if err != nil {
		return nil, err
	}
	return sc.GrosFichiers(mode, tri)
}

// Types renvoie la répartition par famille de fichiers du dossier id
// (0 : toute l'analyse).
func (s *ScopeService) Types(gen int, id int32, mode string) ([]scan.Famille, error) {
	sc, err := s.courant(gen)
	if err != nil {
		return nil, err
	}
	return sc.Types(id, mode)
}

// Ages renvoie la répartition par âge du dossier id (0 : toute l'analyse).
func (s *ScopeService) Ages(gen int, id int32) ([]scan.Age, error) {
	sc, err := s.courant(gen)
	if err != nil {
		return nil, err
	}
	return sc.Ages(id)
}

// Resumer récapitule une sélection (doublons écartés).
func (s *ScopeService) Resumer(gen int, cibles []scan.Cible) (scan.Resume, error) {
	sc, err := s.courant(gen)
	if err != nil {
		return scan.Resume{}, err
	}
	return sc.Resumer(cibles)
}

// Carte renvoie les blocs de la treemap du dossier id, mesurés selon mode.
func (s *ScopeService) Carte(gen int, id int32, mode string) (scan.Bloc, error) {
	sc, err := s.courant(gen)
	if err != nil {
		return scan.Bloc{}, err
	}
	return sc.Carte(id, mode)
}

// Fil renvoie le chemin de la racine analysée jusqu'au dossier id.
func (s *ScopeService) Fil(gen int, id int32) ([]scan.Etape, error) {
	sc, err := s.courant(gen)
	if err != nil {
		return nil, err
	}
	return sc.Fil(id)
}

// Montrer ouvre l'Explorateur sur un dossier, ou sur un fichier (sélectionné)
// si fichier n'est pas vide.
func (s *ScopeService) Montrer(gen int, id int32, fichier string) error {
	sc, err := s.courant(gen)
	if err != nil {
		return err
	}
	chemin, err := sc.Chemin(id)
	if err != nil {
		return err
	}
	if s.app == nil {
		return errors.New("application non prête")
	}
	if fichier != "" {
		return s.app.Env.OpenFileManager(joindre(chemin, fichier), true)
	}
	return s.app.Env.OpenFileManager(chemin, false)
}
