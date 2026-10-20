//go:build !windows

package volumes

import "syscall"

// Lister renvoie la racine du système (base de la future version Linux).
func Lister() []Volume {
	var st syscall.Statfs_t
	if err := syscall.Statfs("/", &st); err != nil {
		return nil
	}
	return []Volume{{
		Chemin: "/",
		Nom:    "Système",
		Type:   "fixe",
		Total:  int64(st.Blocks) * int64(st.Bsize),
		Libre:  int64(st.Bavail) * int64(st.Bsize),
	}}
}
