package scan

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDateDeModification(t *testing.T) {
	r := arbre(t)
	dater := func(rel string, quand time.Time) {
		t.Helper()
		if err := os.Chtimes(filepath.Join(r, rel), quand, quand); err != nil {
			t.Fatal(err)
		}
	}
	ancien := time.Date(2020, 1, 1, 12, 0, 0, 0, time.UTC)
	for _, f := range []string{"a.bin", "gros/g1", "gros/g2", "petit/p"} {
		dater(f, ancien)
	}
	// Le fichier le plus récent est tout au fond de « gros ».
	recent := time.Date(2026, 10, 20, 18, 30, 0, 0, time.UTC)
	dater("gros/profond/x/y/z", recent)

	s := scanner(t, r)
	racine, _ := s.RacineEntree()
	if racine.Modif != recent.Unix() {
		t.Fatalf("racine : %v, attendu %v (le plus récent du sous-arbre)", time.Unix(racine.Modif, 0), recent)
	}

	enf, _ := s.Enfants(0, ModeTaille, Tri{Colonne: TriModif}, 0)
	if got := noms(enf); got != "[gros [fichiers] petit vide]" {
		t.Fatalf("tri par date : %s", got)
	}
	if enf[0].Modif != recent.Unix() || enf[1].Modif != ancien.Unix() {
		t.Errorf("dates : gros %d, fichiers %d", enf[0].Modif, enf[1].Modif)
	}
	if vide := enf[3]; vide.Modif != 0 {
		t.Errorf("un dossier vide n'a pas de date : %d", vide.Modif)
	}

	fich, _ := s.Fichiers(enf[0].ID, ModeTaille, Tri{}, 0)
	if fich[0].Modif != ancien.Unix() {
		t.Errorf("date du fichier g1 : %d", fich[0].Modif)
	}
}
