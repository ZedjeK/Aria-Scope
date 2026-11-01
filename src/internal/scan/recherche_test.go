package scan

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// photos : de quoi chercher.
//
//	r/Photos/2019/   IMG_0001.jpg 4000 (2019), IMG_0002.JPG 6000 (2019), notes.txt 50
//	r/Photos/2024/   IMG_1000.jpg 9000 (2024)
//	r/Films/         vacances.mkv 50000 (2024), bande-annonce.mp4 7000
//	r/Projets/node_modules/lib/  index.js 300
func photos(t *testing.T) string {
	t.Helper()
	r := t.TempDir()
	ecrire(t, filepath.Join(r, "Photos", "2019", "IMG_0001.jpg"), 4000)
	ecrire(t, filepath.Join(r, "Photos", "2019", "IMG_0002.JPG"), 6000)
	ecrire(t, filepath.Join(r, "Photos", "2019", "notes.txt"), 50)
	ecrire(t, filepath.Join(r, "Photos", "2024", "IMG_1000.jpg"), 9000)
	ecrire(t, filepath.Join(r, "Films", "vacances.mkv"), 50000)
	ecrire(t, filepath.Join(r, "Films", "bande-annonce.mp4"), 7000)
	ecrire(t, filepath.Join(r, "Projets", "node_modules", "lib", "index.js"), 300)
	dater := func(p string, annee int) {
		d := time.Date(annee, 6, 15, 12, 0, 0, 0, time.Local)
		if err := os.Chtimes(filepath.Join(r, p), d, d); err != nil {
			t.Fatal(err)
		}
	}
	dater("Photos/2019/IMG_0001.jpg", 2019)
	dater("Photos/2019/IMG_0002.JPG", 2019)
	dater("Photos/2024/IMG_1000.jpg", 2024)
	dater("Films/vacances.mkv", 2024)
	return r
}

func chercher(t *testing.T, s *Scan, id int32, c Criteres) ([]Trouve, SuiviRecherche) {
	t.Helper()
	r, err := s.Chercher(id, c)
	if err != nil {
		t.Fatal(err)
	}
	r.Attendre()
	l, err := r.Resultats(ModeTaille, Tri{})
	if err != nil {
		t.Fatal(err)
	}
	return l, r.Suivi()
}

func nomsDe(l []Trouve) string {
	var n []string
	for _, x := range l {
		n = append(n, x.Nom)
	}
	return strings.Join(n, " ")
}

func TestRechercheCriteres(t *testing.T) {
	s := scanner(t, photos(t))
	cas := []struct {
		nom     string
		c       Criteres
		attendu string
	}{
		{"tout", Criteres{}, "vacances.mkv IMG_1000.jpg bande-annonce.mp4 IMG_0002.JPG IMG_0001.jpg index.js notes.txt"},
		{"joker, casse ignorée", Criteres{Nom: "img_*.jpg"}, "IMG_1000.jpg IMG_0002.JPG IMG_0001.jpg"},
		{"plusieurs motifs", Criteres{Nom: "*.mkv ; *.mp4"}, "vacances.mkv bande-annonce.mp4"},
		{"contient", Criteres{Nom: "ANNONCE"}, "bande-annonce.mp4"},
		{"regex", Criteres{Nom: `^img_000\d`, Regex: true}, "IMG_0002.JPG IMG_0001.jpg"},
		{"taille", Criteres{TailleMin: 5000, TailleMax: 9000}, "IMG_1000.jpg bande-annonce.mp4 IMG_0002.JPG"},
		{"dates", Criteres{ModifAvant: time.Date(2020, 1, 1, 0, 0, 0, 0, time.Local).Unix()}, "IMG_0002.JPG IMG_0001.jpg"},
		{"famille", Criteres{Familles: []string{"Vidéos"}}, "vacances.mkv bande-annonce.mp4"},
		{"famille et taille", Criteres{Familles: []string{"Images", "Code"}, TailleMax: 5000}, "IMG_0001.jpg index.js"},
		{"dossiers", Criteres{Quoi: QuoiDossiers, Nom: "20*"}, "2019 2024"},
		{"tout type", Criteres{Quoi: QuoiTout, Nom: "node"}, "node_modules"},
		{"rien", Criteres{Nom: "*.iso"}, ""},
	}
	for _, c := range cas {
		l, su := chercher(t, s, 0, c.c)
		if got := nomsDe(l); got != c.attendu {
			t.Errorf("%s : %q, attendu %q", c.nom, got, c.attendu)
		}
		if su.Etat != Fini || su.Trouves != int64(len(l)) {
			t.Errorf("%s : suivi %+v", c.nom, su)
		}
	}
}

func TestRechercheDansUnDossier(t *testing.T) {
	r := photos(t)
	s := scanner(t, r)
	enf, _ := s.Enfants(0, ModeTaille, Tri{}, 0)
	var idPhotos int32
	for _, e := range enf {
		if e.Nom == "Photos" {
			idPhotos = e.ID
		}
	}
	l, su := chercher(t, s, idPhotos, Criteres{Nom: "*.jpg"})
	if nomsDe(l) != "IMG_1000.jpg IMG_0002.JPG IMG_0001.jpg" || su.Dossiers != 3 || su.Lus != 3 {
		t.Fatalf("%s, %+v", nomsDe(l), su)
	}
	// Le dossier de chaque résultat, et la longueur du chemin complet
	x := l[0]
	if x.Dossier != filepath.Join(r, "Photos", "2024") || x.Type != "fichier" {
		t.Errorf("résultat : %+v", x)
	}
	if want := len(filepath.Join(r, "Photos", "2024", "IMG_1000.jpg")); x.Longueur != want {
		t.Errorf("longueur %d, attendu %d", x.Longueur, want)
	}
}

func TestRechercheCheminsLongs(t *testing.T) {
	r := t.TempDir()
	long := strings.Repeat("dossier-au-nom-interminable-", 3) // 84 caractères
	ecrire(t, filepath.Join(r, long, long, "court.txt"), 10)
	ecrire(t, filepath.Join(r, "court.txt"), 10)
	s := scanner(t, r)
	seuil := len(r) + 2*85 + 1 // seul le fichier du fond le dépasse
	l, _ := chercher(t, s, 0, Criteres{CheminMin: seuil})
	if len(l) != 1 || l[0].Longueur < seuil || l[0].Dossier != filepath.Join(r, long, long) {
		t.Fatalf("%+v", l)
	}
	// Les caractères hors du plan de base comptent double, comme pour Windows.
	if longueur("a😀b") != 4 || longueur("été") != 3 {
		t.Error("longueur UTF-16 incorrecte")
	}
}

func TestRechercheRespecteLesFiltres(t *testing.T) {
	r := photos(t)
	s := scannerAvec(t, r, Filtres{Exclure: []string{"node_modules"}, TailleMin: 100})
	l, _ := chercher(t, s, 0, Criteres{})
	if n := nomsDe(l); strings.Contains(n, "index.js") || strings.Contains(n, "notes.txt") {
		t.Fatalf("les filtres de l'analyse doivent s'appliquer : %s", n)
	}
}

// MaxTrouvesTest change le plafond le temps d'un test et renvoie l'ancien.
func MaxTrouvesTest(n int) int {
	ancien := plafond
	plafond = n
	return ancien
}

func TestRecherchePlafond(t *testing.T) {
	r := t.TempDir()
	for i := range 30 {
		ecrire(t, filepath.Join(r, "lot", strings.Repeat("x", i+1)+".dat"), i+1)
	}
	s := scanner(t, r)
	ancien := MaxTrouvesTest(10)
	defer MaxTrouvesTest(ancien)
	l, su := chercher(t, s, 0, Criteres{})
	if len(l) != 10 || su.Trouves != 30 || su.Gardes != 10 || l[0].Taille != 30 || l[9].Taille != 21 {
		t.Fatalf("plafond : %d gardés sur %d, %+v", len(l), su.Trouves, su)
	}
}

func TestRechercheApresMenage(t *testing.T) {
	r := photos(t)
	s := scanner(t, r)
	rech, _ := s.Chercher(0, Criteres{Nom: "*.jpg"})
	rech.Attendre()
	l, _ := rech.Resultats(ModeTaille, Tri{})
	// Le plus gros part à la corbeille (simulé : retiré du disque puis de l'analyse).
	c := Cible{Type: "fichier", ID: l[0].ID, Nom: l[0].Nom}
	rt, err := s.PreparerRetrait([]Cible{c})
	if err != nil {
		t.Fatal(err)
	}
	os.Remove(rt[0].Chemin)
	s.Retirer(rt)
	l2, _ := rech.Resultats(ModeTaille, Tri{})
	if len(l2) != 2 || slices.ContainsFunc(l2, func(x Trouve) bool { return x.Nom == l[0].Nom }) {
		t.Fatalf("après ménage : %s", nomsDe(l2))
	}
}

func TestRechercheInvalide(t *testing.T) {
	s := scanner(t, photos(t))
	for nom, c := range map[string]Criteres{
		"regex":   {Nom: "(abc", Regex: true},
		"motif":   {Nom: "[a-"},
		"tailles": {TailleMin: 10, TailleMax: 5},
		"dates":   {ModifApres: 2000, ModifAvant: 1000},
		"quoi":    {Quoi: "liens"},
	} {
		if _, err := s.Chercher(0, c); err == nil {
			t.Errorf("%s : critère invalide accepté", nom)
		}
	}
}

func TestRechercheArret(t *testing.T) {
	r := t.TempDir()
	for i := range 400 {
		ecrire(t, filepath.Join(r, "d", strings.Repeat("a", 1+i%20), "f"+strings.Repeat("b", i%7), "x.bin"), 1)
	}
	s := scanner(t, r)
	rech, err := s.Chercher(0, Criteres{})
	if err != nil {
		t.Fatal(err)
	}
	rech.Arreter()
	rech.Attendre()
	if su := rech.Suivi(); su.Etat != Arrete && su.Etat != Fini {
		t.Fatalf("état %s", su.Etat)
	}
}

func TestNomsFamilles(t *testing.T) {
	n := NomsFamilles()
	if len(n) != len(familles)+2 || n[len(n)-1] != familleSans || !slices.Contains(n, "Vidéos") {
		t.Fatalf("%v", n)
	}
}
