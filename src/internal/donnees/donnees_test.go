package donnees

import (
	"os"
	"path/filepath"
	"testing"
)

func preparer(t *testing.T) (home, conf string) {
	root := t.TempDir()
	home, conf = filepath.Join(root, "profil"), filepath.Join(root, "profil", ".config")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", conf)
	t.Setenv("APPDATA", conf)
	os.MkdirAll(conf, 0o755)
	return home, conf
}

func TestDossierDeplaceAncien(t *testing.T) {
	home, conf := preparer(t)
	ancien := filepath.Join(conf, ancienNom)
	os.MkdirAll(ancien, 0o755)
	os.WriteFile(filepath.Join(ancien, "journal.json"), []byte("[]"), 0o644)

	dir, deplace, err := Dossier()
	if err != nil || !deplace || dir != filepath.Join(home, Nom) {
		t.Fatalf("Dossier = %q, %v, %v", dir, deplace, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "journal.json")); err != nil {
		t.Error("journal non déplacé")
	}
	if _, err := os.Stat(ancien); !os.IsNotExist(err) {
		t.Error("l'ancien dossier doit avoir disparu")
	}
	if _, deplace, _ := Dossier(); deplace {
		t.Error("rien à déplacer au second lancement")
	}
}

func TestDossierGardeLeNouveau(t *testing.T) {
	home, conf := preparer(t)
	os.MkdirAll(filepath.Join(home, Nom), 0o755)
	os.WriteFile(filepath.Join(home, Nom, "reglages.json"), []byte("récent"), 0o644)
	os.MkdirAll(filepath.Join(conf, ancienNom), 0o755)
	os.WriteFile(filepath.Join(conf, ancienNom, "reglages.json"), []byte("ancien"), 0o644)

	dir, deplace, err := Dossier()
	if err != nil || deplace {
		t.Fatalf("deplace=%v err=%v", deplace, err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "reglages.json")); string(b) != "récent" {
		t.Error("le dossier .Aria-Scope existant ne doit pas être écrasé")
	}
}

func TestNettoyerAncienWebview(t *testing.T) {
	_, conf := preparer(t)
	vieux := filepath.Join(conf, "Aria-Scope.exe")
	os.MkdirAll(filepath.Join(vieux, "EBWebView", "Default"), 0o755)
	NettoyerAncienWebview(filepath.Join("outils", "Aria-Scope.exe"))
	if _, err := os.Stat(vieux); !os.IsNotExist(err) {
		t.Error("l'ancien profil WebView2 doit être supprimé")
	}

	// Un dossier qui contient autre chose n'est jamais touché.
	autre := filepath.Join(conf, "Autre.exe")
	os.MkdirAll(filepath.Join(autre, "EBWebView"), 0o755)
	os.WriteFile(filepath.Join(autre, "important.txt"), []byte("x"), 0o644)
	NettoyerAncienWebview(filepath.Join("outils", "Autre.exe"))
	if _, err := os.Stat(filepath.Join(autre, "important.txt")); err != nil {
		t.Error("un dossier qui n'est pas qu'un profil WebView2 ne doit pas être supprimé")
	}
}
