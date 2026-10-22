package menage

import (
	"cmp"
	"errors"
	"path/filepath"
	"slices"
	"sync"

	"aria-scope/internal/scan"
	"aria-scope/internal/volumes"
)

// Une seule opération de ménage à la fois.
var opMu sync.Mutex

// Refus : un élément laissé de côté, et pourquoi.
type Refus struct {
	Chemin string `json:"chemin"`
	Raison string `json:"raison"`
}

// ElementApercu : une ligne du récapitulatif avant confirmation.
type ElementApercu struct {
	Chemin string `json:"chemin"`
	Type   string `json:"type"`
	Taille int64  `json:"taille"`
	Disque int64  `json:"disque"`
}

// Apercu : ce qu'une opération va toucher, présenté avant confirmation.
type Apercu struct {
	Elements  []ElementApercu `json:"elements"` // les 100 plus gros
	Nombre    int             `json:"nombre"`
	Fichiers  int64           `json:"fichiers"`
	Taille    int64           `json:"taille"`
	Disque    int64           `json:"disque"`
	Proteges  []Refus         `json:"proteges"`  // ne seront pas touchés
	Amovibles []string        `json:"amovibles"` // disques où Windows n'a souvent pas de corbeille
	Filtre    bool            `json:"filtre"`
}

// Bilan : le résultat d'une opération.
type Bilan struct {
	Retires  int     `json:"retires"`
	Taille   int64   `json:"taille"`
	Disque   int64   `json:"disque"`
	Restants []Refus `json:"restants"` // toujours là : refusés, protégés ou annulés
	Annule   bool    `json:"annule"`
	Erreur   string  `json:"erreur"`
}

const maxApercu = 100

// Protections : ce que le ménage refuse de toucher (remplaçable dans les tests).
var Protections = ReglesSysteme()

// examiner sépare les cibles autorisées des cibles protégées.
func examiner(sc *scan.Scan, cibles []scan.Cible) (ok []scan.Cible, chemins []string, refus []Refus, err error) {
	for _, c := range cibles {
		chemin, err := sc.Chemin(c.ID)
		if err != nil {
			return nil, nil, nil, err
		}
		if c.Type == "fichier" {
			if c.Nom == "" || filepath.Base(c.Nom) != c.Nom || c.Nom == ".." {
				return nil, nil, nil, errors.New("nom de fichier invalide : " + c.Nom)
			}
			chemin = filepath.Join(chemin, c.Nom)
		}
		if raison := Protections.Raison(chemin); raison != "" {
			refus = append(refus, Refus{Chemin: chemin, Raison: raison})
			continue
		}
		ok = append(ok, c)
		chemins = append(chemins, chemin)
	}
	return ok, chemins, refus, nil
}

// ApercuCorbeille récapitule ce qu'enverrait la corbeille, sans rien toucher.
func ApercuCorbeille(sc *scan.Scan, cibles []scan.Cible) (Apercu, error) {
	resume, err := sc.Resumer(cibles)
	if err != nil {
		return Apercu{}, err
	}
	ok, chemins, refus, err := examiner(sc, resume.Cibles)
	if err != nil {
		return Apercu{}, err
	}
	// Listes jamais nulles : l'interface reçoit [] et non null.
	a := Apercu{Elements: []ElementApercu{}, Proteges: append([]Refus{}, refus...), Amovibles: []string{}, Filtre: resume.Filtre}
	for i, c := range ok {
		r, err := sc.Resumer([]scan.Cible{c})
		if err != nil {
			return Apercu{}, err
		}
		a.Nombre++
		a.Fichiers += r.Fichiers
		a.Taille += r.Taille
		a.Disque += r.Disque
		a.Elements = append(a.Elements, ElementApercu{Chemin: chemins[i], Type: c.Type, Taille: r.Taille, Disque: r.Disque})
		if v, trouve := volumes.De(chemins[i]); trouve && v.Type == "amovible" && !slices.Contains(a.Amovibles, v.Chemin) {
			a.Amovibles = append(a.Amovibles, v.Chemin)
		}
	}
	slices.SortStableFunc(a.Elements, func(x, y ElementApercu) int { return cmp.Compare(y.Taille, x.Taille) })
	if len(a.Elements) > maxApercu {
		a.Elements = a.Elements[:maxApercu]
	}
	return a, nil
}

// Corbeille envoie les cibles à la corbeille de Windows, puis met l'analyse à
// jour pour ce qui a vraiment quitté le disque. Les éléments protégés ne sont
// jamais touchés.
// hwnd : la fenêtre d'Aria, pour y rattacher les boîtes de Windows.
func EnvoyerCorbeille(sc *scan.Scan, cibles []scan.Cible, hwnd uintptr) (Bilan, error) {
	if !opMu.TryLock() {
		return Bilan{}, errors.New("une opération de ménage est déjà en cours")
	}
	defer opMu.Unlock()

	resume, err := sc.Resumer(cibles)
	if err != nil {
		return Bilan{}, err
	}
	ok, _, refus, err := examiner(sc, resume.Cibles)
	if err != nil {
		return Bilan{}, err
	}
	b := Bilan{Restants: append([]Refus{}, refus...)}
	if len(ok) == 0 {
		return b, nil
	}
	retraits, err := sc.PreparerRetrait(ok)
	if err != nil {
		return Bilan{}, err
	}
	chemins := make([]string, len(retraits))
	for i, r := range retraits {
		chemins[i] = r.Chemin
	}

	b.Annule, err = Corbeille(chemins, hwnd)
	if err != nil {
		b.Erreur = err.Error()
	}
	faits := sc.Retirer(retraits)
	partis := map[string]bool{}
	for _, f := range faits {
		b.Retires++
		b.Taille += f.Taille
		b.Disque += f.Disque
		partis[f.Chemin] = true
	}
	for _, r := range retraits {
		if !partis[r.Chemin] {
			raison := "toujours présent : Windows l'a refusé ou l'opération a été interrompue"
			if b.Annule {
				raison = "opération interrompue avant lui"
			}
			b.Restants = append(b.Restants, Refus{Chemin: r.Chemin, Raison: raison})
		}
	}
	return b, nil
}
