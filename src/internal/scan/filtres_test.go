package scan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// medias :
//
//	films/      a.mkv 5000, b.MP4 3000, bande-annonce.mkv 200
//	films/node_modules/  x.mkv 9999 (doit disparaître avec l'exclusion)
//	docs/       cv.pdf 400 (vieux de 3 ans)
//	vide/
func medias(t *testing.T) string {
	r := t.TempDir()
	ecrire(t, filepath.Join(r, "films", "a.mkv"), 5000)
	ecrire(t, filepath.Join(r, "films", "b.MP4"), 3000)
	ecrire(t, filepath.Join(r, "films", "bande-annonce.mkv"), 200)
	ecrire(t, filepath.Join(r, "films", "node_modules", "x.mkv"), 9999)
	ecrire(t, filepath.Join(r, "docs", "cv.pdf"), 400)
	vieux := time.Now().AddDate(-3, 0, 0)
	if err := os.Chtimes(filepath.Join(r, "docs", "cv.pdf"), vieux, vieux); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(r, "vide"), 0o755); err != nil {
		t.Fatal(err)
	}
	return r
}

func scannerAvec(t *testing.T, r string, f Filtres) *Scan {
	t.Helper()
	s, err := LancerAvec(r, f)
	if err != nil {
		t.Fatal(err)
	}
	s.Attendre()
	return s
}

func TestFiltres(t *testing.T) {
	r := medias(t)
	cas := []struct {
		nom              string
		f                Filtres
		taille, fichiers int64
		dossiers         string // sous-dossiers de la racine, après élagage
	}{
		{"sans filtre", Filtres{}, 18599, 5, "[films docs vide]"},
		{"exclure un dossier", Filtres{Exclure: []string{"node_modules"}}, 8600, 4, "[films docs vide]"},
		{"exclure et inclure", Filtres{Exclure: []string{"NODE_MODULES"}, Inclure: []string{"*.mkv"}}, 5200, 2, "[films]"},
		{"inclure, casse ignorée", Filtres{Inclure: []string{"*.mp4"}}, 3000, 1, "[films]"},
		{"taille minimale", Filtres{TailleMin: 1000}, 17999, 3, "[films]"},
		{"ancienneté", Filtres{AgeMinJours: 365}, 400, 1, "[docs]"},
		{"regex", Filtres{Regex: `^bande`}, 200, 1, "[films]"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			s := scannerAvec(t, r, c.f)
			racine, _ := s.RacineEntree()
			if racine.Taille != c.taille || racine.Fichiers != c.fichiers {
				t.Errorf("total %d (%d fichiers), attendu %d (%d)", racine.Taille, racine.Fichiers, c.taille, c.fichiers)
			}
			enf, _ := s.Enfants(0, ModeTaille, Tri{}, 0)
			var noms []string
			for _, e := range enf {
				if e.Type == "dossier" {
					noms = append(noms, e.Nom)
				}
			}
			if got := "[" + strings.Join(noms, " ") + "]"; got != c.dossiers {
				t.Errorf("dossiers %s, attendu %s", got, c.dossiers)
			}
			p := s.Progression()
			if p.Fichiers != c.fichiers {
				t.Errorf("progression : %d fichiers", p.Fichiers)
			}
			if c.f.Actifs() && p.Ecartes == 0 && c.nom != "exclure un dossier" {
				t.Error("aucun fichier écarté signalé")
			}
		})
	}
}

func TestFiltresPartout(t *testing.T) {
	s := scannerAvec(t, medias(t), Filtres{Inclure: []string{"*.mkv"}, Exclure: []string{"node_modules"}})
	// Les fichiers relus d'un dossier respectent les filtres
	enf, _ := s.Enfants(0, ModeTaille, Tri{}, 0)
	films := enf[0]
	f, _ := s.Fichiers(films.ID, ModeTaille, Tri{}, 0)
	if len(f) != 2 || f[0].Nom != "a.mkv" {
		t.Fatalf("fichiers de films : %+v", f)
	}
	// Les statistiques aussi, pour toute l'analyse comme pour une branche
	for _, id := range []int32{0, films.ID} {
		types, _ := s.Types(id, ModeTaille)
		if len(types) != 1 || types[0].Nom != "Vidéos" || types[0].Fichiers != 2 {
			t.Fatalf("types (%d) : %+v", id, types)
		}
	}
	g, _ := s.GrosFichiers(ModeTaille, Tri{})
	if len(g) != 2 {
		t.Fatalf("gros fichiers : %d", len(g))
	}
	if s.Progression().Dossiers != 1 {
		t.Errorf("%d dossiers après élagage, attendu 1", s.Progression().Dossiers)
	}
}

func TestFiltresInvalides(t *testing.T) {
	r := t.TempDir()
	for _, f := range []Filtres{
		{Regex: "(non fermée"},
		{Exclure: []string{"[z-a"}},
		{Inclure: []string{`dossier\fichier`}},
	} {
		if _, err := LancerAvec(r, f); err == nil {
			t.Errorf("%+v accepté", f)
		}
	}
	_, err := LancerAvec(r, Filtres{Regex: "(cassée"})
	if err == nil || !strings.Contains(err.Error(), "parenthèse non fermée") || strings.Contains(err.Error(), "(?i)") {
		t.Errorf("message : %v", err)
	}
	s := scannerAvec(t, medias(t), Filtres{Exclure: []string{"  ", ""}})
	if racine, _ := s.RacineEntree(); racine.Fichiers != 5 {
		t.Errorf("des motifs vides ont filtré : %d fichiers", racine.Fichiers)
	}
	if s.Filtres().Actifs() {
		t.Error("des motifs vides ne doivent pas compter comme un filtre actif")
	}
}
