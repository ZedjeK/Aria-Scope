// Package scan parcourt une arborescence en parallèle et garde, pour chaque
// dossier, la taille de ce qu'il contient. Les fichiers ne sont pas gardés en
// mémoire un par un : on les relit à la demande (Fichiers), ce qui permet de
// scanner un disque entier sans exploser la RAM.
package scan

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Etat d'un scan.
const (
	EnCours = "scan"
	Fini    = "fini"
	Arrete  = "arrete"
)

// Modes de mesure : taille réelle des fichiers, ou place occupée sur le disque.
const (
	ModeTaille = "taille"
	ModeDisque = "disque"
)

// Dossier est un nœud de l'arbre. Pendant le scan, seul le goroutine qui lit
// ce dossier écrit dans ses champs ; les totaux sont calculés à la fin.
type Dossier struct {
	nom     string
	parent  *Dossier
	enfants []*Dossier
	id      int32

	// Contenu direct (fichiers posés dans ce dossier)
	nFichiers int64
	fTaille   int64
	fDisque   int64
	fModif    int64 // fichier le plus récemment modifié de ce dossier
	illisible bool  // ce dossier n'a pas pu être lu (accès refusé…)
	nLiens    int64

	// Totaux du sous-arbre, calculés par finaliser
	taille, disque     int64
	modif              int64
	fichiers, dossiers int64
	erreurs, liens     int64
}

// Progression est l'état affiché pendant et après le scan.
type Progression struct {
	Etat     string `json:"etat"`
	Racine   string `json:"racine"`
	Fichiers int64  `json:"fichiers"`
	Dossiers int64  `json:"dossiers"`
	Taille   int64  `json:"taille"`
	Disque   int64  `json:"disque"`
	Erreurs  int64  `json:"erreurs"`
	Liens    int64  `json:"liens"`
	Courant  string `json:"courant"`
	DureeMs  int64  `json:"dureeMs"`
}

// Scan est un parcours en cours ou terminé.
type Scan struct {
	racine  string
	debut   time.Time
	arreter context.CancelFunc
	sem     chan struct{}
	wg      sync.WaitGroup
	fini    chan struct{}

	fichiers, dossiers, taille, disque, erreurs, liens atomic.Int64
	courant                                            atomic.Pointer[string]

	mu      sync.RWMutex
	etat    string
	duree   time.Duration
	noeuds  []*Dossier // index = id, rempli à la fin
	racineD *Dossier
}

// ErrPasUnDossier : la cible du scan n'est pas un dossier.
var ErrPasUnDossier = errors.New("ce chemin n'est pas un dossier")

// ErrEnCours : le résultat n'est consultable qu'une fois le scan terminé ou arrêté.
var ErrEnCours = errors.New("scan en cours")

// Parallelisme : nombre de dossiers lus en même temps. Assez pour saturer un
// SSD NVMe, sans noyer un disque dur sous les requêtes.
func Parallelisme() int {
	n := runtime.NumCPU() * 2
	return max(8, min(n, 32))
}

// Lancer démarre le scan de racine en arrière-plan.
func Lancer(racine string) (*Scan, error) {
	abs, err := filepath.Abs(racine)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, ErrPasUnDossier
	}

	ctx, cancel := context.WithCancel(context.Background())
	s := &Scan{
		racine:  abs,
		debut:   time.Now(),
		arreter: cancel,
		sem:     make(chan struct{}, Parallelisme()),
		fini:    make(chan struct{}),
		etat:    EnCours,
		racineD: &Dossier{nom: abs},
	}
	s.courant.Store(&abs)

	s.wg.Add(1)
	go func() {
		s.parcourir(ctx, s.racineD, abs)
		s.wg.Done()
		s.wg.Wait()

		s.finaliser()
		s.mu.Lock()
		s.duree = time.Since(s.debut)
		if ctx.Err() != nil {
			s.etat = Arrete
		} else {
			s.etat = Fini
		}
		s.mu.Unlock()
		cancel()
		close(s.fini)
	}()
	return s, nil
}

// Arreter interrompt le scan ; le résultat partiel reste consultable.
func (s *Scan) Arreter() { s.arreter() }

// Attendre bloque jusqu'à la fin du scan.
func (s *Scan) Attendre() { <-s.fini }

// Racine renvoie le chemin scanné.
func (s *Scan) Racine() string { return s.racine }

func (s *Scan) parcourir(ctx context.Context, d *Dossier, chemin string) {
	if ctx.Err() != nil {
		return
	}
	s.courant.Store(&chemin)

	var sousDossiers []*Dossier
	err := listerDossier(chemin, func(e entree) {
		switch {
		case e.Lien:
			d.nLiens++
			s.liens.Add(1)
		case e.Dossier:
			sousDossiers = append(sousDossiers, &Dossier{nom: e.Nom, parent: d})
		default:
			d.nFichiers++
			d.fTaille += e.Taille
			d.fDisque += e.Disque
			d.fModif = max(d.fModif, e.Modif)
			s.fichiers.Add(1)
			s.taille.Add(e.Taille)
			s.disque.Add(e.Disque)
		}
	})
	if err != nil {
		d.illisible = true
		s.erreurs.Add(1)
	}
	d.enfants = sousDossiers
	s.dossiers.Add(int64(len(sousDossiers)))

	for _, c := range sousDossiers {
		cc := filepath.Join(chemin, c.nom)
		select {
		case s.sem <- struct{}{}:
			s.wg.Add(1)
			go func() {
				defer func() {
					<-s.sem
					s.wg.Done()
				}()
				s.parcourir(ctx, c, cc)
			}()
		default:
			// Tous les ouvriers sont occupés : on descend nous-mêmes.
			s.parcourir(ctx, c, cc)
		}
	}
}

// finaliser numérote les dossiers et remonte les totaux vers la racine.
func (s *Scan) finaliser() {
	noeuds := []*Dossier{s.racineD}
	for i := 0; i < len(noeuds); i++ {
		d := noeuds[i]
		d.id = int32(i)
		noeuds = append(noeuds, d.enfants...)
	}
	// Ordre inverse du parcours en largeur : chaque enfant passe avant son parent.
	for i := len(noeuds) - 1; i >= 0; i-- {
		d := noeuds[i]
		d.taille += d.fTaille
		d.disque += d.fDisque
		d.modif = max(d.modif, d.fModif)
		d.fichiers += d.nFichiers
		d.liens += d.nLiens
		if d.illisible {
			d.erreurs++
		}
		if p := d.parent; p != nil {
			p.taille += d.taille
			p.disque += d.disque
			p.modif = max(p.modif, d.modif)
			p.fichiers += d.fichiers
			p.dossiers += d.dossiers + 1
			p.erreurs += d.erreurs
			p.liens += d.liens
		}
	}
	s.mu.Lock()
	s.noeuds = noeuds
	s.mu.Unlock()
}

// Progression renvoie l'avancement (compteurs en direct pendant le scan).
func (s *Scan) Progression() Progression {
	s.mu.RLock()
	etat, duree := s.etat, s.duree
	s.mu.RUnlock()
	if etat == EnCours {
		duree = time.Since(s.debut)
	}
	p := Progression{
		Etat:     etat,
		Racine:   s.racine,
		Fichiers: s.fichiers.Load(),
		Dossiers: s.dossiers.Load(),
		Taille:   s.taille.Load(),
		Disque:   s.disque.Load(),
		Erreurs:  s.erreurs.Load(),
		Liens:    s.liens.Load(),
		DureeMs:  duree.Milliseconds(),
	}
	if etat == EnCours {
		p.Courant = *s.courant.Load()
	}
	return p
}

// ---------- Consultation du résultat ----------

// Entree est une ligne de l'arborescence affichée.
type Entree struct {
	ID       int32  `json:"id"`   // dossier : son id ; « fichiers » et « reste » : l'id du parent
	Type     string `json:"type"` // "dossier", "fichiers", "fichier", "reste"
	Nom      string `json:"nom"`
	Taille   int64  `json:"taille"`
	Disque   int64  `json:"disque"`
	Fichiers int64  `json:"fichiers"`
	Dossiers int64  `json:"dossiers"`
	Erreurs  int64  `json:"erreurs"`
	Liens    int64  `json:"liens"`
	Modif    int64  `json:"modif"`    // fichier le plus récemment modifié, secondes Unix (0 : aucun)
	Ouvrable bool   `json:"ouvrable"` // a un contenu à déplier
	Nombre   int    `json:"nombre"`   // « reste » : nombre d'éléments regroupés
}

func (s *Scan) noeud(id int32) (*Dossier, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.noeuds == nil {
		return nil, ErrEnCours
	}
	if id < 0 || int(id) >= len(s.noeuds) {
		return nil, fmt.Errorf("dossier %d inconnu", id)
	}
	return s.noeuds[id], nil
}

func entreeDossier(d *Dossier) Entree {
	return Entree{
		ID: d.id, Type: "dossier", Nom: d.nom,
		Taille: d.taille, Disque: d.disque, Modif: d.modif,
		Fichiers: d.fichiers, Dossiers: d.dossiers,
		Erreurs: d.erreurs, Liens: d.liens,
		Ouvrable: len(d.enfants) > 0 || d.nFichiers > 0,
	}
}

// RacineEntree renvoie la ligne du dossier scanné.
func (s *Scan) RacineEntree() (Entree, error) {
	d, err := s.noeud(0)
	if err != nil {
		return Entree{}, err
	}
	return entreeDossier(d), nil
}

// limiter garde les limite premières lignes (déjà triées) et regroupe les
// autres dans une ligne « reste ».
func limiter(lignes []Entree, limite int, parent int32) []Entree {
	if limite <= 0 || len(lignes) <= limite {
		return lignes
	}
	reste := Entree{ID: parent, Type: "reste", Nombre: len(lignes) - limite}
	for _, l := range lignes[limite:] {
		reste.Taille += l.Taille
		reste.Disque += l.Disque
		reste.Modif = max(reste.Modif, l.Modif)
		reste.Fichiers += l.Fichiers
		reste.Dossiers += l.Dossiers
	}
	return append(lignes[:limite:limite], reste)
}

// Enfants renvoie les sous-dossiers de id, plus une ligne « fichiers » pour
// les fichiers posés directement dedans, triés selon t.
func (s *Scan) Enfants(id int32, mode string, t Tri, limite int) ([]Entree, error) {
	d, err := s.noeud(id)
	if err != nil {
		return nil, err
	}
	lignes := make([]Entree, 0, len(d.enfants)+1)
	for _, c := range d.enfants {
		lignes = append(lignes, entreeDossier(c))
	}
	if d.nFichiers > 0 {
		lignes = append(lignes, Entree{
			ID: d.id, Type: "fichiers",
			Taille: d.fTaille, Disque: d.fDisque, Modif: d.fModif,
			Fichiers: d.nFichiers, Ouvrable: true,
		})
	}
	t.trier(lignes, mode)
	return limiter(lignes, limite, d.id), nil
}

// Fichiers relit le dossier id sur le disque et renvoie ses fichiers.
// La liste reflète donc le disque au moment du clic, pas au moment du scan.
func (s *Scan) Fichiers(id int32, mode string, t Tri, limite int) ([]Entree, error) {
	if _, err := s.noeud(id); err != nil {
		return nil, err
	}
	chemin, err := s.Chemin(id)
	if err != nil {
		return nil, err
	}
	var lignes []Entree
	err = listerDossier(chemin, func(e entree) {
		if e.Dossier || e.Lien {
			return
		}
		lignes = append(lignes, Entree{ID: id, Type: "fichier", Nom: e.Nom, Taille: e.Taille, Disque: e.Disque, Modif: e.Modif, Fichiers: 1})
	})
	if err != nil {
		return nil, err
	}
	t.trier(lignes, mode)
	return limiter(lignes, limite, id), nil
}

// Chemin reconstruit le chemin complet du dossier id.
func (s *Scan) Chemin(id int32) (string, error) {
	d, err := s.noeud(id)
	if err != nil {
		return "", err
	}
	var noms []string
	for ; d.parent != nil; d = d.parent {
		noms = append(noms, d.nom)
	}
	morceaux := []string{s.racine}
	for i := len(noms) - 1; i >= 0; i-- {
		morceaux = append(morceaux, noms[i])
	}
	return filepath.Join(morceaux...), nil
}
