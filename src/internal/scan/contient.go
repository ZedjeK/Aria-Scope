package scan

import (
	"path/filepath"
	"runtime"
	"strings"
)

// Contient indique si chemin se trouve dans ce que couvre l'analyse : sous
// sa racine, ou sous l'un des disques pour « Ce PC ». Sert à prévenir qu'une
// copie ou un déplacement vers un dossier analysé n'y apparaîtra qu'après
// une nouvelle analyse.
func (s *Scan) Contient(chemin string) bool {
	s.arbre.RLock()
	defer s.arbre.RUnlock()
	racines := []string{s.racine}
	if s.pc && s.racineD != nil {
		racines = racines[:0]
		for _, d := range s.racineD.enfants {
			racines = append(racines, d.nom)
		}
	}
	p := comparable(chemin)
	sep := string(filepath.Separator)
	for _, r := range racines {
		r = comparable(r)
		if p == r || strings.HasPrefix(p, strings.TrimSuffix(r, sep)+sep) {
			return true
		}
	}
	return false
}

func comparable(p string) string {
	p = filepath.Clean(p)
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	return p
}
