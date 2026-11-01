package scan

import (
	"cmp"
	"path/filepath"
	"slices"
)

// LigneArbre : un dossier de l'arbre, tel qu'on l'exporte.
type LigneArbre struct {
	Niveau   int    // 0 : le dossier de départ
	Chemin   string // chemin complet (« Ce PC » pour la racine d'une analyse de tout le PC)
	Nom      string
	Taille   int64
	Disque   int64
	Fichiers int64
	Dossiers int64
	Modif    int64   // fichier le plus récent de la branche, secondes Unix
	Erreurs  int64   // dossiers illisibles dans la branche
	Part     float64 // part de son parent (0 à 1), selon le mode
}

// ParcourirArbre passe à fn les dossiers sous id, en profondeur d'abord,
// les plus lourds en premier (comme l'arborescence affichée), jusqu'à
// profondeur niveaux sous le départ (0 : tous). L'arbre reste verrouillé en
// lecture pendant le parcours : fn doit être rapide (écrire une ligne).
func (s *Scan) ParcourirArbre(id int32, profondeur int, mode string, fn func(LigneArbre) error) error {
	s.arbre.RLock()
	defer s.arbre.RUnlock()
	depart, err := s.noeud(id)
	if err != nil {
		return err
	}
	chemin := s.racine
	if !(id == 0 && s.pc) {
		if chemin, err = s.chemin(id); err != nil {
			return err
		}
	}
	poids := func(d *Dossier) int64 {
		if mode == ModeDisque {
			return d.disque
		}
		return d.taille
	}
	var visiter func(d *Dossier, chemin string, niveau int, part float64) error
	visiter = func(d *Dossier, chemin string, niveau int, part float64) error {
		if err := fn(LigneArbre{
			Niveau: niveau, Chemin: chemin, Nom: d.nom,
			Taille: d.taille, Disque: d.disque, Fichiers: d.fichiers, Dossiers: d.dossiers,
			Modif: d.modif, Erreurs: d.erreurs, Part: part,
		}); err != nil {
			return err
		}
		if profondeur > 0 && niveau >= profondeur {
			return nil
		}
		enfants := slices.Clone(d.enfants)
		slices.SortStableFunc(enfants, func(a, b *Dossier) int {
			if c := cmp.Compare(poids(b), poids(a)); c != 0 {
				return c
			}
			return cmp.Compare(a.nom, b.nom)
		})
		for _, e := range enfants {
			if e.retire {
				continue
			}
			p := 0.0
			if total := poids(d); total > 0 {
				p = float64(poids(e)) / float64(total)
			}
			ce := filepath.Join(chemin, e.nom)
			if d == s.racineD && s.pc {
				ce = e.nom // un disque, directement sous « Ce PC »
			}
			if err := visiter(e, ce, niveau+1, p); err != nil {
				return err
			}
		}
		return nil
	}
	return visiter(depart, chemin, 0, 1)
}
