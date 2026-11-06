package main

import (
	"errors"
	"io"
	"time"

	"aria-scope/internal/export"
)

// choisirFichier ouvre « Enregistrer sous » ; "" si l'utilisateur annule.
func (s *ScopeService) choisirFichier(titre, nom, filtre, motif string) (string, error) {
	if s.app == nil {
		return "", errors.New("application non prête")
	}
	return s.app.Dialog.SaveFile().
		SetMessage(titre).
		SetFilename(nom).
		AddFilter(filtre, motif).
		CanCreateDirectories(true).
		PromptForSingleSelection()
}

// ExporterCSV enregistre une vue de l'analyse gen en CSV, à l'endroit choisi
// par l'utilisateur. Renvoie le chemin du fichier, ou "" s'il a annulé.
func (s *ScopeService) ExporterCSV(gen int, o export.Options) (string, error) {
	sc, err := s.courant(gen)
	if err != nil {
		return "", err
	}
	titre, ok := export.Titres[o.Vue]
	if !ok {
		return "", errors.New("vue inconnue : " + o.Vue)
	}
	s.mu.Lock()
	r, d := s.recherche, s.doublons
	if s.genRecherche != gen {
		r = nil
	}
	if s.genDoublons != gen {
		d = nil
	}
	s.mu.Unlock()

	chemin, err := s.choisirFichier("Exporter en CSV", export.NomFichier(titre, "csv", time.Now()), "Fichiers CSV (Excel)", "*.csv")
	if err != nil || chemin == "" {
		return "", err
	}
	err = export.Ecrire(chemin, func(w io.Writer) error { return export.VueCSV(w, sc, r, d, o) })
	if err != nil {
		return "", err
	}
	return chemin, nil
}

// ExporterHTML enregistre le rapport HTML du dossier o.Dans de l'analyse gen
// (0 : toute l'analyse). Les doublons y figurent s'ils ont été cherchés.
func (s *ScopeService) ExporterHTML(gen int, o export.Options) (string, error) {
	sc, err := s.courant(gen)
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	d := s.doublons
	if s.genDoublons != gen {
		d = nil
	}
	s.mu.Unlock()
	chemin, err := s.choisirFichier("Enregistrer le rapport", export.NomFichier("Rapport", "html", time.Now()), "Page web (rapport)", "*.html")
	if err != nil || chemin == "" {
		return "", err
	}
	err = export.Ecrire(chemin, func(w io.Writer) error { return export.RapportHTML(w, sc, d, o, version, time.Now()) })
	if err != nil {
		return "", err
	}
	return chemin, nil
}

// OuvrirFichier ouvre un fichier exporté avec le programme associé
// (le navigateur pour un rapport).
func (s *ScopeService) OuvrirFichier(chemin string) error {
	if s.app == nil {
		return errors.New("application non prête")
	}
	return s.app.Browser.OpenFile(chemin)
}
