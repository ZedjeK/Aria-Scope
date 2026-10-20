package scan

import (
	"fmt"
	"path/filepath"
	"testing"
)

// somme vérifie que chaque bloc vaut au moins la somme de ses enfants.
func verifierSommes(t *testing.T, b Bloc) int {
	t.Helper()
	n := 1
	var somme int64
	for _, e := range b.Enfants {
		somme += e.Valeur
		n += verifierSommes(t, e)
	}
	if len(b.Enfants) > 0 && somme != b.Valeur {
		t.Errorf("%s (%s) : %d, enfants %d", b.Nom, b.Type, b.Valeur, somme)
	}
	return n
}

func TestCarte(t *testing.T) {
	s := scanner(t, arbre(t))
	c, err := s.Carte(0, ModeTaille)
	if err != nil {
		t.Fatal(err)
	}
	if c.Valeur != 10110 {
		t.Fatalf("racine = %d", c.Valeur)
	}
	verifierSommes(t, c)

	// gros (10000), fichiers de la racine (100), petit (10) ; « vide » ne se dessine pas
	var ordre []string
	for _, e := range c.Enfants {
		ordre = append(ordre, e.Type+":"+e.Nom)
	}
	if fmt.Sprint(ordre) != "[dossier:gros fichiers: dossier:petit]" {
		t.Fatalf("enfants = %v", ordre)
	}
	// gros -> fichiers (8000) + profond (2000) -> x -> y : 4 niveaux sous la racine
	gros := c.Enfants[0]
	if len(gros.Enfants) != 2 || gros.Enfants[0].Type != "fichiers" || gros.Enfants[1].Nom != "profond" {
		t.Fatalf("gros = %+v", gros.Enfants)
	}
	y := gros.Enfants[1].Enfants[0].Enfants[0]
	if y.Nom != "y" || len(y.Enfants) != 0 || !y.Ouvrable {
		t.Fatalf("niveau 4 : %+v (on s'arrête là, mais on peut y plonger)", y)
	}

	// Plonger dans « gros » : la carte repart de lui
	sous, err := s.Carte(gros.ID, ModeTaille)
	if err != nil || sous.Nom != "gros" || sous.Valeur != 10000 {
		t.Fatalf("sous-carte : %+v, %v", sous, err)
	}
}

func TestCarteRegroupeLesMiettes(t *testing.T) {
	r := t.TempDir()
	ecrire(t, filepath.Join(r, "enorme", "f"), 1_000_000)
	for i := range 100 {
		ecrire(t, filepath.Join(r, fmt.Sprintf("miette%03d", i), "f"), 10) // 0,001 % chacune
	}
	s := scanner(t, r)
	c, err := s.Carte(0, ModeTaille)
	if err != nil {
		t.Fatal(err)
	}
	verifierSommes(t, c)
	if len(c.Enfants) != 2 {
		t.Fatalf("%d blocs, attendu « enorme » + le reste", len(c.Enfants))
	}
	if r := c.Enfants[1]; r.Type != "reste" || r.Nombre != 100 || r.Valeur != 1000 {
		t.Fatalf("reste = %+v", r)
	}
}

func TestFil(t *testing.T) {
	s := scanner(t, arbre(t))
	c, _ := s.Carte(0, ModeTaille)
	x := c.Enfants[0].Enfants[1].Enfants[0] // gros/profond/x
	fil, err := s.Fil(x.ID)
	if err != nil {
		t.Fatal(err)
	}
	var noms []string
	for _, e := range fil {
		noms = append(noms, e.Nom)
	}
	if len(noms) != 4 || noms[0] != s.Racine() || fmt.Sprint(noms[1:]) != "[gros profond x]" {
		t.Fatalf("fil = %v", noms)
	}
}
