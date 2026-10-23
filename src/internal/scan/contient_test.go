package scan

import (
	"path/filepath"
	"testing"
)

func TestContient(t *testing.T) {
	r := arbre(t)
	s := scanner(t, r)
	if !s.Contient(r) || !s.Contient(filepath.Join(r, "gros", "nouveau")) {
		t.Error("la racine et ce qu'elle contient doivent être couverts")
	}
	if s.Contient(filepath.Dir(r)) || s.Contient(r+"-voisin") {
		t.Error("un dossier voisin n'est pas couvert")
	}

	a, b := t.TempDir(), t.TempDir()
	pc, err := LancerPC("Ce PC", []string{a}, Filtres{})
	if err != nil {
		t.Fatal(err)
	}
	pc.Attendre()
	if !pc.Contient(filepath.Join(a, "x")) || pc.Contient(filepath.Join(b, "x")) {
		t.Error("« Ce PC » couvre ses disques, et seulement eux")
	}
}
