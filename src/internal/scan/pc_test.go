package scan

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestPC(t *testing.T) {
	c := arbre(t) // 10110 octets, 5 fichiers
	d := t.TempDir()
	ecrire(t, filepath.Join(d, "jeux", "gros.pak"), 50000)
	ecrire(t, filepath.Join(d, "jeux", "save.dat"), 300)

	s, err := LancerPC("Ce PC", []string{c, d, c, filepath.Join(d, "absent")}, Filtres{})
	if err != nil {
		t.Fatal(err)
	}
	s.Attendre()

	p := s.Progression()
	if !p.PC || p.Racine != "Ce PC" || p.Etat != Fini {
		t.Fatalf("progression : %+v", p)
	}
	racine, _ := s.RacineEntree()
	if racine.Taille != 60410 || racine.Fichiers != 7 {
		t.Fatalf("total : %d octets, %d fichiers", racine.Taille, racine.Fichiers)
	}
	// Doublon et racine absente ignorés : deux disques
	disques, _ := s.Enfants(0, ModeTaille, Tri{}, 0)
	if len(disques) != 2 || disques[0].Nom != d || disques[1].Nom != c {
		t.Fatalf("disques : %+v", disques)
	}
	if p.Dossiers != racine.Dossiers {
		t.Errorf("dossiers : progression %d, arbre %d", p.Dossiers, racine.Dossiers)
	}

	// Les chemins partent du disque, pas de « Ce PC »
	if _, err := s.Chemin(0); !errors.Is(err, ErrRacineVirtuelle) {
		t.Errorf("Chemin(racine) : %v", err)
	}
	if ch, _ := s.Chemin(disques[0].ID); ch != d {
		t.Errorf("chemin du disque : %q", ch)
	}
	jeux, _ := s.Enfants(disques[0].ID, ModeTaille, Tri{}, 0)
	if ch, _ := s.Chemin(jeux[0].ID); ch != filepath.Join(d, "jeux") {
		t.Errorf("chemin de jeux : %q", ch)
	}
	f, err := s.Fichiers(jeux[0].ID, ModeTaille, Tri{}, 0)
	if err != nil || len(f) != 2 || f[0].Nom != "gros.pak" {
		t.Fatalf("fichiers de jeux : %+v, %v", f, err)
	}

	// Gros fichiers, types, âge et carte couvrent tous les disques
	g, _ := s.GrosFichiers(ModeTaille, Tri{})
	if len(g) != 7 || g[0].Nom != "gros.pak" || g[0].Dossier != filepath.Join(d, "jeux") {
		t.Fatalf("gros fichiers : %+v", g[0])
	}
	types, _ := s.Types(0, ModeTaille)
	if types[0].Nom != "Jeux" || types[0].Taille != 50000 {
		t.Errorf("types : %+v", types[0])
	}
	var n int64
	typesDisque, _ := s.Types(disques[1].ID, ModeTaille)
	for _, f := range typesDisque {
		n += f.Fichiers
	}
	if n != 5 {
		t.Errorf("types du second disque : %d fichiers, attendu 5", n)
	}
	carte, _ := s.Carte(0, ModeTaille)
	verifierSommes(t, carte)
	if fil, _ := s.Fil(jeux[0].ID); len(fil) != 3 || fil[0].Nom != "Ce PC" || fil[1].Nom != d {
		t.Errorf("fil : %+v", fil)
	}
}

func TestPCSansDisque(t *testing.T) {
	if _, err := LancerPC("Ce PC", []string{filepath.Join(t.TempDir(), "absent")}, Filtres{}); !errors.Is(err, ErrAucunDisque) {
		t.Errorf("erreur : %v", err)
	}
}

func TestPCArret(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	for i := range 100 {
		ecrire(t, filepath.Join(a, "d", string(rune('a'+i%26)), "f"), 1)
		ecrire(t, filepath.Join(b, "d", string(rune('a'+i%26)), "g"), 1)
	}
	s, err := LancerPC("Ce PC", []string{a, b}, Filtres{})
	if err != nil {
		t.Fatal(err)
	}
	s.Arreter()
	s.Attendre()
	if _, err := s.Enfants(0, ModeTaille, Tri{}, 0); err != nil {
		t.Fatal(err)
	}
}
