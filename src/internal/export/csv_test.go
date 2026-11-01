package export

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aria-scope/internal/scan"
)

func TestLisible(t *testing.T) {
	for n, attendu := range map[int64]string{
		0: "0 o", 512: "512 o", 1024: "1,00 Ko", 1536: "1,50 Ko",
		10 << 20: "10,0 Mo", 1500000: "1,43 Mo", 987 << 30: "987 Go", 3 << 40: "3,00 To",
	} {
		if got := Lisible(n); got != attendu {
			t.Errorf("Lisible(%d) = %q, attendu %q", n, got, attendu)
		}
	}
}

func TestCSVFormat(t *testing.T) {
	var b bytes.Buffer
	c := nouveauCSV(&b, "Nom", "Valeur")
	c.ligne("simple", "1")
	c.ligne(`point;virgule`, `guillemet "x"`)
	c.ligne("=SOMME(A1)", "-12,5")
	c.ligne("@commande", "+33")
	c.ligne("retour\nligne", "")
	s := b.String()
	if !strings.HasPrefix(s, "\uFEFFNom;Valeur\r\n") {
		t.Fatalf("BOM ou en-tête : %q", s[:20])
	}
	for _, attendu := range []string{
		"simple;1\r\n",
		`"point;virgule";"guillemet ""x"""` + "\r\n",
		"'=SOMME(A1);-12,5\r\n", // une formule n'est jamais exécutée ; un nombre négatif reste un nombre
		"'@commande;+33\r\n",
		"\"retour\nligne\";\r\n",
	} {
		if !strings.Contains(s, attendu) {
			t.Errorf("ligne %q absente de\n%s", attendu, s)
		}
	}
}

func TestArbreCSV(t *testing.T) {
	r := t.TempDir()
	ecrire := func(p string, n int) {
		os.MkdirAll(filepath.Dir(filepath.Join(r, p)), 0o755)
		os.WriteFile(filepath.Join(r, p), make([]byte, n), 0o644)
	}
	ecrire("gros/a.bin", 3000)
	ecrire("gros/profond/b.bin", 1000)
	ecrire("petit/c.bin", 1000)
	s, err := scan.Lancer(r)
	if err != nil {
		t.Fatal(err)
	}
	s.Attendre()

	exporter := func(profondeur int) []string {
		var b bytes.Buffer
		err := ArbreCSV(&b, scan.ModeTaille, func(fn func(scan.LigneArbre) error) error {
			return s.ParcourirArbre(0, profondeur, scan.ModeTaille, fn)
		})
		if err != nil {
			t.Fatal(err)
		}
		return strings.Split(strings.TrimSuffix(b.String(), "\r\n"), "\r\n")
	}
	tout := exporter(0)
	if len(tout) != 5 { // en-tête + racine, gros, profond, petit
		t.Fatalf("%d lignes :\n%s", len(tout), strings.Join(tout, "\n"))
	}
	// Ordre de l'arbre : le plus lourd d'abord, en profondeur
	for i, nom := range []string{"gros", "profond", "petit"} {
		if champs := strings.Split(tout[i+2], ";"); champs[2] != nom {
			t.Errorf("ligne %d : %s au lieu de %s", i+2, champs[2], nom)
		}
	}
	gros := strings.Split(tout[2], ";")
	if gros[0] != "1" || gros[1] != filepath.Join(r, "gros") || gros[3] != "4000" || gros[5] != "3,91 Ko" || gros[6] != "80,0" {
		t.Errorf("ligne de « gros » : %v", gros)
	}
	if n := len(exporter(1)); n != 4 {
		t.Errorf("profondeur 1 : %d lignes au lieu de 4", n)
	}
}

func TestEcrireAtomique(t *testing.T) {
	d := t.TempDir()
	chemin := filepath.Join(d, "export.csv")
	if err := Ecrire(chemin, func(w io.Writer) error { _, err := w.Write([]byte("ok")); return err }); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(chemin); string(b) != "ok" {
		t.Fatal("contenu")
	}
	// Une erreur en route : l'ancien fichier reste intact, pas de provisoire.
	err := Ecrire(chemin, func(w io.Writer) error { w.Write([]byte("moitié")); return errors.New("panne") })
	if err == nil {
		t.Fatal("erreur perdue")
	}
	if b, _ := os.ReadFile(chemin); string(b) != "ok" {
		t.Fatal("l'ancien fichier a été abîmé")
	}
	if _, err := os.Stat(chemin + ".part"); err == nil {
		t.Fatal("fichier provisoire resté")
	}
}

func TestNomFichier(t *testing.T) {
	if n := NomFichier("Gros fichiers", "csv", time.Date(2026, 11, 1, 9, 0, 0, 0, time.Local)); n != "Aria Scope - Gros fichiers - 2026-11-01.csv" {
		t.Error(n)
	}
}
