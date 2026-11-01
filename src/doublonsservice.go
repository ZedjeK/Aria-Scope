package main

import (
	"errors"

	"aria-scope/internal/scan"
)

// SuiviDoublons : l'avancement de la dernière recherche de doublons, et ce
// sur quoi elle porte (Gen 0 : aucune).
type SuiviDoublons struct {
	scan.SuiviDoublons
	Gen       int   `json:"gen"`
	TailleMin int64 `json:"tailleMin"`
	Dans      int32 `json:"dans"`
}

// ChercherDoublons lance la recherche des fichiers identiques dans le dossier
// id de l'analyse gen (0 : toute l'analyse). Une recherche en cours est arrêtée.
func (s *ScopeService) ChercherDoublons(gen int, id int32, tailleMin int64) error {
	sc, err := s.courant(gen)
	if err != nil {
		return err
	}
	d, err := sc.ChercherDoublons(id, tailleMin)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if s.doublons != nil {
		s.doublons.Arreter()
	}
	s.doublons, s.genDoublons, s.minDoublons, s.dansDoublons = d, gen, tailleMin, id
	s.mu.Unlock()
	return nil
}

// SuiviDoublons renvoie l'avancement de la dernière recherche de doublons.
func (s *ScopeService) SuiviDoublons() SuiviDoublons {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.doublons == nil || s.genDoublons != s.gen {
		return SuiviDoublons{}
	}
	return SuiviDoublons{SuiviDoublons: s.doublons.Suivi(), Gen: s.genDoublons, TailleMin: s.minDoublons, Dans: s.dansDoublons}
}

// GroupesDoublons renvoie les groupes de fichiers identiques de l'analyse gen.
func (s *ScopeService) GroupesDoublons(gen int) ([]scan.GroupeDoublons, error) {
	s.mu.Lock()
	d, g := s.doublons, s.genDoublons
	s.mu.Unlock()
	if d == nil || g != gen {
		return nil, errors.New("aucune recherche de doublons sur cette analyse")
	}
	return d.Groupes()
}

// ArreterDoublons interrompt la recherche de doublons en cours.
func (s *ScopeService) ArreterDoublons() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.doublons != nil {
		s.doublons.Arreter()
	}
}
