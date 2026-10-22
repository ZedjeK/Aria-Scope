//go:build windows

package menage

import (
	"os"
	"path/filepath"
)

// ReglesSysteme : les protections de Windows.
func ReglesSysteme() Regles {
	env := func(nom, defaut string) string {
		if v := os.Getenv(nom); v != "" {
			return v
		}
		return defaut
	}
	systeme := env("SystemDrive", "C:") + `\`
	profil := os.Getenv("USERPROFILE")
	r := Regles{
		Arbres: []string{
			env("SystemRoot", `C:\Windows`),
			env("ProgramFiles", `C:\Program Files`),
			env("ProgramFiles(x86)", `C:\Program Files (x86)`),
			env("ProgramW6432", ""),
			env("ProgramData", `C:\ProgramData`),
			filepath.Join(systeme, "Recovery"),
			filepath.Join(systeme, "Boot"),
			filepath.Join(systeme, "EFI"),
		},
		Exacts: []string{
			filepath.Join(systeme, "Users"),
			filepath.Join(systeme, "Users", "Public"),
			filepath.Join(systeme, "Users", "Default"),
		},
		NomsRacine: []string{
			"pagefile.sys", "hiberfil.sys", "swapfile.sys", "DumpStack.log", "DumpStack.log.tmp",
			"bootmgr", "BOOTNXT", "System Volume Information", "Config.Msi", "Documents and Settings",
		},
	}
	if profil != "" {
		r.Exacts = append(r.Exacts, profil,
			filepath.Join(profil, "AppData"),
			filepath.Join(profil, "AppData", "Local"),
			filepath.Join(profil, "AppData", "LocalLow"),
			filepath.Join(profil, "AppData", "Roaming"),
		)
	}
	return r
}
