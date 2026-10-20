package scan

import (
	"fmt"
	"testing"
)

func noms(lignes []Entree) string {
	var out []string
	for _, l := range lignes {
		if l.Type == "fichiers" {
			out = append(out, "[fichiers]")
			continue
		}
		out = append(out, l.Nom)
	}
	return fmt.Sprint(out)
}

func TestTriColonnes(t *testing.T) {
	// racine : a.bin 100 ; gros/ 3 fichiers 10000 (3 sous-dossiers) ; petit/ 10 ; vide/
	s := scanner(t, arbre(t))

	cas := []struct {
		tri     Tri
		attendu string
	}{
		{Tri{}, "[gros [fichiers] petit vide]"},
		{Tri{Inverse: true}, "[vide petit [fichiers] gros]"},
		{Tri{Colonne: TriNom}, "[gros petit vide [fichiers]]"},
		{Tri{Colonne: TriNom, Inverse: true}, "[[fichiers] vide petit gros]"},
		// gros : 3 fichiers ; [fichiers] et petit : 1 chacun, départagés par la taille
		{Tri{Colonne: TriFichiers}, "[gros [fichiers] petit vide]"},
		{Tri{Colonne: TriDossiers}, "[gros [fichiers] petit vide]"},
	}
	for _, c := range cas {
		enf, err := s.Enfants(0, ModeTaille, c.tri, 0)
		if err != nil {
			t.Fatal(err)
		}
		if got := noms(enf); got != c.attendu {
			t.Errorf("tri %+v : %s, attendu %s", c.tri, got, c.attendu)
		}
	}
}

func TestTriPuisLimite(t *testing.T) {
	r := t.TempDir()
	for i := 1; i <= 10; i++ {
		ecrire(t, fmt.Sprintf("%s/d%02d/f", r, i), i*100)
	}
	s := scanner(t, r)
	// Par nom, les 3 premiers sont d01 à d03, les 7 autres regroupés.
	enf, err := s.Enfants(0, ModeTaille, Tri{Colonne: TriNom}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if got := noms(enf[:3]); got != "[d01 d02 d03]" {
		t.Fatalf("par nom : %s", got)
	}
	if r := enf[3]; r.Type != "reste" || r.Nombre != 7 || r.Taille != 4900 {
		t.Fatalf("reste = %+v", r)
	}
}

func TestTriFichiersRelus(t *testing.T) {
	s := scanner(t, arbre(t))
	enf, _ := s.Enfants(0, ModeTaille, Tri{}, 0)
	fich, err := s.Fichiers(enf[0].ID, ModeTaille, Tri{Colonne: TriNom, Inverse: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := noms(fich); got != "[g2 g1]" {
		t.Fatalf("fichiers par nom inversé : %s", got)
	}
}
