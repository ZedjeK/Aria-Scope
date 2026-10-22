package scan

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRetirer(t *testing.T) {
	// racine : a.bin 100 ; gros/ (g1 5000, g2 3000, profond/x/y/z 2000) ; petit/ p 10 ; vide/
	r := arbre(t)
	s := scanner(t, r)
	enf, _ := s.Enfants(0, ModeTaille, Tri{}, 0)
	gros := enf[0]
	sousGros, _ := s.Enfants(gros.ID, ModeTaille, Tri{}, 0)
	var profond Entree
	for _, e := range sousGros {
		if e.Nom == "profond" {
			profond = e
		}
	}

	cibles := []Cible{
		{Type: "dossier", ID: profond.ID},
		{Type: "fichier", ID: 0, Nom: "a.bin", Taille: 100},
		{Type: "fichier", ID: gros.ID, Nom: "g2"}, // restera sur le disque
	}
	retraits, err := s.PreparerRetrait(cibles)
	if err != nil {
		t.Fatal(err)
	}
	if retraits[0].Taille != 2000 || retraits[0].Chemin != filepath.Join(r, "gros", "profond") || retraits[1].Taille != 100 {
		t.Fatalf("préparation : %+v", retraits)
	}

	// Seuls profond et a.bin quittent vraiment le disque.
	if err := os.RemoveAll(filepath.Join(r, "gros", "profond")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(r, "a.bin")); err != nil {
		t.Fatal(err)
	}
	faits := s.Retirer(retraits)
	if len(faits) != 2 {
		t.Fatalf("%d retraits appliqués, attendu 2", len(faits))
	}

	racine, _ := s.RacineEntree()
	if racine.Taille != 8010 || racine.Fichiers != 3 || racine.Dossiers != 3 {
		t.Errorf("racine après ménage : %+v", racine)
	}
	if p := s.Progression(); p.Taille != 8010 || p.Fichiers != 3 || p.Dossiers != 3 {
		t.Errorf("progression : %+v", p)
	}
	sous, _ := s.Enfants(gros.ID, ModeTaille, Tri{}, 0)
	if len(sous) != 1 || sous[0].Type != "fichiers" || sous[0].Taille != 8000 {
		t.Errorf("gros après ménage : %+v", sous)
	}
	if _, err := s.Enfants(profond.ID, ModeTaille, Tri{}, 0); !errors.Is(err, ErrRetire) {
		t.Errorf("profond devrait être retiré : %v", err)
	}
	enf, _ = s.Enfants(0, ModeTaille, Tri{}, 0)
	for _, e := range enf {
		if e.Type == "fichiers" {
			t.Errorf("a.bin était le seul fichier de la racine : %+v", e)
		}
	}

	// Statistiques et gros fichiers suivent
	g, _ := s.GrosFichiers(ModeTaille, Tri{})
	var noms []string
	for _, f := range g {
		noms = append(noms, f.Nom)
	}
	if len(g) != 3 || g[0].Nom != "g1" {
		t.Errorf("gros fichiers : %v", noms)
	}
	var n, taille int64
	types, _ := s.Types(0, ModeTaille)
	for _, f := range types {
		n += f.Fichiers
		taille += f.Taille
	}
	if n != 3 || taille != 8010 {
		t.Errorf("types : %d fichiers, %d octets", n, taille)
	}
	ages, _ := s.Ages(0)
	if ages[0].Fichiers != 3 {
		t.Errorf("âges : %+v", ages[0])
	}
	c, _ := s.Carte(0, ModeTaille)
	verifierSommes(t, c)

	// Un second passage ne change plus rien
	if len(s.Retirer(retraits)) != 0 {
		t.Error("un retrait ne doit s'appliquer qu'une fois")
	}
}

func TestPreparerRetraitRefuse(t *testing.T) {
	s := scanner(t, arbre(t))
	for _, c := range []Cible{
		{Type: "dossier", ID: 0},
		{Type: "fichier", ID: 0, Nom: "../ailleurs"},
		{Type: "fichier", ID: 0, Nom: "absent.txt"},
		{Type: "bizarre", ID: 0},
	} {
		if _, err := s.PreparerRetrait([]Cible{c}); err == nil {
			t.Errorf("%+v accepté", c)
		}
	}
}
