package scan

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestGrosFichiers(t *testing.T) {
	s := scanner(t, arbre(t))
	g, err := s.GrosFichiers(ModeTaille, Tri{})
	if err != nil {
		t.Fatal(err)
	}
	var noms []string
	for _, f := range g {
		noms = append(noms, f.Nom)
	}
	if fmt.Sprint(noms) != "[g1 g2 z a.bin p]" {
		t.Fatalf("ordre = %v", noms)
	}
	z := g[2]
	if z.Dossier != filepath.Join(s.Racine(), "gros", "profond", "x", "y") || z.Taille != 2000 {
		t.Fatalf("z = %+v", z)
	}
	// L'id renvoie bien au dossier : on peut relire ses fichiers.
	f, err := s.Fichiers(z.ID, ModeTaille, Tri{}, 0)
	if err != nil || len(f) != 1 || f[0].Nom != "z" {
		t.Fatalf("fichiers du dossier de z : %+v, %v", f, err)
	}

	g, _ = s.GrosFichiers(ModeTaille, Tri{Colonne: TriNom})
	if g[0].Nom != "a.bin" {
		t.Errorf("par nom : %s en premier", g[0].Nom)
	}
}

func TestPalmaresGardeLesPlusGros(t *testing.T) {
	r := t.TempDir()
	// 1 200 fichiers de 1 à 1 200 octets répartis dans 12 dossiers
	for i := 1; i <= 1200; i++ {
		ecrire(t, filepath.Join(r, fmt.Sprintf("d%02d", i%12), fmt.Sprintf("f%04d", i)), i)
	}
	s := scanner(t, r)
	g, err := s.GrosFichiers(ModeTaille, Tri{})
	if err != nil {
		t.Fatal(err)
	}
	if len(g) != NombreGros {
		t.Fatalf("%d retenus, attendu %d", len(g), NombreGros)
	}
	if g[0].Taille != 1200 || g[len(g)-1].Taille != 201 {
		t.Fatalf("du %d au %d, attendu du 1200 au 201", g[0].Taille, g[len(g)-1].Taille)
	}
}
