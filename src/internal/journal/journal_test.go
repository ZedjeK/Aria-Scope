package journal

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestJournal(t *testing.T) {
	d := filepath.Join(t.TempDir(), "AriaScope")
	j := Dans(d)
	if len(j.Lister()) != 0 || j.Present() {
		t.Fatal("premier lancement : journal vide, aucun fichier")
	}

	jour := time.Date(2026, 10, 23, 9, 30, 0, 0, time.Local)
	id1, err := j.Ajouter(Entree{Date: jour, Operation: "deplacer", Destination: "/archives",
		Elements: []Element{{Origine: "/jeux/vieux", Arrivee: "/archives/vieux", Type: "dossier", Taille: 2000}}, Taille: 2000})
	if err != nil {
		t.Fatal(err)
	}
	id2, _ := j.Ajouter(Entree{Date: jour, Operation: "corbeille"}) // même instant
	if id1 == id2 {
		t.Fatalf("deux identifiants identiques : %s", id1)
	}

	// Relu par un autre lancement
	j2 := Dans(d)
	l := j2.Lister()
	if len(l) != 2 || l[0].ID != id2 || l[1].Elements[0].Arrivee != "/archives/vieux" || l[0].Elements == nil {
		t.Fatalf("relu : %+v", l)
	}
	quand := jour.Add(time.Hour)
	if err := j2.Marquer(id1, quand, "1 élément remis en place"); err != nil {
		t.Fatal(err)
	}
	if e, ok := Dans(d).Trouver(id1); !ok || e.Annulee == nil || !e.Annulee.Equal(quand) || e.Annulation == "" {
		t.Fatalf("annulation non gardée : %+v", e)
	}
	if err := j2.Marquer("inconnu", quand, ""); err == nil {
		t.Error("une opération inconnue a été marquée")
	}
}

func TestJournalLimite(t *testing.T) {
	j := Dans(t.TempDir())
	for range Max + 20 {
		if _, err := j.Ajouter(Entree{Operation: "copier"}); err != nil {
			t.Fatal(err)
		}
	}
	if n := len(j.Lister()); n != Max {
		t.Fatalf("%d entrées gardées au lieu de %d", n, Max)
	}
}

func TestJournalAbime(t *testing.T) {
	d := t.TempDir()
	chemin := filepath.Join(d, "journal.json")
	os.WriteFile(chemin, []byte("[{pas du json"), 0o600)
	j := Dans(d)
	if len(j.Lister()) != 0 {
		t.Fatal("un journal abîmé doit repartir à vide")
	}
	if _, err := os.Stat(chemin + ".abime"); err != nil {
		t.Fatal("le fichier abîmé doit être mis de côté, pas écrasé")
	}
	if _, err := j.Ajouter(Entree{Operation: "copier"}); err != nil || len(Dans(d).Lister()) != 1 {
		t.Fatal("le journal doit repartir normalement")
	}
}
