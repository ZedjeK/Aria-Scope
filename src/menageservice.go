package main

import (
	"time"

	"aria-scope/internal/menage"
	"aria-scope/internal/scan"
)

// Apercu récapitule ce que ferait une opération de ménage (« corbeille »,
// « deplacer », « copier »), sans rien toucher. dest : dossier d'arrivée
// d'un déplacement ou d'une copie.
func (s *ScopeService) Apercu(gen int, op string, cibles []scan.Cible, dest string) (menage.Apercu, error) {
	sc, err := s.courant(gen)
	if err != nil {
		return menage.Apercu{}, err
	}
	return menage.Apercevoir(sc, op, cibles, dest)
}

// Executer réalise l'opération par l'Explorateur de Windows et met l'analyse
// à jour pour ce qui a quitté son emplacement. Les éléments protégés ne sont
// jamais touchés.
func (s *ScopeService) Executer(gen int, op string, cibles []scan.Cible, dest string) (menage.Bilan, error) {
	sc, err := s.courant(gen)
	if err != nil {
		return menage.Bilan{}, err
	}
	return menage.Executer(sc, op, cibles, dest, s.fenetre())
}

// ChoisirDestination ouvre le sélecteur du dossier d'arrivée. Renvoie "" si
// l'utilisateur annule.
func (s *ScopeService) ChoisirDestination(titre string) (string, error) {
	if s.app == nil {
		return "", nil
	}
	return s.app.Dialog.OpenFile().
		SetTitle(titre).
		CanChooseDirectories(true).
		CanChooseFiles(false).
		CanCreateDirectories(true).
		PromptForSingleSelection()
}

// fenetre renvoie le handle natif de la fenêtre d'Aria (0 si indisponible).
func (s *ScopeService) fenetre() uintptr {
	if s.app == nil {
		return 0
	}
	if w, ok := s.app.Window.GetByName("main"); ok && w != nil {
		return uintptr(w.NativeWindow())
	}
	return 0
}

// ApercuArchive récapitule l'archive .zip à créer dans dest. nom : celui
// tapé dans la fenêtre, vide pour en proposer un.
func (s *ScopeService) ApercuArchive(gen int, cibles []scan.Cible, dest, nom string) (menage.Apercu, error) {
	sc, err := s.courant(gen)
	if err != nil {
		return menage.Apercu{}, err
	}
	return menage.ApercevoirArchive(sc, cibles, dest, nom)
}

// Archiver lance l'archivage en arrière-plan ; SuiviArchive donne sa progression.
func (s *ScopeService) Archiver(gen int, cibles []scan.Cible, dest string, opts menage.OptionsArchive) error {
	sc, err := s.courant(gen)
	if err != nil {
		return err
	}
	a, err := menage.Archiver(sc, cibles, dest, opts, s.fenetre())
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.archivage = a
	s.mu.Unlock()
	return nil
}

// SuiviArchive : progression du dernier archivage (phase vide s'il n'y en a pas).
func (s *ScopeService) SuiviArchive() menage.Suivi {
	s.mu.Lock()
	a := s.archivage
	s.mu.Unlock()
	if a == nil {
		return menage.Suivi{}
	}
	return a.Suivi()
}

// AnnulerArchive arrête l'archivage en cours ; rien n'est laissé sur le disque.
func (s *ScopeService) AnnulerArchive() {
	s.mu.Lock()
	a := s.archivage
	s.mu.Unlock()
	if a != nil {
		a.Annuler()
	}
}

// ServiceShutdown (appelé par Wails à la fermeture) : un archivage en cours
// est annulé, pour ne pas laisser d'archive à moitié écrite.
func (s *ScopeService) ServiceShutdown() error {
	s.mu.Lock()
	a := s.archivage
	s.mu.Unlock()
	if a == nil {
		return nil
	}
	a.Annuler()
	select {
	case <-a.Fini():
	case <-time.After(10 * time.Second):
	}
	return nil
}
