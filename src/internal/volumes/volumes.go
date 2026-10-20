// Package volumes liste les disques de la machine et leur remplissage.
package volumes

import (
	"path/filepath"
	"strings"
)

// Volume est un disque affiché sur l'accueil.
type Volume struct {
	Chemin  string `json:"chemin"`  // « C:\ »
	Nom     string `json:"nom"`     // étiquette du volume
	Systeme string `json:"systeme"` // NTFS, exFAT…
	Type    string `json:"type"`    // "fixe", "amovible", "reseau", "optique", "ram"
	Total   int64  `json:"total"`
	Libre   int64  `json:"libre"`
}

// Utilise renvoie l'espace occupé du volume dont racine est la racine, ou 0
// si racine n'est pas une racine de volume. Sert à estimer l'avancement d'un
// scan de disque entier.
func Utilise(racine string) int64 {
	r := normaliser(racine)
	for _, v := range Lister() {
		if normaliser(v.Chemin) == r {
			return v.Total - v.Libre
		}
	}
	return 0
}

func normaliser(p string) string {
	p = filepath.Clean(p)
	if len(p) == 2 && p[1] == ':' { // « C: » désigne le dossier courant, pas la racine
		p += string(filepath.Separator)
	}
	return strings.ToUpper(p)
}
