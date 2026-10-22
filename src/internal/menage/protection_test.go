package menage

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestRaison(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chemins Unix")
	}
	r := Regles{
		Arbres:     []string{"/systeme"},
		Exacts:     []string{"/home/jeremy"},
		NomsRacine: []string{"pagefile.sys"},
	}
	cas := map[string]bool{
		"/":                            true,
		"/systeme":                     true,
		"/systeme/sous/fichier":        true,
		"/systemeX/autre":              false, // même début de nom, autre dossier
		"/home/jeremy":                 true,
		"/home/jeremy/Téléchargements": false,
		"/pagefile.sys":                true,
		"/$Recycle.Bin":                true,
		"/donnees/pagefile.sys":        false,
		"relatif/chemin":               true,
	}
	for chemin, protege := range cas {
		if got := r.Raison(chemin) != ""; got != protege {
			t.Errorf("%s : protégé = %v, attendu %v (%q)", chemin, got, protege, r.Raison(chemin))
		}
	}
}

func TestReglesSysteme(t *testing.T) {
	r := ReglesSysteme()
	if runtime.GOOS == "windows" {
		for _, p := range []string{`C:\`, `C:\Windows`, `c:\windows\system32\drivers\etc\hosts`, `C:\Program Files\App`, `C:\pagefile.sys`, `C:\Users`} {
			if r.Raison(p) == "" {
				t.Errorf("%s devrait être protégé", p)
			}
		}
		for _, p := range []string{`C:\Users\Public\Documents\x.txt`, `D:\Jeux\vieux`, `C:\Temp\gros.iso`} {
			if raison := r.Raison(p); raison != "" {
				t.Errorf("%s ne devrait pas être protégé : %s", p, raison)
			}
		}
		return
	}
	if r.Raison("/usr/lib/x") == "" || r.Raison(filepath.Join(t.TempDir(), "x")) != "" {
		t.Error("règles Unix incohérentes")
	}
}
