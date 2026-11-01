package scan

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func contenu(t *testing.T, chemin string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(chemin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chemin, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasard(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

// jumeaux : des doublons, et des pièges pour chaque passe.
func jumeaux(t *testing.T) (string, []byte) {
	t.Helper()
	r := t.TempDir()
	a := hasard(200 << 10)
	contenu(t, filepath.Join(r, "a", "x.bin"), a)
	contenu(t, filepath.Join(r, "b", "x-copie.bin"), a)
	contenu(t, filepath.Join(r, "c", "autre nom.bin"), a)
	// Même taille, même début et même fin : seul le SHA-256 complet les sépare.
	b := slices.Clone(a)
	b[100<<10] ^= 0xFF
	contenu(t, filepath.Join(r, "c", "piege-milieu.bin"), b)
	// Même taille, début différent : écarté dès l'empreinte rapide.
	c := slices.Clone(a)
	c[0] ^= 0xFF
	contenu(t, filepath.Join(r, "c", "piege-debut.bin"), c)
	// Petits fichiers : lus en entier dès l'empreinte rapide.
	d := hasard(10 << 10)
	contenu(t, filepath.Join(r, "a", "petit.txt"), d)
	contenu(t, filepath.Join(r, "b", "petit.txt"), d)
	contenu(t, filepath.Join(r, "b", "petit-different.txt"), hasard(10<<10))
	// Fichiers vides : jamais des doublons.
	contenu(t, filepath.Join(r, "a", "vide"), nil)
	contenu(t, filepath.Join(r, "b", "vide"), nil)
	// Lien physique vers x.bin : le même fichier, pas une copie.
	if err := os.MkdirAll(filepath.Join(r, "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(filepath.Join(r, "a", "x.bin"), filepath.Join(r, "d", "lien.bin")); err != nil {
		t.Fatalf("lien physique : %v", err)
	}
	return r, a
}

func doublons(t *testing.T, s *Scan, id int32, min int64) ([]GroupeDoublons, SuiviDoublons) {
	t.Helper()
	d, err := s.ChercherDoublons(id, min)
	if err != nil {
		t.Fatal(err)
	}
	d.Attendre()
	g, err := d.Groupes()
	if err != nil {
		t.Fatal(err)
	}
	return g, d.Suivi()
}

func nomsGroupe(g GroupeDoublons) string {
	var n []string
	for _, f := range g.Fichiers {
		n = append(n, f.Nom)
	}
	slices.Sort(n)
	return strings.Join(n, ",")
}

func TestDoublons(t *testing.T) {
	r, _ := jumeaux(t)
	s := scanner(t, r)
	g, su := doublons(t, s, 0, 1)
	if len(g) != 2 {
		for _, x := range g {
			t.Log(nomsGroupe(x))
		}
		t.Fatalf("%d groupes au lieu de 2 ; suivi %+v", len(g), su)
	}
	if nomsGroupe(g[0]) != "autre nom.bin,x-copie.bin,x.bin" {
		t.Errorf("premier groupe : %s", nomsGroupe(g[0]))
	}
	if g[0].Recuperable != 2*(200<<10) || g[0].Taille != 200<<10 || len(g[0].Empreinte) != 64 {
		t.Errorf("premier groupe : %+v", g[0])
	}
	if nomsGroupe(g[1]) != "petit.txt,petit.txt" || g[1].Recuperable != 10<<10 {
		t.Errorf("second groupe : %s", nomsGroupe(g[1]))
	}
	if su.Etat != Fini || su.Groupes != 2 || su.Copies != 3 || su.Recuperable != 2*(200<<10)+10<<10 {
		t.Errorf("suivi : %+v", su)
	}
	// Chaque fichier sait où il est, comme dans la recherche
	for _, f := range g[0].Fichiers {
		if _, err := os.Stat(filepath.Join(f.Dossier, f.Nom)); err != nil {
			t.Errorf("chemin introuvable : %s / %s", f.Dossier, f.Nom)
		}
	}
}

func TestDoublonsTailleMin(t *testing.T) {
	r, _ := jumeaux(t)
	s := scanner(t, r)
	g, _ := doublons(t, s, 0, 50<<10)
	if len(g) != 1 || !strings.Contains(nomsGroupe(g[0]), "x.bin") {
		t.Fatalf("avec 50 Ko minimum : %d groupes", len(g))
	}
}

func TestDoublonsApresMenage(t *testing.T) {
	r, _ := jumeaux(t)
	s := scanner(t, r)
	d, _ := s.ChercherDoublons(0, 1)
	d.Attendre()
	g, _ := d.Groupes()
	retirer := func(f Trouve) {
		c := Cible{Type: "fichier", ID: f.ID, Nom: f.Nom}
		rt, err := s.PreparerRetrait([]Cible{c})
		if err != nil {
			t.Fatal(err)
		}
		os.Remove(rt[0].Chemin)
		s.Retirer(rt)
	}
	retirer(g[0].Fichiers[0]) // une copie du gros
	retirer(g[1].Fichiers[0]) // un des deux petits : plus de doublon
	g2, _ := d.Groupes()
	if len(g2) != 1 || len(g2[0].Fichiers) != 2 || g2[0].Recuperable != 200<<10 {
		t.Fatalf("après ménage : %d groupes", len(g2))
	}
}

func TestDoublonsDansUnDossier(t *testing.T) {
	r, _ := jumeaux(t)
	s := scanner(t, r)
	enf, _ := s.Enfants(0, ModeTaille, Tri{}, 0)
	var idC int32
	for _, e := range enf {
		if e.Nom == "c" {
			idC = e.ID
		}
	}
	// Dans « c », une seule copie de x.bin : pas de doublon.
	if g, _ := doublons(t, s, idC, 1); len(g) != 0 {
		t.Fatalf("%d groupes dans c", len(g))
	}
}

func TestDoublonsArret(t *testing.T) {
	r := t.TempDir()
	a := hasard(4 << 20)
	for i := range 6 {
		contenu(t, filepath.Join(r, strings.Repeat("d", i+1), "gros.bin"), a)
	}
	s := scanner(t, r)
	d, err := s.ChercherDoublons(0, 1)
	if err != nil {
		t.Fatal(err)
	}
	d.Arreter()
	d.Attendre()
	if su := d.Suivi(); su.Etat == EnCours {
		t.Fatalf("toujours en cours : %+v", su)
	}
}
