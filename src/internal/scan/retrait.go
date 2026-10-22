package scan

import (
	"container/heap"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// Retrait : un élément sur le point de quitter le disque (corbeille,
// déplacement), avec ce qu'il pèse dans l'analyse, relevé avant l'opération.
type Retrait struct {
	Cible  Cible  `json:"cible"`
	Chemin string `json:"chemin"`
	Taille int64  `json:"taille"`
	Disque int64  `json:"disque"`

	fichiers int64
	stats    *stats // fichiers par extension et par âge, à retrancher
}

// PreparerRetrait relève, pour chaque cible, son chemin et ce qu'elle pèse.
// Pour un dossier, sa branche est relue pour les statistiques par type et âge.
func (s *Scan) PreparerRetrait(cibles []Cible) ([]Retrait, error) {
	s.arbre.RLock()
	defer s.arbre.RUnlock()
	var out []Retrait
	for _, c := range cibles {
		d, err := s.noeud(c.ID)
		if err != nil {
			return nil, err
		}
		chemin, err := s.chemin(c.ID)
		if err != nil {
			return nil, err
		}
		r := Retrait{Cible: c, stats: &stats{}}
		switch c.Type {
		case "dossier":
			if c.ID == 0 {
				return nil, errors.New("la racine de l'analyse ne peut pas être retirée")
			}
			r.Chemin = chemin
			r.Taille, r.Disque, r.fichiers = d.taille, d.disque, d.fichiers
			s.relireBranche(d, chemin, r.stats.fusionner)
		case "fichier":
			r.Chemin = joindreNom(chemin, c.Nom)
			info, err := os.Lstat(r.Chemin)
			if err != nil {
				return nil, err
			}
			if info.IsDir() {
				return nil, errors.New(r.Chemin + " est un dossier")
			}
			e := entree{Nom: c.Nom, Taille: info.Size(), Disque: c.Disque, Modif: info.ModTime().Unix()}
			if e.Disque <= 0 {
				e.Disque = e.Taille
			}
			r.Taille, r.Disque = e.Taille, e.Disque
			if s.filtre.garderFichier(e) { // un fichier écarté par les filtres n'est pas compté
				r.fichiers = 1
				var l statsLocales
				l.ajouter(e, s.ref)
				r.stats.fusionner(&l)
			} else {
				r.Taille, r.Disque = 0, 0
			}
		default:
			return nil, errors.New("type d'élément inconnu : " + c.Type)
		}
		out = append(out, r)
	}
	return out, nil
}

// joindreNom ajoute un nom de fichier à un dossier en refusant tout ce qui
// ressemble à un chemin : une cible ne peut pas désigner un autre dossier.
func joindreNom(dossier, nom string) string {
	if nom == "" || nom == "." || nom == ".." || filepath.Base(nom) != nom || filepath.IsAbs(nom) {
		return filepath.Join(dossier, "\x00invalide")
	}
	return filepath.Join(dossier, nom)
}

// Retirer met l'analyse à jour pour les éléments qui ont bien disparu du
// disque, et renvoie ceux-là. Ceux qui sont toujours là ne changent rien.
// La date « modifié » des dossiers parents n'est pas recalculée.
func (s *Scan) Retirer(retraits []Retrait) []Retrait {
	s.arbre.Lock()
	defer s.arbre.Unlock()
	var faits []Retrait
	for _, r := range retraits {
		if _, err := os.Lstat(r.Chemin); !errors.Is(err, fs.ErrNotExist) {
			continue // toujours là (opération refusée ou annulée)
		}
		d, err := s.noeud(r.Cible.ID)
		if err != nil {
			continue // déjà retiré (par un dossier parent)
		}
		if r.Cible.Type == "fichier" {
			cle := cleCible(r.Cible)
			if s.fichiersRetires[cle] {
				continue // déjà retranché
			}
			if s.fichiersRetires == nil {
				s.fichiersRetires = map[string]bool{}
			}
			s.fichiersRetires[cle] = true
		}
		var dossiers, erreurs, liens int64
		depart := d
		if r.Cible.Type == "dossier" {
			depart = d.parent
			dossiers, erreurs, liens = d.dossiers+1, d.erreurs, d.liens
			d.retire = true
			p := d.parent
			for i, c := range p.enfants {
				if c == d {
					p.enfants = append(p.enfants[:i:i], p.enfants[i+1:]...)
					break
				}
			}
		} else if r.fichiers > 0 {
			d.nFichiers--
			d.fTaille -= r.Taille
			d.fDisque -= r.Disque
		}
		for a := depart; a != nil; a = a.parent {
			a.taille -= r.Taille
			a.disque -= r.Disque
			a.fichiers -= r.fichiers
			a.dossiers -= dossiers
			a.erreurs -= erreurs
			a.liens -= liens
		}
		s.fichiers.Add(-r.fichiers)
		s.taille.Add(-r.Taille)
		s.disque.Add(-r.Disque)
		s.dossiers.Add(-dossiers)
		s.stats.retrancher(r.stats)
		s.gros.retirer(d, r.Cible)
		faits = append(faits, r)
	}
	if len(faits) > 0 {
		s.cacheMu.Lock()
		s.cache = nil
		s.cacheMu.Unlock()
	}
	return faits
}

// retrancher enlève o des statistiques.
func (st *stats) retrancher(o *stats) {
	if o == nil {
		return
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	for k, v := range o.exts {
		c := st.exts[k]
		c.taille -= v.taille
		c.disque -= v.disque
		c.fichiers -= v.fichiers
		if c.fichiers <= 0 {
			delete(st.exts, k)
		} else {
			st.exts[k] = c
		}
	}
	for i := range st.ages {
		st.ages[i].taille -= o.ages[i].taille
		st.ages[i].disque -= o.ages[i].disque
		st.ages[i].fichiers -= o.ages[i].fichiers
	}
}

// retirer enlève du palmarès un fichier retiré, ou tous ceux d'un dossier retiré.
func (p *palmares) retirer(d *Dossier, c Cible) {
	p.mu.Lock()
	defer p.mu.Unlock()
	garde := p.tas[:0]
	for _, g := range p.tas {
		parti := false
		if c.Type == "fichier" {
			parti = g.dossier == d && g.nom == c.Nom
		} else {
			for a := g.dossier; a != nil; a = a.parent {
				if a == d {
					parti = true
					break
				}
			}
		}
		if !parti {
			garde = append(garde, g)
		}
	}
	clear(p.tas[len(garde):])
	p.tas = garde
	heap.Init(&p.tas)
	// Le palmarès n'est plus forcément complet : on ne le complète pas, mais
	// on rouvre la porte aux fichiers plus petits s'il y a de la place.
	if len(p.tas) < NombreGros {
		p.seuil.Store(0)
	}
}
