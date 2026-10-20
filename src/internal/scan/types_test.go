package scan

import (
	"path/filepath"
	"testing"
)

func TestExtension(t *testing.T) {
	cas := map[string]string{
		"film.MKV": "mkv", "archive.tar.gz": "gz", ".gitignore": "", "LISEZMOI": "",
		"photo.": "", "bizarre.extension-trop-longue": "", "deux mots.a b": "",
	}
	for nom, attendu := range cas {
		if got := extension(nom); got != attendu {
			t.Errorf("extension(%q) = %q, attendu %q", nom, got, attendu)
		}
	}
}

func TestTypes(t *testing.T) {
	r := t.TempDir()
	ecrire(t, filepath.Join(r, "films", "a.mkv"), 5000)
	ecrire(t, filepath.Join(r, "films", "b.MP4"), 3000)
	ecrire(t, filepath.Join(r, "films", "bonus", "c.mkv"), 1000)
	ecrire(t, filepath.Join(r, "docs", "cv.pdf"), 400)
	ecrire(t, filepath.Join(r, "docs", "LISEZMOI"), 100)
	ecrire(t, filepath.Join(r, "figurine.ctb"), 700)
	ecrire(t, filepath.Join(r, "truc.xyz"), 50)
	s := scanner(t, r)

	fam, err := s.Types(0, ModeTaille)
	if err != nil {
		t.Fatal(err)
	}
	attendu := []struct {
		nom      string
		taille   int64
		fichiers int64
	}{{"Vidéos", 9000, 3}, {"3D et impression 3D", 700, 1}, {"Documents", 400, 1}, {"Sans extension", 100, 1}, {"Autres", 50, 1}}
	if len(fam) != len(attendu) {
		t.Fatalf("%d familles : %+v", len(fam), fam)
	}
	for i, a := range attendu {
		if f := fam[i]; f.Nom != a.nom || f.Taille != a.taille || f.Fichiers != a.fichiers {
			t.Errorf("famille %d = %s %d (%d fichiers), attendu %s %d (%d)", i, f.Nom, f.Taille, f.Fichiers, a.nom, a.taille, a.fichiers)
		}
	}
	if ext := fam[0].Extensions; len(ext) != 2 || ext[0].Ext != "mkv" || ext[0].Taille != 6000 || ext[1].Ext != "mp4" {
		t.Errorf("extensions des vidéos : %+v", ext)
	}

	// Pour une branche, relue à la demande : « films » seulement
	enf, _ := s.Enfants(0, ModeTaille, Tri{}, 0)
	films, err := s.Types(enf[0].ID, ModeTaille)
	if err != nil {
		t.Fatal(err)
	}
	if len(films) != 1 || films[0].Taille != 9000 || films[0].Fichiers != 3 {
		t.Fatalf("branche films : %+v", films)
	}
}
