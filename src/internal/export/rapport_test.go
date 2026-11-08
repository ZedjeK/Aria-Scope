package export

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"aria-scope/internal/scan"
)

func terrainRapport(t *testing.T) string {
	t.Helper()
	r := t.TempDir()
	ecrire := func(p string, data []byte) {
		os.MkdirAll(filepath.Dir(filepath.Join(r, p)), 0o755)
		if err := os.WriteFile(filepath.Join(r, p), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	film := bytes.Repeat([]byte("film"), 50000)
	ecrire("Vidéos/vacances.mkv", film)
	ecrire("Sauvegarde/vacances.mkv", film)
	ecrire("Photos/2019/IMG_0001.jpg", make([]byte, 30000))
	// Un nom piégé : il doit rester du texte dans le rapport.
	if piegesPossibles() {
		ecrire(`Pièges/<script>alert("x")</script>.txt`, make([]byte, 100))
		ecrire(`Pièges/<img src=x onerror=alert(1)>/a.txt`, make([]byte, 10))
	}
	return r
}

// piegesPossibles : Windows interdit < > " dans un nom de fichier, les noms piégés ne peuvent exister que
// sur les autres systèmes. Ces contrôles d'échappement tournent donc sous Linux (voir tests.yml).
func piegesPossibles() bool { return runtime.GOOS != "windows" }

func TestRapportHTML(t *testing.T) {
	r := terrainRapport(t)
	s, err := scan.Lancer(r)
	if err != nil {
		t.Fatal(err)
	}
	s.Attendre()
	d, _ := s.ChercherDoublons(0, 1)
	d.Attendre()

	var b bytes.Buffer
	quand := time.Date(2026, 11, 6, 10, 30, 0, 0, time.Local)
	if err := RapportHTML(&b, s, d, Options{Mode: scan.ModeTaille}, "0.7.0", quand); err != nil {
		t.Fatal(err)
	}
	h := b.String()
	for _, attendu := range []string{
		"<!doctype html>", "Rapport du <strong>06/11/2026 à 10:30</strong>", "Aria Scope 0.7.0",
		"Plus gros dossiers", "Vidéos", "Types de fichiers", "Âge des fichiers", "vacances.mkv",
		"Doublons", "copies en trop", `<svg class="carte"`, "@media print", "window.print()",
	} {
		if !strings.Contains(h, attendu) {
			t.Errorf("%q absent du rapport", attendu)
		}
	}
	// Les noms piégés sont échappés, jamais interprétés.
	if strings.Contains(h, "<script>alert") || strings.Contains(h, "<img src=x") {
		t.Fatal("un nom de fichier a été injecté tel quel dans le HTML !")
	}
	if piegesPossibles() && !strings.Contains(h, "&lt;script&gt;") && !strings.Contains(h, "&lt;img src=x") {
		t.Error("les noms piégés devraient apparaître, échappés")
	}
	// Aucune ressource externe : le rapport se suffit à lui-même.
	for _, externe := range []string{"http://", "https://", "<link", "src=\"http"} {
		if strings.Contains(h, externe) {
			t.Errorf("ressource externe dans le rapport : %s", externe)
		}
	}
	// Les largeurs CSS gardent un point décimal (une virgule les casserait).
	if strings.Contains(h, "width:") && strings.Contains(h, ",00%") {
		t.Error("virgule décimale dans le CSS")
	}
}

func TestRapportSousDossier(t *testing.T) {
	r := terrainRapport(t)
	s, _ := scan.Lancer(r)
	s.Attendre()
	enf, _ := s.Enfants(0, scan.ModeTaille, scan.Tri{}, 0)
	var id int32
	for _, e := range enf {
		if e.Nom == "Photos" {
			id = e.ID
		}
	}
	var b bytes.Buffer
	if err := RapportHTML(&b, s, nil, Options{Mode: scan.ModeTaille, Dans: id}, "dev", time.Now()); err != nil {
		t.Fatal(err)
	}
	h := b.String()
	if !strings.Contains(h, "analyse de") || !strings.Contains(h, "IMG_0001.jpg") {
		t.Error("rapport partiel incomplet")
	}
	if strings.Contains(h, "vacances.mkv") {
		t.Error("un fichier hors du dossier figure dans le rapport")
	}
	if strings.Contains(h, "Doublons récupérables") {
		t.Error("pas de doublons cherchés : pas de section")
	}
}

func TestSquarifier(t *testing.T) {
	rs := squarifier([]float64{6, 6, 4, 3, 2, 2, 1}, 0, 0, 600, 400)
	var aire float64
	for _, r := range rs {
		aire += r.w * r.h
		if r.x < -1e-9 || r.y < -1e-9 || r.x+r.w > 600+1e-6 || r.y+r.h > 400+1e-6 {
			t.Errorf("rectangle hors cadre : %+v", r)
		}
	}
	if d := aire - 600*400; d > 1e-3 || d < -1e-3 {
		t.Errorf("aire totale %f au lieu de %d", aire, 600*400)
	}
	// Proportionnel : le premier fait 6/24 de l'aire
	if a := rs[0].w * rs[0].h; a < 0.249*240000 || a > 0.251*240000 {
		t.Errorf("aire du premier : %f", a)
	}
}
