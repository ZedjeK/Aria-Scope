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

// Locaux renvoie les disques analysés par « Tout le PC » : internes et
// externes (clés USB, disques USB), sans les lecteurs réseau ni optiques.
func Locaux() []Volume {
	var out []Volume
	for _, v := range Lister() {
		if v.Type == "fixe" || v.Type == "amovible" {
			out = append(out, v)
		}
	}
	return out
}

// De renvoie le volume qui contient chemin (le plus long point de montage
// correspondant), pour savoir par exemple s'il est amovible.
func De(chemin string) (Volume, bool) {
	p := normaliser(chemin)
	var trouve Volume
	long := -1
	for _, v := range Lister() {
		r := normaliser(v.Chemin)
		sep := string(filepath.Separator)
		if (p == r || strings.HasPrefix(p, strings.TrimSuffix(r, sep)+sep)) && len(r) > long {
			trouve, long = v, len(r)
		}
	}
	return trouve, long >= 0
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
	// « C: » seul désigne le dossier courant du lecteur (Clean le transforme
	// en « C:. ») : on le complète en racine avant de nettoyer.
	if len(p) == 2 && p[1] == ':' {
		p += string(filepath.Separator)
	}
	return strings.ToUpper(filepath.Clean(p))
}
