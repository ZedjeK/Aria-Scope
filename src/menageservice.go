package main

import (
	"aria-scope/internal/menage"
	"aria-scope/internal/scan"
)

// ApercuCorbeille récapitule ce qu'enverrait la corbeille, sans rien toucher.
func (s *ScopeService) ApercuCorbeille(gen int, cibles []scan.Cible) (menage.Apercu, error) {
	sc, err := s.courant(gen)
	if err != nil {
		return menage.Apercu{}, err
	}
	return menage.ApercuCorbeille(sc, cibles)
}

// Corbeille envoie les cibles à la corbeille de Windows et met l'analyse à
// jour pour ce qui a vraiment quitté le disque. Les éléments protégés ne
// sont jamais touchés.
func (s *ScopeService) Corbeille(gen int, cibles []scan.Cible) (menage.Bilan, error) {
	sc, err := s.courant(gen)
	if err != nil {
		return menage.Bilan{}, err
	}
	return menage.EnvoyerCorbeille(sc, cibles, s.fenetre())
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
