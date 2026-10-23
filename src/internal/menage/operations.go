package menage

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"aria-scope/internal/scan"
	"aria-scope/internal/volumes"
)

// Les opérations de ménage.
const (
	OpCorbeille = "corbeille"
	OpDeplacer  = "deplacer"
	OpCopier    = "copier"
)

// Une seule opération de ménage à la fois.
var opMu sync.Mutex

// Protections : ce que le ménage refuse de toucher (remplaçable dans les tests).
var Protections = ReglesSysteme()

// volumeDe : le disque d'un chemin (remplaçable dans les tests, pour simuler
// un disque plein).
var volumeDe = volumes.De

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

// Apercu : ce qu'une opération va faire, présenté avant confirmation.
type Apercu struct {
	Operation   string          `json:"operation"`
	Destination string          `json:"destination"`
	Elements    []ElementApercu `json:"elements"` // les 100 plus gros
	Nombre      int             `json:"nombre"`
	Fichiers    int64           `json:"fichiers"`
	Taille      int64           `json:"taille"`
	Disque      int64           `json:"disque"`
	Ecartes     []Refus         `json:"ecartes"`   // ne seront pas touchés, avec la raison
	Conflits    []string        `json:"conflits"`  // noms déjà pris à l'arrivée : Windows demandera
	Amovibles   []string        `json:"amovibles"` // corbeille : disques sans corbeille probable
	Filtre      bool            `json:"filtre"`
	// Place à l'arrivée (copie, ou déplacement vers un autre disque).
	Besoin int64 `json:"besoin"`
	Libre  int64 `json:"libre"`  // -1 : inconnue
	Manque int64 `json:"manque"` // > 0 : pas assez de place, l'opération est bloquée
	// DansAnalyse : l'arrivée est dans ce que couvre l'analyse ; ce qui y arrive
	// n'apparaîtra qu'après une nouvelle analyse.
	DansAnalyse bool `json:"dansAnalyse"`
}

// Bilan : le résultat d'une opération.
type Bilan struct {
	Operation   string  `json:"operation"`
	Destination string  `json:"destination"`
	Faits       int     `json:"faits"`
	Taille      int64   `json:"taille"`
	Disque      int64   `json:"disque"`
	Restants    []Refus `json:"restants"` // non traités : écartés, refusés par Windows, interrompus
	Annule      bool    `json:"annule"`
	Erreur      string  `json:"erreur"`
	Perime      bool    `json:"perime"` // l'analyse ne montre pas encore ce qui est arrivé
}

const maxApercu = 100

// prepa : ce qui est retenu pour une opération, après tous les contrôles.
type prepa struct {
	apercu  Apercu
	cibles  []scan.Cible
	chemins []string
}

func nomValide(nom string) bool {
	return nom != "" && nom != "." && nom != ".." && filepath.Base(nom) == nom
}

// memeChemin compare deux chemins comme le système de fichiers.
func memeChemin(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// dedans : p est d ou se trouve dans d.
func dedans(p, d string) bool {
	if memeChemin(p, d) {
		return true
	}
	rel, err := filepath.Rel(d, p)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// verifierDestination contrôle le dossier d'arrivée d'un déplacement ou d'une copie.
func verifierDestination(dest string) (string, error) {
	if dest == "" {
		return "", errors.New("aucun dossier d'arrivée choisi")
	}
	if !filepath.IsAbs(dest) {
		return "", errors.New("dossier d'arrivée incomplet : " + dest)
	}
	dest = filepath.Clean(dest)
	info, err := os.Stat(dest)
	if err != nil {
		return "", fmt.Errorf("dossier d'arrivée introuvable : %s", dest)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s n'est pas un dossier", dest)
	}
	if raison := Protections.RaisonDestination(dest); raison != "" {
		return "", fmt.Errorf("on ne dépose rien dans %s : %s", dest, raison)
	}
	return dest, nil
}

// preparer fait tous les contrôles d'une opération sans rien toucher.
func preparer(sc *scan.Scan, op string, cibles []scan.Cible, dest string) (prepa, error) {
	var p prepa
	switch op {
	case OpCorbeille:
		dest = ""
	case OpDeplacer, OpCopier:
		var err error
		if dest, err = verifierDestination(dest); err != nil {
			return p, err
		}
	default:
		return p, errors.New("opération inconnue : " + op)
	}
	resume, err := sc.Resumer(cibles)
	if err != nil {
		return p, err
	}
	// Listes jamais nulles : l'interface reçoit [] et non null.
	a := Apercu{
		Operation: op, Destination: dest, Filtre: resume.Filtre, Libre: -1,
		Elements: []ElementApercu{}, Ecartes: []Refus{}, Conflits: []string{}, Amovibles: []string{},
	}
	var volDest volumes.Volume
	if dest != "" {
		volDest, _ = volumeDe(dest)
		a.DansAnalyse = sc.Contient(dest)
	}

	for _, c := range resume.Cibles {
		chemin, err := sc.Chemin(c.ID)
		if err != nil {
			return p, err
		}
		if c.Type == "fichier" {
			if !nomValide(c.Nom) {
				return p, errors.New("nom de fichier invalide : " + c.Nom)
			}
			chemin = filepath.Join(chemin, c.Nom)
		}
		ecarter := func(raison string) { a.Ecartes = append(a.Ecartes, Refus{Chemin: chemin, Raison: raison}) }

		// Ce qui quitte son emplacement (corbeille, déplacement) obéit à toutes
		// les protections ; une copie ne fait que lire, seule une racine de
		// disque est refusée.
		if op == OpCopier {
			if filepath.Dir(chemin) == chemin {
				ecarter("racine d'un disque")
				continue
			}
		} else if raison := Protections.Raison(chemin); raison != "" {
			ecarter(raison)
			continue
		}
		if dest != "" {
			if c.Type == "dossier" && dedans(dest, chemin) {
				ecarter("un dossier ne peut pas aller dans lui-même")
				continue
			}
			if memeChemin(filepath.Dir(chemin), dest) {
				ecarter("déjà dans ce dossier")
				continue
			}
		}

		r, err := sc.Resumer([]scan.Cible{c})
		if err != nil {
			return p, err
		}
		a.Nombre++
		a.Fichiers += r.Fichiers
		a.Taille += r.Taille
		a.Disque += r.Disque
		a.Elements = append(a.Elements, ElementApercu{Chemin: chemin, Type: c.Type, Taille: r.Taille, Disque: r.Disque})
		p.cibles = append(p.cibles, c)
		p.chemins = append(p.chemins, chemin)

		vol, trouve := volumeDe(chemin)
		switch op {
		case OpCorbeille:
			if trouve && vol.Type == "amovible" && !slices.Contains(a.Amovibles, vol.Chemin) {
				a.Amovibles = append(a.Amovibles, vol.Chemin)
			}
		case OpCopier:
			a.Besoin += r.Taille
		case OpDeplacer:
			// Sur le même disque, un déplacement n'est qu'un renommage.
			if !trouve || vol.Chemin != volDest.Chemin {
				a.Besoin += r.Taille
			}
		}
		if dest != "" {
			if _, err := os.Lstat(filepath.Join(dest, filepath.Base(chemin))); err == nil {
				a.Conflits = append(a.Conflits, filepath.Base(chemin))
			}
		}
	}

	if dest != "" && volDest.Chemin != "" {
		a.Libre = volDest.Libre
		if a.Besoin > a.Libre {
			a.Manque = a.Besoin - a.Libre
		}
	}
	slices.SortStableFunc(a.Elements, func(x, y ElementApercu) int { return cmp.Compare(y.Taille, x.Taille) })
	if len(a.Elements) > maxApercu {
		a.Elements = a.Elements[:maxApercu]
	}
	p.apercu = a
	return p, nil
}

// Apercevoir récapitule ce que ferait l'opération, sans rien toucher.
func Apercevoir(sc *scan.Scan, op string, cibles []scan.Cible, dest string) (Apercu, error) {
	p, err := preparer(sc, op, cibles, dest)
	return p.apercu, err
}

// Executer réalise l'opération par l'Explorateur de Windows, puis met
// l'analyse à jour pour ce qui a vraiment quitté son emplacement. Les
// éléments écartés (protégés, dossier dans lui-même…) ne sont jamais touchés.
// hwnd : la fenêtre d'Aria, pour y rattacher les boîtes de Windows.
func Executer(sc *scan.Scan, op string, cibles []scan.Cible, dest string, hwnd uintptr) (Bilan, error) {
	if !opMu.TryLock() {
		return Bilan{}, errors.New("une opération de ménage est déjà en cours")
	}
	defer opMu.Unlock()

	p, err := preparer(sc, op, cibles, dest)
	if err != nil {
		return Bilan{}, err
	}
	a := p.apercu
	b := Bilan{Operation: op, Destination: a.Destination, Restants: append([]Refus{}, a.Ecartes...)}
	if a.Manque > 0 {
		return b, fmt.Errorf("pas assez de place à l'arrivée : il manque %d octets", a.Manque)
	}
	if len(p.cibles) == 0 {
		return b, nil
	}

	// Copie : les originaux ne bougent pas, l'analyse non plus.
	if op == OpCopier {
		tailles := map[string]ElementApercu{}
		for _, e := range a.Elements {
			tailles[e.Chemin] = e
		}
		existaient := map[string]bool{}
		for _, c := range p.chemins {
			_, err := os.Lstat(filepath.Join(a.Destination, filepath.Base(c)))
			existaient[c] = err == nil
		}
		b.Annule, err = Copier(p.chemins, a.Destination, hwnd)
		if err != nil {
			b.Erreur = err.Error()
		}
		for _, c := range p.chemins {
			if _, err := os.Lstat(filepath.Join(a.Destination, filepath.Base(c))); err == nil && !existaient[c] {
				b.Faits++
				b.Taille += tailles[c].Taille
				b.Disque += tailles[c].Disque
			} else if !existaient[c] {
				b.Restants = append(b.Restants, Refus{Chemin: c, Raison: raisonRestant(b.Annule)})
			}
			// Un nom déjà pris : Windows a demandé, on ne sait pas ce qui a été
			// choisi. Ni compté, ni signalé comme raté.
		}
		b.Perime = b.Faits > 0 && a.DansAnalyse
		return b, nil
	}

	// Corbeille et déplacement : l'élément quitte son emplacement.
	retraits, err := sc.PreparerRetrait(p.cibles)
	if err != nil {
		return Bilan{}, err
	}
	chemins := make([]string, len(retraits))
	for i, r := range retraits {
		chemins[i] = r.Chemin
	}
	if op == OpCorbeille {
		b.Annule, err = Corbeille(chemins, hwnd)
	} else {
		b.Annule, err = Deplacer(chemins, a.Destination, hwnd)
	}
	if err != nil {
		b.Erreur = err.Error()
	}
	faits := sc.Retirer(retraits)
	partis := map[string]bool{}
	for _, f := range faits {
		b.Faits++
		b.Taille += f.Taille
		b.Disque += f.Disque
		partis[f.Chemin] = true
	}
	for _, r := range retraits {
		if !partis[r.Chemin] {
			b.Restants = append(b.Restants, Refus{Chemin: r.Chemin, Raison: raisonRestant(b.Annule)})
		}
	}
	b.Perime = op == OpDeplacer && b.Faits > 0 && a.DansAnalyse
	return b, nil
}

func raisonRestant(annule bool) string {
	if annule {
		return "opération interrompue avant lui"
	}
	return "toujours à sa place : Windows l'a refusé ou un nom était déjà pris"
}
