package scan

import (
	"container/heap"
	"sync"
	"sync/atomic"
)

// NombreGros : combien de plus gros fichiers on retient pendant le scan.
const NombreGros = 1000

type gros struct {
	dossier *Dossier
	nom     string
	taille  int64
	disque  int64
	modif   int64
}

// palmares garde les NombreGros plus gros fichiers (taille réelle) vus pendant
// le scan. Le seuil atomique évite de prendre le verrou pour l'immense
// majorité des fichiers, trop petits pour entrer dans le classement.
type palmares struct {
	mu    sync.Mutex
	tas   tasMin
	seuil atomic.Int64 // taille du plus petit retenu, une fois le palmarès plein
}

func (p *palmares) proposer(d *Dossier, e entree) {
	if e.Taille <= p.seuil.Load() {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.tas) < NombreGros {
		heap.Push(&p.tas, gros{d, e.Nom, e.Taille, e.Disque, e.Modif})
		if len(p.tas) == NombreGros {
			p.seuil.Store(p.tas[0].taille)
		}
		return
	}
	if e.Taille <= p.tas[0].taille {
		return
	}
	p.tas[0] = gros{d, e.Nom, e.Taille, e.Disque, e.Modif}
	heap.Fix(&p.tas, 0)
	p.seuil.Store(p.tas[0].taille)
}

// tasMin : le plus petit des retenus en tête, pour l'évincer vite.
type tasMin []gros

func (t tasMin) Len() int           { return len(t) }
func (t tasMin) Less(i, j int) bool { return t[i].taille < t[j].taille }
func (t tasMin) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t *tasMin) Push(x any)        { *t = append(*t, x.(gros)) }
func (t *tasMin) Pop() any {
	old := *t
	x := old[len(old)-1]
	*t = old[:len(old)-1]
	return x
}

// GrosFichier est une ligne de la liste des plus gros fichiers.
type GrosFichier struct {
	Entree
	Dossier string `json:"dossier"` // chemin du dossier qui le contient
}

// GrosFichiers renvoie les plus gros fichiers de l'analyse, triés selon t.
// L'ID de chaque ligne est celui de son dossier, comme pour les fichiers de l'arbre.
func (s *Scan) GrosFichiers(mode string, t Tri) ([]GrosFichier, error) {
	if _, err := s.noeud(0); err != nil {
		return nil, err
	}
	s.gros.mu.Lock()
	lignes := make([]Entree, 0, len(s.gros.tas))
	for _, g := range s.gros.tas {
		lignes = append(lignes, Entree{ID: g.dossier.id, Type: "fichier", Nom: g.nom, Taille: g.taille, Disque: g.disque, Modif: g.modif, Fichiers: 1})
	}
	s.gros.mu.Unlock()

	t.trier(lignes, mode)
	chemins := map[int32]string{}
	out := make([]GrosFichier, len(lignes))
	for i, l := range lignes {
		c, ok := chemins[l.ID]
		if !ok {
			c, _ = s.Chemin(l.ID)
			chemins[l.ID] = c
		}
		out[i] = GrosFichier{Entree: l, Dossier: c}
	}
	return out, nil
}
