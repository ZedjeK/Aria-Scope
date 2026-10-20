//go:build !windows

package scan

import (
	"io/fs"
	"os"
	"syscall"
)

// Version portable (Linux, macOS) : sert aux tests, et de base pour la future
// version Linux d'Aria Scope.
func lister(chemin string, fn func(e entree)) error {
	elems, err := os.ReadDir(chemin)
	if err != nil {
		return err
	}
	for _, d := range elems {
		e := entree{Nom: d.Name()}
		if d.Type()&fs.ModeSymlink != 0 {
			e.Lien = true
			fn(e)
			continue
		}
		info, err := d.Info() // Lstat : le lien lui-même, jamais sa cible
		if err != nil {
			continue // supprimé entre-temps
		}
		e.Dossier = info.IsDir()
		if !e.Dossier {
			e.Taille = info.Size()
			e.Disque = info.Size()
			if st, ok := info.Sys().(*syscall.Stat_t); ok {
				e.Disque = int64(st.Blocks) * 512
			}
		}
		fn(e)
	}
	return nil
}
