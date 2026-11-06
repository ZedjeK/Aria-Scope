package export

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"
	"time"

	"aria-scope/internal/scan"
)

func TestColonne(t *testing.T) {
	for i, attendu := range map[int]string{0: "A", 25: "Z", 26: "AA", 27: "AB", 701: "ZZ", 702: "AAA"} {
		if got := colonne(i); got != attendu {
			t.Errorf("colonne(%d) = %s, attendu %s", i, got, attendu)
		}
	}
}

func TestNomOnglet(t *testing.T) {
	if n := nomOnglet("a/b:c*d?[e]"); n != "a-b-c-d--e-" {
		t.Error(n)
	}
	if n := nomOnglet(strings.Repeat("é", 40)); len([]rune(n)) != 31 {
		t.Errorf("%d caractères", len([]rune(n)))
	}
}

func TestTexteXML(t *testing.T) {
	if s := texteXML("a\x01b\tc\x7fé\xff"); s != "a�b\tc\x7fé�" {
		t.Errorf("%q", s)
	}
}

func TestDateExcel(t *testing.T) {
	// 1er janvier 2026 à midi, heure locale : 46023,5
	sec := time.Date(2026, 1, 1, 12, 0, 0, 0, time.Local).Unix()
	if c := dateExcel(sec); !c.estNb || c.nombre < 46023.4999 || c.nombre > 46023.5001 {
		t.Errorf("%+v", c)
	}
	if c := dateExcel(0); c.estNb {
		t.Error("date inconnue : cellule vide attendue")
	}
}

// bienForme vérifie que chaque XML du classeur se lit, et renvoie les parts.
func bienForme(t *testing.T, data []byte) map[string]string {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	parts := map[string]string{}
	for _, f := range z.File {
		rc, _ := f.Open()
		b, _ := io.ReadAll(rc)
		rc.Close()
		parts[f.Name] = string(b)
		d := xml.NewDecoder(bytes.NewReader(b))
		for {
			if _, err := d.Token(); err == io.EOF {
				break
			} else if err != nil {
				t.Fatalf("%s mal formé : %v", f.Name, err)
			}
		}
	}
	return parts
}

func TestClasseurXLSX(t *testing.T) {
	r := terrainRapport(t)
	s, _ := scan.Lancer(r)
	s.Attendre()
	d, _ := s.ChercherDoublons(0, 1)
	d.Attendre()
	rech, _ := s.Chercher(0, scan.Criteres{Nom: "*.txt"})
	rech.Attendre()

	var b bytes.Buffer
	if err := ClasseurXLSX(&b, s, rech, d, Options{Mode: scan.ModeTaille}, "0.7.0", time.Now()); err != nil {
		t.Fatal(err)
	}
	parts := bienForme(t, b.Bytes())
	for _, nom := range []string{"[Content_Types].xml", "_rels/.rels", "xl/workbook.xml", "xl/_rels/workbook.xml.rels", "xl/styles.xml", "docProps/core.xml"} {
		if _, ok := parts[nom]; !ok {
			t.Errorf("%s absent", nom)
		}
	}
	wb := parts["xl/workbook.xml"]
	for _, onglet := range []string{"Résumé", "Dossiers", "Gros fichiers", "Types", "Âge", "Recherche", "Doublons"} {
		if !strings.Contains(wb, `name="`+onglet+`"`) {
			t.Errorf("onglet %s absent", onglet)
		}
	}
	if n := strings.Count(wb, "<sheet "); n != 7 {
		t.Errorf("%d onglets", n)
	}
	// Les noms piégés sont du texte : jamais de balise, jamais de formule
	tout := strings.Join(func() []string {
		var l []string
		for _, v := range parts {
			l = append(l, v)
		}
		return l
	}(), "")
	if strings.Contains(tout, "<script>") || strings.Contains(tout, "<f>") {
		t.Fatal("contenu non échappé ou formule dans le classeur")
	}
	if !strings.Contains(parts["xl/worksheets/sheet7.xml"], "vacances.mkv") {
		t.Error("le doublon manque dans l'onglet Doublons")
	}
}

func TestClasseurSansRechercheNiDoublons(t *testing.T) {
	s, _ := scan.Lancer(terrainRapport(t))
	s.Attendre()
	var b bytes.Buffer
	if err := ClasseurXLSX(&b, s, nil, nil, Options{Mode: scan.ModeDisque, Profondeur: 1}, "dev", time.Now()); err != nil {
		t.Fatal(err)
	}
	parts := bienForme(t, b.Bytes())
	if n := strings.Count(parts["xl/workbook.xml"], "<sheet "); n != 5 {
		t.Errorf("%d onglets au lieu de 5", n)
	}
	// Profondeur 1 : la racine et ses 4 dossiers, plus l'en-tête
	if n := strings.Count(parts["xl/worksheets/sheet2.xml"], "<row "); n != 6 {
		t.Errorf("%d lignes dans Dossiers", n)
	}
}
