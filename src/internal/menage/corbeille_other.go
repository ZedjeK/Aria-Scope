//go:build !windows

package menage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DossierCorbeille : où vont les éléments hors Windows (tests, banc d'essai,
// future version Linux). Vide : la corbeille freedesktop de l'utilisateur.
var DossierCorbeille = ""

// Corbeille déplace les chemins dans le dossier corbeille.
func Corbeille(chemins []string, _ uintptr) (annule bool, err error) {
	dossier := DossierCorbeille
	if dossier == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return false, err
		}
		dossier = filepath.Join(h, ".local", "share", "Trash", "files")
	}
	if err := os.MkdirAll(dossier, 0o700); err != nil {
		return false, err
	}
	var premiere error
	for i, c := range chemins {
		dest := filepath.Join(dossier, fmt.Sprintf("%d-%d-%s", time.Now().UnixNano(), i, filepath.Base(c)))
		if err := os.Rename(c, dest); err != nil && premiere == nil {
			premiere = err
		}
	}
	return false, premiere
}
