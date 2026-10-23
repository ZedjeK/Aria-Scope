//go:build !windows

package menage

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"aria-scope/internal/journal"
	"aria-scope/internal/scan"
)

func entreeDe(t *testing.T, b Bilan) journal.Entree {
	t.Helper()
	e, ok := b.Entree()
	if !ok {
		t.Fatalf("rien à inscrire au journal : %+v", b)
	}
	return e
}

func TestAnnulerDeplacement(t *testing.T) {
	racine, s := bac(t)
	archives := dossierArchives(t, racine)
	telech := cibleDossier(t, s, 0, "Téléchargements")
	jeux := cibleDossier(t, s, 0, "Jeux")
	vieux := cibleDossier(t, s, jeux.ID, "vieux")
	film := scan.Cible{Type: "fichier", ID: telech.ID, Nom: "film.mkv"}
	setup := scan.Cible{Type: "fichier", ID: telech.ID, Nom: "setup.exe"}

	b, err := Executer(s, OpDeplacer, []scan.Cible{vieux, film, setup}, archives, 0)
	if err != nil || b.Faits != 3 {
		t.Fatalf("déplacement : %+v, %v", b, err)
	}
	e := entreeDe(t, b)
	if len(e.Elements) != 3 || e.Elements[0].Arrivee != filepath.Join(archives, e.Elements[0].Origine[len(filepath.Dir(e.Elements[0].Origine))+1:]) {
		t.Fatalf("traces : %+v", e.Elements)
	}
	// Entre-temps, un nouveau setup.exe a pris la place de l'ancien.
	ecrire(t, filepath.Join(racine, "Téléchargements", "setup.exe"), 10)
	// Et le dossier d'origine de « vieux » a été supprimé : il sera recréé.
	os.Remove(filepath.Join(racine, "Jeux"))

	a := ApercevoirAnnulation(e)
	if a.Impossible != "" || len(a.Lignes) != 2 || len(a.Ecartes) != 1 || a.Taille != 7000 {
		t.Fatalf("aperçu : %+v", a)
	}
	ba, err := Annuler(e, 0)
	if err != nil || ba.Faits != 2 || len(ba.Restants) != 1 || ba.Resume != "2 éléments remis en place · 1 laissé de côté" {
		t.Fatalf("annulation : %+v, %v", ba, err)
	}
	if !existe(filepath.Join(racine, "Jeux", "vieux", "data.pak")) || !existe(filepath.Join(racine, "Téléchargements", "film.mkv")) {
		t.Error("les éléments ne sont pas revenus")
	}
	if existe(filepath.Join(archives, "vieux")) || !existe(filepath.Join(archives, "setup.exe")) {
		t.Error("l'arrivée n'est pas dans l'état attendu")
	}
	if info, _ := os.Stat(filepath.Join(racine, "Téléchargements", "setup.exe")); info.Size() != 10 {
		t.Fatal("le nouveau fichier a été écrasé !")
	}
}

func TestAnnulerCopie(t *testing.T) {
	racine, s := bac(t)
	archives := dossierArchives(t, racine)
	telech := cibleDossier(t, s, 0, "Téléchargements")
	b, err := Executer(s, OpCopier, []scan.Cible{telech}, archives, 0)
	if err != nil || b.Faits != 1 {
		t.Fatalf("copie : %+v, %v", b, err)
	}
	e := entreeDe(t, b)
	ba, err := Annuler(e, 0)
	if err != nil || ba.Faits != 1 || ba.Resume != "1 copie envoyée à la corbeille" {
		t.Fatalf("annulation : %+v, %v", ba, err)
	}
	if existe(filepath.Join(archives, "Téléchargements")) {
		t.Error("la copie est toujours là")
	}
	if dans, _ := os.ReadDir(DossierCorbeille); len(dans) != 1 {
		t.Error("la copie doit être dans la corbeille, pas supprimée")
	}
	if !existe(filepath.Join(racine, "Téléchargements", "film.mkv")) {
		t.Fatal("l'original a disparu !")
	}
}

func TestAnnulerArchiveRestaure(t *testing.T) {
	racine, s := bac(t)
	archives := dossierArchives(t, racine)
	// Un contenu reconnaissable, une date ancienne, un sous-dossier vide
	data := make([]byte, 2000)
	rand.Read(data)
	pak := filepath.Join(racine, "Jeux", "vieux", "data.pak")
	os.WriteFile(pak, data, 0o644)
	ancien := time.Date(2019, 3, 14, 15, 9, 26, 0, time.Local)
	os.Chtimes(pak, ancien, ancien)
	os.Mkdir(filepath.Join(racine, "Jeux", "vieux", "vide"), 0o755)

	telech := cibleDossier(t, s, 0, "Téléchargements")
	jeux := cibleDossier(t, s, 0, "Jeux")
	vieux := cibleDossier(t, s, jeux.ID, "vieux")
	film := scan.Cible{Type: "fichier", ID: telech.ID, Nom: "film.mkv"}

	ar, err := Archiver(s, []scan.Cible{vieux, film}, archives, OptionsArchive{Corbeille: true}, 0)
	if err != nil {
		t.Fatal(err)
	}
	b := ar.Attendre()
	if b.Recycles != 2 {
		t.Fatalf("archive : %+v", b)
	}
	e := entreeDe(t, b)
	if e.Archive == "" || len(e.Elements) != 2 || !e.Elements[0].Recycle || e.Elements[0].Arrivee != "vieux" {
		t.Fatalf("entrée : %+v", e)
	}

	a := ApercevoirAnnulation(e)
	if a.Impossible != "" || len(a.Lignes) != 2 || !a.ArchiveRecyclee {
		t.Fatalf("aperçu : %+v", a)
	}
	ba, err := Annuler(e, 0)
	if err != nil || ba.Faits != 2 || !ba.ArchiveRecyclee || ba.Resume != "2 originaux restaurés, archive envoyée à la corbeille" {
		t.Fatalf("annulation : %+v, %v", ba, err)
	}
	if lu, _ := os.ReadFile(pak); !bytes.Equal(lu, data) {
		t.Error("contenu restauré différent")
	}
	if info, err := os.Stat(pak); err != nil || !info.ModTime().Equal(ancien) {
		t.Errorf("date non restaurée : %v", info.ModTime())
	}
	if !existe(filepath.Join(racine, "Jeux", "vieux", "vide")) || !existe(filepath.Join(racine, "Téléchargements", "film.mkv")) {
		t.Error("restauration incomplète")
	}
	if existe(e.Archive) {
		t.Error("l'archive devait partir à la corbeille")
	}
	if l, _ := filepath.Glob(filepath.Join(racine, "*", "*"+suffixeRestauration)); len(l) != 0 {
		t.Errorf("restes de restauration : %v", l)
	}
}

func TestAnnulerArchivePartielle(t *testing.T) {
	racine, s := bac(t)
	archives := dossierArchives(t, racine)
	telech := cibleDossier(t, s, 0, "Téléchargements")
	jeux := cibleDossier(t, s, 0, "Jeux")
	ar, _ := Archiver(s, []scan.Cible{telech, jeux}, archives, OptionsArchive{Corbeille: true}, 0)
	e := entreeDe(t, ar.Attendre())
	// Un nouveau dossier Jeux a été créé depuis.
	ecrire(t, filepath.Join(racine, "Jeux", "nouveau.sav"), 5)

	a := ApercevoirAnnulation(e)
	if len(a.Lignes) != 1 || len(a.Ecartes) != 1 || a.ArchiveRecyclee || a.ArchiveGardee == "" {
		t.Fatalf("aperçu : %+v", a)
	}
	ba, err := Annuler(e, 0)
	if err != nil || ba.Faits != 1 || ba.ArchiveRecyclee || !existe(e.Archive) {
		t.Fatalf("annulation : %+v, %v", ba, err)
	}
	if !existe(filepath.Join(racine, "Téléchargements", "film.mkv")) || existe(filepath.Join(racine, "Jeux", "vieux")) {
		t.Error("seul Téléchargements devait revenir")
	}
}

func TestAnnulerArchiveSimple(t *testing.T) {
	racine, s := bac(t)
	archives := dossierArchives(t, racine)
	telech := cibleDossier(t, s, 0, "Téléchargements")
	ar, _ := Archiver(s, []scan.Cible{telech}, archives, OptionsArchive{}, 0)
	e := entreeDe(t, ar.Attendre())
	ba, err := Annuler(e, 0)
	if err != nil || !ba.ArchiveRecyclee || ba.Resume != "archive envoyée à la corbeille" || existe(e.Archive) {
		t.Fatalf("annulation : %+v, %v", ba, err)
	}
	// Une seconde fois : l'archive n'existe plus.
	if a := ApercevoirAnnulation(e); a.Impossible == "" {
		t.Errorf("aperçu : %+v", a)
	}
}

func TestAnnulationImpossible(t *testing.T) {
	quand := time.Now()
	for nom, e := range map[string]journal.Entree{
		"corbeille":  {Operation: OpCorbeille, Elements: []journal.Element{{Origine: "/x"}}},
		"déjà faite": {Operation: OpCopier, Annulee: &quand},
		"inconnue":   {Operation: "effacer"},
		"disparus":   {Operation: OpDeplacer, Elements: []journal.Element{{Origine: "/nulle/part/a", Arrivee: "/nulle/part/b"}}},
	} {
		if a := ApercevoirAnnulation(e); a.Impossible == "" {
			t.Errorf("%s : aperçu possible %+v", nom, a)
		}
		if _, err := Annuler(e, 0); err == nil {
			t.Errorf("%s : annulation lancée", nom)
		}
	}
}

func TestRestaurerNomSuspect(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	zw.Create("vieux/")
	w, _ := zw.Create("vieux/../../evade.txt")
	w.Write([]byte("hors du dossier"))
	zw.Close()
	z, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil && z == nil {
		t.Fatal(err)
	}
	d := t.TempDir()
	vers := filepath.Join(d, "a", "vieux")
	if err := restaurer(z, "vieux", vers); err == nil {
		t.Fatal("un nom suspect a été accepté")
	}
	if existe(filepath.Join(d, "evade.txt")) || existe(vers) || existe(vers+suffixeRestauration) {
		t.Fatal("quelque chose a été écrit")
	}
}
