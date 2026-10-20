package scan

import (
	"cmp"
	"slices"
	"strings"
)

// Colonnes de tri.
const (
	TriAuto     = ""         // la taille du mode choisi (réelle ou sur disque)
	TriNom      = "nom"      // ordre alphabétique
	TriTaille   = "taille"   // taille réelle
	TriDisque   = "disque"   // taille sur disque
	TriFichiers = "fichiers" // nombre de fichiers
	TriDossiers = "dossiers" // nombre de sous-dossiers
)

// Tri décrit l'ordre demandé par l'interface. Par défaut, les tailles et les
// compteurs vont du plus grand au plus petit, les noms de A à Z ; Inverse
// renverse cet ordre.
type Tri struct {
	Colonne string `json:"colonne"`
	Inverse bool   `json:"inverse"`
}

// cleNom : la ligne « fichiers de ce dossier » n'a pas de nom ; on la range
// après les dossiers dans l'ordre alphabétique.
func cleNom(e Entree) string {
	if e.Type == "fichiers" {
		return "￿"
	}
	return strings.ToLower(e.Nom)
}

func (t Tri) trier(lignes []Entree, mode string) {
	col := t.Colonne
	if col == TriAuto {
		col = TriTaille
		if mode == ModeDisque {
			col = TriDisque
		}
	}
	var principal func(a, b Entree) int
	switch col {
	case TriNom:
		principal = func(a, b Entree) int { return cmp.Compare(cleNom(a), cleNom(b)) }
	case TriDisque:
		principal = func(a, b Entree) int { return cmp.Compare(b.Disque, a.Disque) }
	case TriFichiers:
		principal = func(a, b Entree) int { return cmp.Compare(b.Fichiers, a.Fichiers) }
	case TriDossiers:
		principal = func(a, b Entree) int { return cmp.Compare(b.Dossiers, a.Dossiers) }
	default:
		principal = func(a, b Entree) int { return cmp.Compare(b.Taille, a.Taille) }
	}
	slices.SortStableFunc(lignes, func(a, b Entree) int {
		c := principal(a, b)
		if t.Inverse {
			c = -c
		}
		if c != 0 {
			return c
		}
		// Égalité : le plus lourd d'abord, puis l'ordre alphabétique.
		if c := cmp.Compare(b.Taille, a.Taille); c != 0 {
			return c
		}
		return cmp.Compare(cleNom(a), cleNom(b))
	})
}
