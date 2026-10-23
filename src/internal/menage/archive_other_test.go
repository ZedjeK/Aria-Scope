//go:build !windows

package menage

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"io"
	"os"
	"path/filepath"
	"testing"

	"aria-scope/internal/scan"
	"aria-scope/internal/volumes"
)

// contenuZip : nom d'entrée → contenu (les dossiers finissent par « / »).
func contenuZip(t *testing.T, chemin string) map[string][]byte {
	t.Helper()
	r, err := zip.OpenReader(chemin)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	out := map[string][]byte{}
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("%s : %v", f.Name, err)
		}
		out[f.Name] = b
	}
	return out
}

func dossierArchives(t *testing.T, racine string) string {
	d := filepath.Join(filepath.Dir(racine), "archives")
	if err := os.Mkdir(d, 0o755); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestArchiverPuisCorbeille(t *testing.T) {
	racine, s := bac(t)
	archives := dossierArchives(t, racine)
	// Du vrai contenu, de même taille que dans l'analyse.
	film := make([]byte, 5000)
	rand.Read(film)
	os.WriteFile(filepath.Join(racine, "Téléchargements", "film.mkv"), film, 0o644)

	telech := cibleDossier(t, s, 0, "Téléchargements")
	jeux := cibleDossier(t, s, 0, "Jeux")
	vieux := cibleDossier(t, s, jeux.ID, "vieux")
	systeme := cibleDossier(t, s, 0, "Systeme")
	cibles := []scan.Cible{vieux, {Type: "fichier", ID: telech.ID, Nom: "film.mkv", Taille: 5000}, systeme}

	ap, err := ApercevoirArchive(s, cibles, archives, "")
	if err != nil {
		t.Fatal(err)
	}
	if ap.Archive == "" || ap.Nombre != 3 || ap.Taille != 7900 || len(ap.Gardes) != 1 || len(ap.Ecartes) != 0 || ap.Manque != 0 {
		t.Fatalf("aperçu : %+v", ap)
	}
	if ap.Besoin < ap.Taille {
		t.Errorf("place comptée trop juste : %d pour %d", ap.Besoin, ap.Taille)
	}

	a, err := Archiver(s, cibles, archives, OptionsArchive{Nom: "ménage", Corbeille: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	b := a.Attendre()
	if b.Erreur != "" || !b.Verifie || b.Faits != 3 || b.Taille != 7900 || b.Recycles != 2 || b.Libere != 7000 || len(b.Gardes) != 1 || len(b.Sautes) != 0 {
		t.Fatalf("bilan : %+v", b)
	}
	if b.Archive != filepath.Join(archives, "ménage.zip") || b.Compresse <= 0 {
		t.Fatalf("archive : %s (%d octets)", b.Archive, b.Compresse)
	}
	if existe(b.Archive + suffixePart) {
		t.Error("le fichier provisoire est resté")
	}

	z := contenuZip(t, b.Archive)
	for _, nom := range []string{"vieux/", "vieux/data.pak", "film.mkv", "Systeme/", "Systeme/noyau.sys"} {
		if _, ok := z[nom]; !ok {
			t.Errorf("%s absent de l'archive (contenu : %v)", nom, len(z))
		}
	}
	if !bytes.Equal(z["film.mkv"], film) || len(z["vieux/data.pak"]) != 2000 {
		t.Error("le contenu archivé ne correspond pas")
	}

	// Recyclés : tout sauf le protégé.
	if existe(filepath.Join(racine, "Jeux", "vieux")) || existe(filepath.Join(racine, "Téléchargements", "film.mkv")) {
		t.Error("les originaux auraient dû partir à la corbeille")
	}
	if dans, _ := os.ReadDir(DossierCorbeille); len(dans) != 2 {
		t.Errorf("%d élément(s) dans la corbeille au lieu de 2", len(dans))
	}
	if !existe(filepath.Join(racine, "Systeme", "noyau.sys")) {
		t.Fatal("un élément protégé a été recyclé !")
	}
	if r, _ := s.RacineEntree(); r.Taille != 1200 {
		t.Errorf("l'analyse n'a pas suivi : %+v", r)
	}
}

func TestArchiverSansToucherAuxOriginaux(t *testing.T) {
	racine, s := bac(t)
	telech := cibleDossier(t, s, 0, "Téléchargements")
	dest := filepath.Join(racine, "Jeux") // dans l'analyse

	var noms []string
	for range 2 {
		a, err := Archiver(s, []scan.Cible{telech}, dest, OptionsArchive{}, 0)
		if err != nil {
			t.Fatal(err)
		}
		b := a.Attendre()
		if b.Erreur != "" || !b.Verifie || b.Recycles != 0 || !b.Perime {
			t.Fatalf("bilan : %+v", b)
		}
		noms = append(noms, filepath.Base(b.Archive))
	}
	if noms[0] != "Téléchargements.zip" || noms[1] != "Téléchargements (2).zip" {
		t.Errorf("noms : %v", noms)
	}
	z := contenuZip(t, filepath.Join(dest, noms[0]))
	if len(z) != 3 || len(z["Téléchargements/film.mkv"]) != 5000 || len(z["Téléchargements/setup.exe"]) != 300 {
		t.Errorf("contenu : %d entrées", len(z))
	}
	if !existe(filepath.Join(racine, "Téléchargements", "film.mkv")) {
		t.Error("sans l'option, les originaux restent")
	}
	if r, _ := s.RacineEntree(); r.Taille != 8200 {
		t.Errorf("l'analyse a changé : %+v", r)
	}
}

func TestArchiverRefus(t *testing.T) {
	racine, s := bac(t)
	jeux := cibleDossier(t, s, 0, "Jeux")

	// Une archive rangée dans le dossier qu'elle contient
	ap, err := ApercevoirArchive(s, []scan.Cible{jeux}, filepath.Join(racine, "Jeux", "vieux"), "")
	if err != nil || ap.Nombre != 0 || len(ap.Ecartes) != 1 || ap.Ecartes[0].Raison != "l'archive serait rangée dans ce dossier" {
		t.Fatalf("aperçu : %+v, %v", ap, err)
	}
	if _, err := Archiver(s, []scan.Cible{jeux}, filepath.Join(racine, "Jeux", "vieux"), OptionsArchive{}, 0); err == nil {
		t.Error("archive lancée sans rien à archiver")
	}
	// Nom invalide, destination système
	if _, err := ApercevoirArchive(s, []scan.Cible{jeux}, filepath.Dir(racine), "a:b"); err == nil {
		t.Error("nom invalide accepté")
	}
	if _, err := ApercevoirArchive(s, []scan.Cible{jeux}, filepath.Join(racine, "Systeme"), ""); err == nil {
		t.Error("archive déposée dans un dossier protégé")
	}

	// Pas assez de place
	petit := filepath.Join(filepath.Dir(racine), "petit")
	os.Mkdir(petit, 0o755)
	volumeDe = func(p string) (volumes.Volume, bool) {
		if dedans(p, petit) {
			return volumes.Volume{Chemin: petit, Type: "amovible", Total: 4000, Libre: 1000}, true
		}
		return volumes.Volume{Chemin: "/", Type: "fixe", Total: 1 << 40, Libre: 1 << 39}, true
	}
	t.Cleanup(func() { volumeDe = volumes.De })
	if ap, _ := ApercevoirArchive(s, []scan.Cible{jeux}, petit, ""); ap.Manque <= 0 {
		t.Errorf("place manquante non vue : %+v", ap)
	}
	if _, err := Archiver(s, []scan.Cible{jeux}, petit, OptionsArchive{}, 0); err == nil {
		t.Error("archive lancée sans la place nécessaire")
	}
	if entrees, _ := os.ReadDir(petit); len(entrees) != 0 {
		t.Error("rien n'aurait dû être écrit")
	}
}

func TestArchiverAnnule(t *testing.T) {
	racine, s := bac(t)
	archives := dossierArchives(t, racine)
	telech := cibleDossier(t, s, 0, "Téléchargements")

	attente := make(chan struct{})
	pointTest = func(phase string) {
		if phase == PhaseEcriture {
			<-attente
		}
	}
	t.Cleanup(func() { pointTest = func(string) {} })

	a, err := Archiver(s, []scan.Cible{telech}, archives, OptionsArchive{Corbeille: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if su := a.Suivi(); su.Phase != PhaseEcriture || su.Octets != 5300 || su.Fichiers != 2 {
		t.Errorf("suivi : %+v", su)
	}
	a.Annuler()
	close(attente)
	b := a.Attendre()
	if !b.Annule || b.Verifie || b.Recycles != 0 || b.Archive != "" {
		t.Fatalf("bilan : %+v", b)
	}
	if entrees, _ := os.ReadDir(archives); len(entrees) != 0 {
		t.Errorf("l'annulation a laissé %d fichier(s)", len(entrees))
	}
	if !existe(filepath.Join(racine, "Téléchargements", "film.mkv")) {
		t.Fatal("annulé : les originaux doivent rester")
	}
	// Le verrou est rendu : on peut relancer.
	if a, err := Archiver(s, []scan.Cible{telech}, archives, OptionsArchive{}, 0); err != nil {
		t.Errorf("relance impossible : %v", err)
	} else if b := a.Attendre(); !b.Verifie {
		t.Errorf("relance : %+v", b)
	}
}

func TestArchiverGardeCeQuiChangeOuManque(t *testing.T) {
	racine, s := bac(t)
	archives := dossierArchives(t, racine)
	telech := cibleDossier(t, s, 0, "Téléchargements")
	jeux := cibleDossier(t, s, 0, "Jeux")
	// Un lien dans Jeux : non archivé, donc Jeux ne doit pas partir à la corbeille.
	if err := os.Symlink(filepath.Join(racine, "Systeme"), filepath.Join(racine, "Jeux", "raccourci")); err != nil {
		t.Skip("liens symboliques indisponibles :", err)
	}
	// Le film change entre l'archivage et la corbeille.
	pointTest = func(phase string) {
		if phase == PhaseCorbeille {
			os.WriteFile(filepath.Join(racine, "Téléchargements", "film.mkv"), []byte("modifié"), 0o644)
		}
	}
	t.Cleanup(func() { pointTest = func(string) {} })

	a, err := Archiver(s, []scan.Cible{telech, jeux}, archives, OptionsArchive{Corbeille: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	b := a.Attendre()
	if !b.Verifie || len(b.Sautes) != 1 || b.Recycles != 0 || len(b.Gardes) != 2 {
		t.Fatalf("bilan : %+v", b)
	}
	raisons := map[string]string{}
	for _, g := range b.Gardes {
		raisons[filepath.Base(g.Chemin)] = g.Raison
	}
	if raisons["Jeux"] != "contient des fichiers qui n'ont pas pu être archivés" || raisons["Téléchargements"] != "a changé pendant l'archivage" {
		t.Errorf("raisons : %v", raisons)
	}
	if !existe(filepath.Join(racine, "Jeux", "vieux", "data.pak")) || !existe(filepath.Join(racine, "Téléchargements", "setup.exe")) {
		t.Fatal("des originaux incomplets dans l'archive sont partis !")
	}
}
