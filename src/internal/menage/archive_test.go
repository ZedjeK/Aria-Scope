package menage

import (
	"testing"
	"time"
)

func TestNomArchive(t *testing.T) {
	bons := map[string]string{
		"photos":           "photos.zip",
		"  Vacances 2026 ": "Vacances 2026.zip",
		"jeux.ZIP":         "jeux.zip",
		"x. ":              "x.zip",
		"Téléchargements":  "Téléchargements.zip",
		"console.txt":      "console.txt.zip",
	}
	for entree, attendu := range bons {
		if n, err := nomArchive(entree); err != nil || n != attendu {
			t.Errorf("%q : %q, %v (attendu %q)", entree, n, err, attendu)
		}
	}
	for _, mauvais := range []string{"", "   ", ".zip", "a:b", "dossier/x", `c\d`, "quoi?", "con", "NUL.zip", "com3.txt", "a\x01b"} {
		if n, err := nomArchive(mauvais); err == nil {
			t.Errorf("%q accepté : %q", mauvais, n)
		}
	}
}

func TestNomPropose(t *testing.T) {
	jour := time.Date(2026, 10, 23, 9, 0, 0, 0, time.Local)
	d := t.TempDir()
	if n := nomPropose([]string{d + "/Jeux"}, jour); n != "Jeux.zip" {
		t.Errorf("un dossier : %s", n)
	}
	if n := nomPropose([]string{d + "/a", d + "/b"}, jour); n != "Archive 2026-10-23.zip" {
		t.Errorf("plusieurs éléments : %s", n)
	}
}

func TestNomUnique(t *testing.T) {
	pris := map[string]bool{}
	for _, attendu := range []string{"data.pak", "data (2).pak", "data (3).pak"} {
		if n := nomUnique(pris, "data.pak"); n != attendu {
			t.Errorf("%s au lieu de %s", n, attendu)
		}
	}
	if n := nomUnique(pris, "DATA.pak"); n != "DATA (4).pak" {
		t.Errorf("la casse compte comme sous Windows : %s", n)
	}
}
