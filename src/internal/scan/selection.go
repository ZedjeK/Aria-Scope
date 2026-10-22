package scan

import "fmt"

// Cible est un élément sélectionné dans l'interface : un dossier (son id) ou
// un fichier (l'id de son dossier et son nom).
type Cible struct {
	Type   string `json:"type"` // "dossier" ou "fichier"
	ID     int32  `json:"id"`
	Nom    string `json:"nom"`    // fichier : son nom
	Taille int64  `json:"taille"` // fichier : tailles affichées dans la liste
	Disque int64  `json:"disque"`
}

// Resume récapitule une sélection, doublons écartés : un fichier ou un
// dossier contenu dans un dossier lui-même sélectionné ne compte qu'une fois.
type Resume struct {
	Elements int     `json:"elements"`
	Dossiers int     `json:"dossiers"`
	Fichiers int64   `json:"fichiers"` // fichiers concernés, contenu des dossiers compris
	Taille   int64   `json:"taille"`
	Disque   int64   `json:"disque"`
	Cibles   []Cible `json:"cibles"` // la sélection sans les doublons
	// Filtre : l'analyse est filtrée ; les tailles des dossiers ne comptent
	// que les fichiers retenus, alors qu'une opération emporte tout le dossier.
	Filtre bool `json:"filtre"`
}

func cleCible(c Cible) string {
	if c.Type == "fichier" {
		return fmt.Sprintf("f:%d/%s", c.ID, c.Nom)
	}
	return fmt.Sprintf("d:%d", c.ID)
}

// Resumer calcule le récapitulatif d'une sélection.
func (s *Scan) Resumer(cibles []Cible) (Resume, error) {
	r := Resume{Filtre: s.filtre.actif}
	if _, err := s.noeud(0); err != nil {
		return r, err
	}
	dossiers := map[int32]bool{}
	for _, c := range cibles {
		if c.Type == "dossier" {
			dossiers[c.ID] = true
		}
	}
	vus := map[string]bool{}
	for _, c := range cibles {
		if (c.Type != "dossier" && c.Type != "fichier") || vus[cleCible(c)] {
			continue
		}
		d, err := s.noeud(c.ID)
		if err != nil {
			return r, err
		}
		// Un ancêtre déjà sélectionné emporte cet élément avec lui.
		depart := d
		if c.Type == "dossier" {
			depart = d.parent
		}
		englobe := false
		for a := depart; a != nil; a = a.parent {
			if dossiers[a.id] {
				englobe = true
				break
			}
		}
		if englobe {
			continue
		}
		vus[cleCible(c)] = true
		r.Elements++
		r.Cibles = append(r.Cibles, c)
		if c.Type == "dossier" {
			r.Dossiers++
			r.Fichiers += d.fichiers
			r.Taille += d.taille
			r.Disque += d.disque
		} else {
			r.Fichiers++
			r.Taille += c.Taille
			r.Disque += c.Disque
		}
	}
	return r, nil
}
