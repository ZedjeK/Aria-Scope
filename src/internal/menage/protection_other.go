//go:build !windows

package menage

import "os"

// ReglesSysteme : les protections d'un système Unix (base de la future version Linux).
func ReglesSysteme() Regles {
	r := Regles{
		Arbres: []string{"/bin", "/boot", "/dev", "/etc", "/lib", "/lib32", "/lib64", "/proc", "/run", "/sbin", "/sys", "/usr", "/var"},
		Exacts: []string{"/home", "/root"},
	}
	if h, err := os.UserHomeDir(); err == nil {
		r.Exacts = append(r.Exacts, h)
	}
	return r
}
