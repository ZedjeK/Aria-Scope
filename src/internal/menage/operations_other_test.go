//go:build !windows

package menage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aria-scope/internal/scan"
)

func ecrire(t *testing.T, chemin string, n int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(chemin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chemin, make([]byte, n), 0o644); err != nil {
		t.Fatal(err)
	}
}

// bac : un dossier à nettoyer, avec un sous-dossier « protégé » par des règles de test.
func bac(t *testing.T) (racine string, s *scan.Scan) {
	d := t.TempDir()
	racine = filepath.Join(d, "disque")
	ecrire(t, filepath.Join(racine, "Téléchargements", "film.mkv"), 5000)
	ecrire(t, filepath.Join(racine, "Téléchargements", "setup.exe"), 300)
	ecrire(t, filepath.Join(racine, "Jeux", "vieux", "data.pak"), 2000)
	ecrire(t, filepath.Join(racine, "Systeme", "noyau.sys"), 900)
	DossierCorbeille = filepath.Join(d, "corbeille")
	ancien := Protections
	Protections = Regles{Arbres: []string{filepath.Join(racine, "Systeme")}}
	t.Cleanup(func() { DossierCorbeille = ""; Protections = ancien })

	s, err := scan.Lancer(racine)
	if err != nil {
		t.Fatal(err)
	}
	s.Attendre()
	return racine, s
}

func cibleDossier(t *testing.T, s *scan.Scan, parent int32, nom string) scan.Cible {
	t.Helper()
	enf, err := s.Enfants(parent, scan.ModeTaille, scan.Tri{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range enf {
		if e.Type == "dossier" && e.Nom == nom {
			return scan.Cible{Type: "dossier", ID: e.ID, Nom: nom}
		}
	}
	t.Fatalf("dossier %s introuvable", nom)
	return scan.Cible{}
}

func TestCorbeilleDeBoutEnBout(t *testing.T) {
	racine, s := bac(t)
	telech := cibleDossier(t, s, 0, "Téléchargements")
	jeux := cibleDossier(t, s, 0, "Jeux")
	vieux := cibleDossier(t, s, jeux.ID, "vieux")
	systeme := cibleDossier(t, s, 0, "Systeme")
	cibles := []scan.Cible{
		vieux,
		{Type: "fichier", ID: telech.ID, Nom: "film.mkv", Taille: 5000},
		systeme, // protégé : jamais touché
	}

	// Aperçu : rien ne bouge
	a, err := ApercuCorbeille(s, cibles)
	if err != nil {
		t.Fatal(err)
	}
	if a.Nombre != 2 || a.Taille != 7000 || len(a.Proteges) != 1 || a.Elements[0].Taille != 5000 {
		t.Fatalf("aperçu : %+v", a)
	}
	if _, err := os.Stat(filepath.Join(racine, "Jeux", "vieux")); err != nil {
		t.Fatal("l'aperçu a touché au disque")
	}

	b, err := EnvoyerCorbeille(s, cibles, 0)
	if err != nil {
		t.Fatal(err)
	}
	if b.Retires != 2 || b.Taille != 7000 || b.Erreur != "" || b.Annule {
		t.Fatalf("bilan : %+v", b)
	}
	if len(b.Restants) != 1 || !strings.Contains(b.Restants[0].Raison, "système") {
		t.Fatalf("restants : %+v", b.Restants)
	}
	for _, p := range []string{"Jeux/vieux", "Téléchargements/film.mkv"} {
		if _, err := os.Lstat(filepath.Join(racine, p)); !os.IsNotExist(err) {
			t.Errorf("%s est toujours là", p)
		}
	}
	if _, err := os.Stat(filepath.Join(racine, "Systeme", "noyau.sys")); err != nil {
		t.Fatal("un élément protégé a été touché !")
	}
	elements, _ := os.ReadDir(DossierCorbeille)
	if len(elements) != 2 {
		t.Errorf("%d éléments dans la corbeille", len(elements))
	}

	// L'analyse suit, sans relance
	r, _ := s.RacineEntree()
	if r.Taille != 1200 || r.Fichiers != 2 {
		t.Errorf("racine : %+v", r)
	}
}

func TestCorbeilleRefuse(t *testing.T) {
	_, s := bac(t)
	telech := cibleDossier(t, s, 0, "Téléchargements")
	if _, err := ApercuCorbeille(s, []scan.Cible{{Type: "fichier", ID: telech.ID, Nom: "../Jeux"}}); err == nil {
		t.Error("un nom de fichier en forme de chemin a été accepté")
	}
	// Tout protégé : rien n'est envoyé
	systeme := cibleDossier(t, s, 0, "Systeme")
	b, err := EnvoyerCorbeille(s, []scan.Cible{systeme}, 0)
	if err != nil || b.Retires != 0 || len(b.Restants) != 1 {
		t.Fatalf("bilan : %+v, %v", b, err)
	}
	if _, err := os.ReadDir(DossierCorbeille); err == nil {
		t.Error("la corbeille ne devrait même pas avoir été ouverte")
	}
}

func TestListesJamaisNulles(t *testing.T) {
	_, s := bac(t)
	jeux := cibleDossier(t, s, 0, "Jeux")
	a, err := ApercuCorbeille(s, []scan.Cible{jeux})
	if err != nil || a.Proteges == nil || a.Amovibles == nil || a.Elements == nil {
		t.Fatalf("aperçu : %+v, %v", a, err)
	}
	b, err := EnvoyerCorbeille(s, []scan.Cible{jeux}, 0)
	if err != nil || b.Restants == nil {
		t.Fatalf("bilan : %+v, %v", b, err)
	}
}
