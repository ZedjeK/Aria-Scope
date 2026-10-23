package main

import (
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
