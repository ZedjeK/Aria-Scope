package scan

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTrancheDe(t *testing.T) {
	ref := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC).Unix()
	jour := int64(86400)
	cas := []struct {
		modif int64
		nom   string
	}{
		{ref, "Moins d'un mois"},
		{ref + 10*jour, "Moins d'un mois"}, // dans le futur
		{ref - 29*jour, "Moins d'un mois"},
		{ref - 30*jour, "1 à 6 mois"},
		{ref - 200*jour, "6 mois à 1 an"},
		{ref - 400*jour, "1 à 2 ans"},
		{ref - 1000*jour, "2 à 5 ans"},
		{ref - 4000*jour, "Plus de 5 ans"},
	}
	for _, c := range cas {
		if got := tranchesAge[trancheDe(c.modif, ref)].nom; got != c.nom {
			t.Errorf("il y a %d jours : %q, attendu %q", (ref-c.modif)/jour, got, c.nom)
		}
	}
	if trancheDe(0, ref) != trancheInconnue {
		t.Error("un fichier sans date doit aller dans « Date inconnue »")
	}
}

func TestAges(t *testing.T) {
	r := arbre(t)
	maintenant := time.Now()
	vieux := maintenant.AddDate(-7, 0, 0)
	for _, f := range []string{"gros/g1", "gros/g2"} { // 8000 octets vieux de 7 ans
		if err := os.Chtimes(filepath.Join(r, f), vieux, vieux); err != nil {
			t.Fatal(err)
		}
	}
	s := scanner(t, r)

	ages, err := s.Ages(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ages) != len(tranchesAge) {
		t.Fatalf("%d tranches : %+v", len(ages), ages)
	}
	if a := ages[0]; a.Fichiers != 3 || a.Taille != 2110 {
		t.Errorf("récents : %+v", a)
	}
	if a := ages[len(ages)-1]; a.Nom != "Plus de 5 ans" || a.Fichiers != 2 || a.Taille != 8000 {
		t.Errorf("anciens : %+v", a)
	}

	// Une branche relue à la demande, puis servie depuis le cache
	enf, _ := s.Enfants(0, ModeTaille, Tri{}, 0)
	gros := enf[0].ID
	for range 2 {
		ages, err = s.Ages(gros)
		if err != nil {
			t.Fatal(err)
		}
		if ages[0].Fichiers != 1 || ages[len(ages)-1].Fichiers != 2 {
			t.Fatalf("branche gros : %+v", ages)
		}
	}
	types, _ := s.Types(gros, ModeTaille)
	if len(types) != 1 || types[0].Fichiers != 3 {
		t.Fatalf("types de gros (même cache) : %+v", types)
	}
}
