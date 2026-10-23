//go:build !windows

package menage

import (
	"os"
	"path/filepath"
	"testing"

	"aria-scope/internal/scan"
	"aria-scope/internal/volumes"
)

func existe(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

// bac (operations_other_test.go) :
//
//	disque/Téléchargements/  film.mkv 5000, setup.exe 300
//	disque/Jeux/vieux/       data.pak 2000
//	disque/Systeme/          noyau.sys 900  (protégé pour les tests)

func TestDeplacerHorsAnalyse(t *testing.T) {
	racine, s := bac(t)
	archives := filepath.Join(filepath.Dir(racine), "archives")
	if err := os.Mkdir(archives, 0o755); err != nil {
		t.Fatal(err)
	}
	telech := cibleDossier(t, s, 0, "Téléchargements")
	jeux := cibleDossier(t, s, 0, "Jeux")
	vieux := cibleDossier(t, s, jeux.ID, "vieux")
	systeme := cibleDossier(t, s, 0, "Systeme")
	cibles := []scan.Cible{vieux, {Type: "fichier", ID: telech.ID, Nom: "film.mkv", Taille: 5000}, systeme}

	a, err := Apercevoir(s, OpDeplacer, cibles, archives)
	if err != nil {
		t.Fatal(err)
	}
	if a.Nombre != 2 || a.Taille != 7000 || len(a.Ecartes) != 1 || a.DansAnalyse || len(a.Conflits) != 0 || a.Manque != 0 {
		t.Fatalf("aperçu : %+v", a)
	}
	if !existe(filepath.Join(racine, "Jeux", "vieux")) {
		t.Fatal("l'aperçu a touché au disque")
	}

	b, err := Executer(s, OpDeplacer, cibles, archives, 0)
	if err != nil {
		t.Fatal(err)
	}
	if b.Faits != 2 || b.Taille != 7000 || b.Erreur != "" || len(b.Restants) != 1 || b.Perime {
		t.Fatalf("bilan : %+v", b)
	}
	if existe(filepath.Join(racine, "Jeux", "vieux")) || existe(filepath.Join(racine, "Téléchargements", "film.mkv")) {
		t.Error("les originaux sont toujours là")
	}
	if !existe(filepath.Join(archives, "vieux", "data.pak")) || !existe(filepath.Join(archives, "film.mkv")) {
		t.Error("rien n'est arrivé dans les archives")
	}
	if !existe(filepath.Join(racine, "Systeme", "noyau.sys")) || existe(filepath.Join(archives, "Systeme")) {
		t.Fatal("un élément protégé a été déplacé !")
	}
	if r, _ := s.RacineEntree(); r.Taille != 1200 || r.Fichiers != 2 {
		t.Errorf("l'analyse n'a pas suivi : %+v", r)
	}
}

func TestDeplacerRefuse(t *testing.T) {
	racine, s := bac(t)
	telech := cibleDossier(t, s, 0, "Téléchargements")
	jeux := cibleDossier(t, s, 0, "Jeux")

	// Un dossier dans lui-même, un fichier déjà dans la destination
	a, err := Apercevoir(s, OpDeplacer, []scan.Cible{jeux}, filepath.Join(racine, "Jeux", "vieux"))
	if err != nil || a.Nombre != 0 || len(a.Ecartes) != 1 || a.Ecartes[0].Raison != "un dossier ne peut pas aller dans lui-même" {
		t.Fatalf("dossier dans lui-même : %+v, %v", a, err)
	}
	a, err = Apercevoir(s, OpDeplacer, []scan.Cible{{Type: "fichier", ID: telech.ID, Nom: "film.mkv"}}, filepath.Join(racine, "Téléchargements"))
	if err != nil || a.Nombre != 0 || a.Ecartes[0].Raison != "déjà dans ce dossier" {
		t.Fatalf("déjà sur place : %+v, %v", a, err)
	}
	b, err := Executer(s, OpDeplacer, []scan.Cible{jeux}, filepath.Join(racine, "Jeux", "vieux"), 0)
	if err != nil || b.Faits != 0 || !existe(filepath.Join(racine, "Jeux", "vieux", "data.pak")) {
		t.Fatalf("bilan : %+v, %v", b, err)
	}

	// Destinations invalides
	for nom, dest := range map[string]string{
		"système":  filepath.Join(racine, "Systeme"),
		"absente":  filepath.Join(racine, "absent"),
		"fichier":  filepath.Join(racine, "Téléchargements", "setup.exe"),
		"relative": "archives",
		"vide":     "",
	} {
		if _, err := Apercevoir(s, OpDeplacer, []scan.Cible{jeux}, dest); err == nil {
			t.Errorf("destination %s acceptée", nom)
		}
	}
	if _, err := Apercevoir(s, "effacer", []scan.Cible{jeux}, ""); err == nil {
		t.Error("opération inconnue acceptée")
	}
}

func TestCopierDansAnalyse(t *testing.T) {
	racine, s := bac(t)
	telech := cibleDossier(t, s, 0, "Téléchargements")
	systeme := cibleDossier(t, s, 0, "Systeme")
	dest := filepath.Join(racine, "Jeux")
	avant, _ := s.RacineEntree()

	a, err := Apercevoir(s, OpCopier, []scan.Cible{telech, systeme}, dest)
	if err != nil {
		t.Fatal(err)
	}
	// Une copie ne fait que lire : un dossier protégé peut être copié.
	if a.Nombre != 2 || a.Besoin != 6200 || len(a.Ecartes) != 0 || !a.DansAnalyse || a.Libre <= 0 {
		t.Fatalf("aperçu : %+v", a)
	}

	b, err := Executer(s, OpCopier, []scan.Cible{telech, systeme}, dest, 0)
	if err != nil {
		t.Fatal(err)
	}
	if b.Faits != 2 || b.Taille != 6200 || !b.Perime || len(b.Restants) != 0 {
		t.Fatalf("bilan : %+v", b)
	}
	for _, p := range []string{"Téléchargements/film.mkv", "Systeme/noyau.sys"} {
		if !existe(filepath.Join(racine, p)) || !existe(filepath.Join(dest, p)) {
			t.Errorf("%s : l'original et la copie doivent exister", p)
		}
	}
	if apres, _ := s.RacineEntree(); apres.Taille != avant.Taille {
		t.Errorf("une copie ne retire rien de l'analyse : %d puis %d", avant.Taille, apres.Taille)
	}
}

func TestCopierSansEcraser(t *testing.T) {
	racine, s := bac(t)
	telech := cibleDossier(t, s, 0, "Téléchargements")
	dest := filepath.Join(filepath.Dir(racine), "cle")
	os.Mkdir(dest, 0o755)
	if err := os.WriteFile(filepath.Join(dest, "film.mkv"), []byte("le mien"), 0o644); err != nil {
		t.Fatal(err)
	}
	film := scan.Cible{Type: "fichier", ID: telech.ID, Nom: "film.mkv", Taille: 5000}

	a, _ := Apercevoir(s, OpCopier, []scan.Cible{film}, dest)
	if len(a.Conflits) != 1 || a.Conflits[0] != "film.mkv" {
		t.Fatalf("conflit non annoncé : %+v", a.Conflits)
	}
	b, err := Executer(s, OpCopier, []scan.Cible{film}, dest, 0)
	if err != nil || b.Faits != 0 {
		t.Fatalf("bilan : %+v, %v", b, err)
	}
	if contenu, _ := os.ReadFile(filepath.Join(dest, "film.mkv")); string(contenu) != "le mien" {
		t.Fatal("un fichier existant a été écrasé !")
	}
}

func TestPlaceManquante(t *testing.T) {
	racine, s := bac(t)
	dest := filepath.Join(filepath.Dir(racine), "petit-disque")
	os.Mkdir(dest, 0o755)
	// L'arrivée est sur un autre disque, presque plein.
	volumeDe = func(p string) (volumes.Volume, bool) {
		if dedans(p, dest) {
			return volumes.Volume{Chemin: dest, Type: "amovible", Total: 10000, Libre: 1000}, true
		}
		return volumes.Volume{Chemin: "/", Type: "fixe", Total: 1 << 40, Libre: 1 << 39}, true
	}
	t.Cleanup(func() { volumeDe = volumes.De })
	telech := cibleDossier(t, s, 0, "Téléchargements") // 5300 octets

	for _, op := range []string{OpCopier, OpDeplacer} {
		a, err := Apercevoir(s, op, []scan.Cible{telech}, dest)
		if err != nil || a.Besoin != 5300 || a.Libre != 1000 || a.Manque != 4300 {
			t.Fatalf("%s, aperçu : %+v, %v", op, a, err)
		}
		if _, err := Executer(s, op, []scan.Cible{telech}, dest, 0); err == nil {
			t.Fatalf("%s lancé sans la place nécessaire", op)
		}
	}
	if existe(filepath.Join(dest, "Téléchargements")) || !existe(filepath.Join(racine, "Téléchargements", "film.mkv")) {
		t.Fatal("rien n'aurait dû bouger")
	}
}
