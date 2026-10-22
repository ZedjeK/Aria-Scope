// Package menage regroupe les opérations qui touchent aux fichiers : envoi à
// la corbeille, et les garde-fous qui les encadrent.
package menage

import (
	"path/filepath"
	"runtime"
	"strings"
)

// Regles décrit ce qu'Aria Scope refuse de toucher.
type Regles struct {
	// Arbres : dossiers protégés, avec tout ce qu'ils contiennent.
	Arbres []string
	// Exacts : dossiers protégés eux-mêmes (leur contenu reste accessible),
	// par exemple le profil de l'utilisateur ou C:\Users.
	Exacts []string
	// NomsRacine : éléments protégés à la racine d'un disque (pagefile.sys…).
	// Tout ce qui commence par « $ » à la racine est protégé aussi.
	NomsRacine []string
}

var sansCasse = runtime.GOOS == "windows"

func normaliser(p string) string {
	p = filepath.Clean(p)
	if sansCasse {
		p = strings.ToLower(p)
	}
	return p
}

// dans indique si p est d ou se trouve dans d.
func dans(p, d string) bool {
	if p == d {
		return true
	}
	sep := string(filepath.Separator)
	return strings.HasPrefix(p, strings.TrimSuffix(d, sep)+sep)
}

// estRacineDisque : « C:\ » sous Windows, « / » ailleurs.
func estRacineDisque(p string) bool {
	return filepath.Dir(p) == p
}

// Raison renvoie pourquoi chemin est protégé, ou "" s'il ne l'est pas.
func (r Regles) Raison(chemin string) string {
	if !filepath.IsAbs(chemin) {
		return "chemin incomplet"
	}
	p := normaliser(chemin)
	if estRacineDisque(p) {
		return "racine d'un disque"
	}
	for _, a := range r.Arbres {
		if a != "" && dans(p, normaliser(a)) {
			return "dossier système (" + a + ")"
		}
	}
	for _, e := range r.Exacts {
		if e != "" && p == normaliser(e) {
			return "dossier indispensable à Windows ou à ton profil"
		}
	}
	if parent := filepath.Dir(p); estRacineDisque(parent) {
		nom := filepath.Base(p)
		if strings.HasPrefix(nom, "$") {
			return "élément système à la racine du disque"
		}
		for _, n := range r.NomsRacine {
			if strings.EqualFold(nom, n) {
				return "élément système à la racine du disque"
			}
		}
	}
	return ""
}
