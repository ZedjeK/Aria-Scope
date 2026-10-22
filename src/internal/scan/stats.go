package scan

import (
	"path/filepath"
	"sync"
)

// Statistiques sur les fichiers : par extension et par âge. Pendant le scan,
// chaque dossier compte d'abord chez lui (statsLocales), puis fusionne d'un
// coup dans le total (stats) : un verrou par dossier, pas par fichier.

type compteur struct{ taille, disque, fichiers int64 }

func (c *compteur) ajouter(e entree) {
	c.taille += e.Taille
	c.disque += e.Disque
	c.fichiers++
}

func (c *compteur) cumuler(o compteur) {
	c.taille += o.taille
	c.disque += o.disque
	c.fichiers += o.fichiers
}

// ---------- Âge ----------

type tranche struct {
	nom   string
	jours int64 // limite haute exclue ; 0 : pas de limite
}

// Tranches d'âge d'après la date de dernière modification, au moment du scan.
var tranchesAge = [...]tranche{
	{"Moins d'un mois", 30},
	{"1 à 6 mois", 182},
	{"6 mois à 1 an", 365},
	{"1 à 2 ans", 730},
	{"2 à 5 ans", 1826},
	{"Plus de 5 ans", 0},
}

const (
	trancheInconnue = len(tranchesAge) // fichier sans date
	nTranches       = len(tranchesAge) + 1
)

// trancheDe range un fichier selon son âge à la date ref (secondes Unix).
// Une date dans le futur (horloge décalée) compte comme récente.
func trancheDe(modif, ref int64) int {
	if modif <= 0 {
		return trancheInconnue
	}
	jours := (ref - modif) / 86400
	for i, t := range tranchesAge {
		if t.jours == 0 || jours < t.jours {
			return i
		}
	}
	return len(tranchesAge) - 1
}

// ---------- Cumul ----------

type statsLocales struct {
	exts map[string]*compteur
	ages [nTranches]compteur
}

func (l *statsLocales) ajouter(e entree, ref int64) {
	if l.exts == nil {
		l.exts = map[string]*compteur{}
	}
	ext := extension(e.Nom)
	c := l.exts[ext]
	if c == nil {
		c = &compteur{}
		l.exts[ext] = c
	}
	c.ajouter(e)
	l.ages[trancheDe(e.Modif, ref)].ajouter(e)
}

type stats struct {
	mu   sync.Mutex
	exts map[string]compteur
	ages [nTranches]compteur
}

func (st *stats) fusionner(l *statsLocales) {
	if l == nil || len(l.exts) == 0 {
		return
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.exts == nil {
		st.exts = map[string]compteur{}
	}
	for k, v := range l.exts {
		c := st.exts[k]
		c.cumuler(*v)
		st.exts[k] = c
	}
	for i := range st.ages {
		st.ages[i].cumuler(l.ages[i])
	}
}

// statsDe donne à f les statistiques du dossier id. Pour la racine, ce sont
// celles relevées pendant le scan ; pour une branche, on la relit (et on garde
// le résultat, le temps de passer de l'onglet « types » à « âge »).
func (s *Scan) statsDe(id int32, f func(st *stats)) error {
	d, err := s.noeud(id)
	if err != nil {
		return err
	}
	if id == 0 {
		s.stats.mu.Lock()
		defer s.stats.mu.Unlock()
		f(&s.stats)
		return nil
	}

	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if s.cache == nil || s.cacheID != id {
		chemin, err := s.chemin(id)
		if err != nil {
			return err
		}
		st := &stats{}
		s.relireBranche(d, chemin, st.fusionner)
		s.cache, s.cacheID = st, id
	}
	f(s.cache)
	return nil
}

// relireBranche relit en parallèle tous les dossiers sous d (connus grâce au
// scan, sans redécouvrir l'arborescence) et confie les fichiers de chacun à fn.
func (s *Scan) relireBranche(d *Dossier, chemin string, fn func(*statsLocales)) {
	type tache struct {
		d      *Dossier
		chemin string
	}
	taches := make(chan tache, 256)
	var wg sync.WaitGroup
	for range Parallelisme() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range taches {
				var local statsLocales
				_ = listerDossier(t.chemin, func(e entree) {
					if !e.Dossier && !e.Lien && s.filtre.garderFichier(e) {
						local.ajouter(e, s.ref)
					}
				})
				fn(&local)
			}
		}()
	}
	pile := []tache{{d, chemin}}
	for len(pile) > 0 {
		t := pile[len(pile)-1]
		pile = pile[:len(pile)-1]
		taches <- t
		for _, c := range t.d.enfants {
			pile = append(pile, tache{c, filepath.Join(t.chemin, c.nom)})
		}
	}
	close(taches)
	wg.Wait()
}

// ---------- Résultat ----------

// Age est une tranche de la répartition par âge.
type Age struct {
	Nom      string `json:"nom"`
	Taille   int64  `json:"taille"`
	Disque   int64  `json:"disque"`
	Fichiers int64  `json:"fichiers"`
}

// Ages renvoie la répartition par âge du dossier id, de la plus récente à la
// plus ancienne tranche, puis les fichiers sans date.
func (s *Scan) Ages(id int32) ([]Age, error) {
	s.arbre.RLock()
	defer s.arbre.RUnlock()
	var out []Age
	err := s.statsDe(id, func(st *stats) {
		for i, c := range st.ages {
			nom := "Date inconnue"
			if i < len(tranchesAge) {
				nom = tranchesAge[i].nom
			} else if c.fichiers == 0 {
				continue
			}
			out = append(out, Age{Nom: nom, Taille: c.taille, Disque: c.disque, Fichiers: c.fichiers})
		}
	})
	return out, err
}
