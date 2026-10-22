//go:build !windows

package menage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCorbeille(t *testing.T) {
	d := t.TempDir()
	DossierCorbeille = filepath.Join(d, "corbeille")
	t.Cleanup(func() { DossierCorbeille = "" })

	a := filepath.Join(d, "a.txt")
	b := filepath.Join(d, "dossier")
	os.WriteFile(a, []byte("x"), 0o644)
	os.MkdirAll(filepath.Join(b, "sous"), 0o755)

	if annule, err := Corbeille([]string{a, b}, 0); err != nil || annule {
		t.Fatalf("annulé %v, erreur %v", annule, err)
	}
	for _, p := range []string{a, b} {
		if _, err := os.Lstat(p); !os.IsNotExist(err) {
			t.Errorf("%s est toujours là", p)
		}
	}
	elements, _ := os.ReadDir(DossierCorbeille)
	if len(elements) != 2 {
		t.Errorf("%d éléments dans la corbeille, attendu 2", len(elements))
	}
}
