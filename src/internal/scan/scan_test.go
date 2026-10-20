package scan

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// ecrire crée un fichier de n octets (et ses dossiers parents).
func ecrire(t *testing.T, chemin string, n int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(chemin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chemin, make([]byte, n), 0o644); err != nil {
		t.Fatal(err)
	}
}

func scanner(t *testing.T, racine string) *Scan {
	t.Helper()
	s, err := Lancer(racine)
	if err != nil {
		t.Fatal(err)
	}
	s.Attendre()
	return s
}

// arbre de test :
//
//	racine/            a.bin 100
//	  gros/            g1 5000, g2 3000
//	    profond/x/y/   z 2000
//	  petit/           p 10
//	  vide/
func arbre(t *testing.T) string {
	r := t.TempDir()
	ecrire(t, filepath.Join(r, "a.bin"), 100)
	ecrire(t, filepath.Join(r, "gros", "g1"), 5000)
	ecrire(t, filepath.Join(r, "gros", "g2"), 3000)
	ecrire(t, filepath.Join(r, "gros", "profond", "x", "y", "z"), 2000)
	ecrire(t, filepath.Join(r, "petit", "p"), 10)
	if err := os.Mkdir(filepath.Join(r, "vide"), 0o755); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestTotaux(t *testing.T) {
	s := scanner(t, arbre(t))

	p := s.Progression()
	if p.Etat != Fini {
		t.Fatalf("état = %q, attendu %q", p.Etat, Fini)
	}
	if p.Fichiers != 5 || p.Taille != 10110 {
		t.Fatalf("progression : %d fichiers, %d octets ; attendu 5 et 10110", p.Fichiers, p.Taille)
	}
	// gros, profond, x, y, petit, vide (la racine elle-même n'est pas comptée)
	if p.Dossiers != 6 {
		t.Fatalf("progression : %d dossiers, attendu 6", p.Dossiers)
	}

	r, err := s.RacineEntree()
	if err != nil {
		t.Fatal(err)
	}
	if r.Taille != 10110 || r.Fichiers != 5 || r.Dossiers != 6 {
		t.Fatalf("racine = %+v", r)
	}
	if r.Disque < r.Taille {
		t.Errorf("taille sur disque %d inférieure à la taille réelle %d", r.Disque, r.Taille)
	}
}

func TestEnfantsTriesEtFichiers(t *testing.T) {
	s := scanner(t, arbre(t))

	enf, err := s.Enfants(0, ModeTaille, Tri{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	var ordre []string
	for _, e := range enf {
		ordre = append(ordre, e.Type+":"+e.Nom)
	}
	attendu := []string{"dossier:gros", "fichiers:", "dossier:petit", "dossier:vide"}
	if fmt.Sprint(ordre) != fmt.Sprint(attendu) {
		t.Fatalf("ordre = %v, attendu %v", ordre, attendu)
	}
	gros := enf[0]
	if gros.Taille != 10000 || gros.Fichiers != 3 || gros.Dossiers != 3 || !gros.Ouvrable {
		t.Fatalf("gros = %+v", gros)
	}
	if enf[3].Ouvrable {
		t.Error("un dossier vide ne doit pas être dépliable")
	}

	// Les fichiers posés dans « gros », relus sur le disque
	fich, err := s.Fichiers(gros.ID, ModeTaille, Tri{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(fich) != 2 || fich[0].Nom != "g1" || fich[0].Taille != 5000 || fich[1].Nom != "g2" {
		t.Fatalf("fichiers de gros = %+v", fich)
	}

	chemin, err := s.Chemin(gros.ID)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(chemin) != "gros" || filepath.Dir(chemin) != s.Racine() {
		t.Fatalf("chemin = %q", chemin)
	}
}

func TestLimiteRegroupeLeReste(t *testing.T) {
	r := t.TempDir()
	for i := 1; i <= 10; i++ {
		ecrire(t, filepath.Join(r, fmt.Sprintf("d%02d", i), "f"), i*100)
	}
	s := scanner(t, r)

	enf, err := s.Enfants(0, ModeTaille, Tri{}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(enf) != 4 {
		t.Fatalf("%d lignes, attendu 3 + le reste", len(enf))
	}
	if enf[0].Nom != "d10" || enf[2].Nom != "d08" {
		t.Fatalf("mauvais tri : %+v", enf[:3])
	}
	reste := enf[3]
	// d01 à d07 : 100 + 200 + … + 700
	if reste.Type != "reste" || reste.Nombre != 7 || reste.Taille != 2800 {
		t.Fatalf("reste = %+v", reste)
	}
}

func TestLiensNonSuivis(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("liens symboliques : droits particuliers sous Windows")
	}
	r := arbre(t)
	// Une boucle : racine/gros/boucle -> racine
	if err := os.Symlink(r, filepath.Join(r, "gros", "boucle")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(r, "a.bin"), filepath.Join(r, "lien.bin")); err != nil {
		t.Fatal(err)
	}
	s := scanner(t, r)
	p := s.Progression()
	if p.Liens != 2 {
		t.Errorf("%d liens, attendu 2", p.Liens)
	}
	if p.Taille != 10110 {
		t.Errorf("taille %d : un lien a été suivi", p.Taille)
	}
}

func TestDossierIllisible(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("nécessite un utilisateur non administrateur sous Unix")
	}
	r := arbre(t)
	interdit := filepath.Join(r, "petit")
	if err := os.Chmod(interdit, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(interdit, 0o755) })

	s := scanner(t, r)
	racine, _ := s.RacineEntree()
	if racine.Erreurs != 1 {
		t.Fatalf("%d erreurs, attendu 1", racine.Erreurs)
	}
}

func TestArretGardeLePartiel(t *testing.T) {
	r := t.TempDir()
	for i := range 200 {
		ecrire(t, filepath.Join(r, fmt.Sprintf("d%03d", i), "a", "b", "f"), 1)
	}
	s, err := Lancer(r)
	if err != nil {
		t.Fatal(err)
	}
	s.Arreter()
	s.Attendre()
	if e := s.Progression().Etat; e != Arrete && e != Fini {
		t.Fatalf("état = %q", e)
	}
	if _, err := s.Enfants(0, ModeTaille, Tri{}, 0); err != nil {
		t.Fatalf("le résultat partiel doit rester consultable : %v", err)
	}
}

func TestErreurs(t *testing.T) {
	r := arbre(t)
	if _, err := Lancer(filepath.Join(r, "a.bin")); !errors.Is(err, ErrPasUnDossier) {
		t.Errorf("fichier comme racine : %v", err)
	}
	if _, err := Lancer(filepath.Join(r, "inexistant")); err == nil {
		t.Error("dossier inexistant accepté")
	}
	s := scanner(t, r)
	if _, err := s.Enfants(999, ModeTaille, Tri{}, 0); err == nil {
		t.Error("id inconnu accepté")
	}
}

func TestModeDisque(t *testing.T) {
	s := scanner(t, arbre(t))
	enf, err := s.Enfants(0, ModeDisque, Tri{}, 0)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(enf); i++ {
		if enf[i-1].Disque < enf[i].Disque {
			t.Fatalf("pas trié par taille sur disque : %+v", enf)
		}
	}
}
