package reglages

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReglages(t *testing.T) {
	d := t.TempDir()
	m := Dans(filepath.Join(d, "AriaScope"))

	if r := m.Lire(); !r.AnalyseAuto {
		t.Fatal("premier lancement : l'analyse automatique doit être active")
	}
	if _, err := m.Modifier(func(r *Reglages) { r.AnalyseAuto = false }); err != nil {
		t.Fatal(err)
	}
	if r := Dans(filepath.Join(d, "AriaScope")).Lire(); r.AnalyseAuto {
		t.Fatal("le réglage n'a pas été gardé")
	}

	// Fichier abîmé : on retombe sur les valeurs par défaut, sans planter.
	if err := os.WriteFile(filepath.Join(d, "AriaScope", "reglages.json"), []byte("{pas du json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if r := m.Lire(); !r.AnalyseAuto {
		t.Fatal("fichier illisible : valeurs par défaut attendues")
	}
}
