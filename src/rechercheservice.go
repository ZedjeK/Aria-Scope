package main

import (
	"errors"

	"aria-scope/internal/scan"
)

// SuiviRecherche : l'avancement de la dernière recherche, et l'analyse sur
// laquelle elle porte (Gen 0 : aucune recherche).
type SuiviRecherche struct {
	scan.SuiviRecherche
	Gen      int           `json:"gen"`
	Criteres scan.Criteres `json:"criteres"`
	Dans     int32         `json:"dans"` // dossier de départ (0 : toute l'analyse)
}

// Chercher lance une recherche dans le dossier id de l'analyse gen (0 :
// toute l'analyse). Une recherche en cours est arrêtée.
func (s *ScopeService) Chercher(gen int, id int32, c scan.Criteres) error {
	sc, err := s.courant(gen)
	if err != nil {
		return err
	}
	r, err := sc.Chercher(id, c)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if s.recherche != nil {
		s.recherche.Arreter()
	}
	s.recherche, s.genRecherche = r, gen
	s.criteres, s.dansRecherche = c, id
	s.mu.Unlock()
	return nil
}

// SuiviRecherche renvoie l'avancement de la dernière recherche.
func (s *ScopeService) SuiviRecherche() SuiviRecherche {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.recherche == nil || s.genRecherche != s.gen {
		return SuiviRecherche{}
	}
	return SuiviRecherche{SuiviRecherche: s.recherche.Suivi(), Gen: s.genRecherche, Criteres: s.criteres, Dans: s.dansRecherche}
}

// ResultatsRecherche renvoie les résultats de la dernière recherche sur
// l'analyse gen, triés.
func (s *ScopeService) ResultatsRecherche(gen int, mode string, t scan.Tri) ([]scan.Trouve, error) {
	s.mu.Lock()
	r, g := s.recherche, s.genRecherche
	s.mu.Unlock()
	if r == nil || g != gen {
		return nil, errors.New("aucune recherche sur cette analyse")
	}
	return r.Resultats(mode, t)
}

// ArreterRecherche interrompt la recherche en cours.
func (s *ScopeService) ArreterRecherche() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.recherche != nil {
		s.recherche.Arreter()
	}
}

// Familles : les familles de types que la recherche sait filtrer.
func (s *ScopeService) Familles() []string { return scan.NomsFamilles() }
