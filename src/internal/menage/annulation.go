package menage

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"aria-scope/internal/journal"
)

// Suffixe d'un élément en cours de restauration depuis une archive :
// renommé seulement une fois entièrement extrait.
const suffixeRestauration = ".aria-restauration"

// LigneAnnulation : ce que l'annulation fera d'un élément.
type LigneAnnulation struct {
	Chemin string `json:"chemin"` // l'élément concerné aujourd'hui
	Vers   string `json:"vers"`   // où il retourne ; vide : à la corbeille
	Type   string `json:"type"`
	Taille int64  `json:"taille"`
}

// ApercuAnnulation : ce qu'annuler une opération du journal va faire.
type ApercuAnnulation struct {
	Operation string            `json:"operation"`
	Lignes    []LigneAnnulation `json:"lignes"`
	Ecartes   []Refus           `json:"ecartes"`
	Taille    int64             `json:"taille"`
	// Archive : le .zip part à la corbeille (tout ce qui en avait été
	// recyclé peut être restauré, ou rien ne l'avait été).
	Archive         string `json:"archive"`
	ArchiveRecyclee bool   `json:"archiveRecyclee"`
	ArchiveGardee   string `json:"archiveGardee"` // pourquoi le .zip est conservé
	// Impossible : pourquoi rien ne peut être annulé (vide sinon).
	Impossible string `json:"impossible"`
}

// BilanAnnulation : ce que l'annulation a fait.
type BilanAnnulation struct {
	Operation       string  `json:"operation"`
	Faits           int     `json:"faits"`
	Taille          int64   `json:"taille"`
	Restants        []Refus `json:"restants"`
	ArchiveRecyclee bool    `json:"archiveRecyclee"`
	ArchiveGardee   string  `json:"archiveGardee"`
	Annule          bool    `json:"annule"`
	Erreur          string  `json:"erreur"`
	Resume          string  `json:"resume"` // une phrase, gardée dans le journal
}

// ApercevoirAnnulation récapitule ce qu'annuler l'opération ferait, sans
// rien toucher.
func ApercevoirAnnulation(e journal.Entree) ApercuAnnulation {
	a := ApercuAnnulation{Operation: e.Operation, Lignes: []LigneAnnulation{}, Ecartes: []Refus{}}
	ecarter := func(chemin, raison string) { a.Ecartes = append(a.Ecartes, Refus{Chemin: chemin, Raison: raison}) }
	ajouter := func(l LigneAnnulation) {
		a.Lignes = append(a.Lignes, l)
		a.Taille += l.Taille
	}
	if e.Annulee != nil {
		a.Impossible = "cette opération a déjà été annulée"
		return a
	}

	switch e.Operation {
	case OpCorbeille:
		a.Impossible = "un envoi à la corbeille se restaure depuis la corbeille de Windows (clic droit, « Restaurer »)"

	case OpDeplacer:
		for _, el := range e.Elements {
			switch {
			case !existe(el.Arrivee):
				ecarter(el.Arrivee, "n'est plus à l'arrivée (déplacé, renommé ou supprimé depuis)")
			case existe(el.Origine):
				ecarter(el.Arrivee, "un élément porte déjà ce nom à son emplacement d'origine")
			default:
				ajouter(LigneAnnulation{Chemin: el.Arrivee, Vers: el.Origine, Type: el.Type, Taille: el.Taille})
			}
		}

	case OpCopier:
		for _, el := range e.Elements {
			switch {
			case !existe(el.Arrivee):
				ecarter(el.Arrivee, "la copie n'existe plus")
			case Protections.Raison(el.Arrivee) != "":
				ecarter(el.Arrivee, Protections.Raison(el.Arrivee))
			default:
				ajouter(LigneAnnulation{Chemin: el.Arrivee, Type: el.Type, Taille: el.Taille})
			}
		}

	case OpArchiver:
		a.Archive = e.Archive
		present := existe(e.Archive)
		recycles := 0
		for _, el := range e.Elements {
			if !el.Recycle {
				continue
			}
			recycles++
			switch {
			case !present:
				ecarter(el.Origine, "l'archive n'existe plus : rien à restaurer")
			case existe(el.Origine):
				ecarter(el.Origine, "un élément porte déjà ce nom à son emplacement d'origine")
			case existe(el.Origine + suffixeRestauration):
				ecarter(el.Origine, "une restauration précédente a laissé "+filepath.Base(el.Origine+suffixeRestauration))
			default:
				ajouter(LigneAnnulation{Chemin: el.Arrivee, Vers: el.Origine, Type: el.Type, Taille: el.Taille})
			}
		}
		switch {
		case !present:
			if recycles == 0 {
				a.Impossible = "l'archive n'existe plus"
			}
		case len(a.Ecartes) > 0:
			a.ArchiveGardee = "tout ne peut pas être restauré : l'archive reste en place"
		default:
			a.ArchiveRecyclee = true
		}
		if len(a.Lignes) == 0 && !a.ArchiveRecyclee && a.Impossible == "" {
			a.Impossible = "rien ne peut être restauré"
		}

	default:
		a.Impossible = "opération inconnue : " + e.Operation
	}

	if a.Impossible == "" && len(a.Lignes) == 0 && !a.ArchiveRecyclee {
		a.Impossible = "plus rien à annuler : tous les éléments ont changé depuis"
	}
	return a
}

// Annuler défait une opération du journal, en ne touchant qu'à ce que
// l'aperçu retient : un déplacement est remis en place, une copie part à la
// corbeille, les originaux d'une archive sont restaurés depuis le .zip, qui
// part ensuite à la corbeille si tout a pu être restauré. Rien n'est
// jamais écrasé ni supprimé définitivement.
func Annuler(e journal.Entree, hwnd uintptr) (BilanAnnulation, error) {
	if !opMu.TryLock() {
		return BilanAnnulation{}, errors.New("une opération de ménage est déjà en cours")
	}
	defer opMu.Unlock()

	a := ApercevoirAnnulation(e)
	if a.Impossible != "" {
		return BilanAnnulation{}, errors.New(a.Impossible)
	}
	b := BilanAnnulation{Operation: e.Operation, Restants: append([]Refus{}, a.Ecartes...), ArchiveGardee: a.ArchiveGardee}
	compter := func(l LigneAnnulation, ok bool, raison string) {
		if ok {
			b.Faits++
			b.Taille += l.Taille
		} else {
			b.Restants = append(b.Restants, Refus{Chemin: l.Chemin, Raison: raison})
		}
	}

	switch e.Operation {
	case OpDeplacer:
		// L'Explorateur déplace vers un seul dossier à la fois : on regroupe.
		groupes := map[string][]LigneAnnulation{}
		var ordre []string
		for _, l := range a.Lignes {
			d := filepath.Dir(l.Vers)
			if _, vu := groupes[d]; !vu {
				ordre = append(ordre, d)
			}
			groupes[d] = append(groupes[d], l)
		}
		for _, d := range ordre {
			lignes := groupes[d]
			if err := os.MkdirAll(d, 0o755); err != nil {
				for _, l := range lignes {
					compter(l, false, "dossier d'origine impossible à recréer")
				}
				continue
			}
			chemins := make([]string, len(lignes))
			for i, l := range lignes {
				chemins[i] = l.Chemin
			}
			annule, err := Deplacer(chemins, d, hwnd)
			if err != nil && b.Erreur == "" {
				b.Erreur = err.Error()
			}
			b.Annule = b.Annule || annule
			for _, l := range lignes {
				compter(l, existe(l.Vers) && !existe(l.Chemin), raisonRestant(annule))
			}
		}

	case OpCopier:
		chemins := make([]string, len(a.Lignes))
		for i, l := range a.Lignes {
			chemins[i] = l.Chemin
		}
		annule, err := Corbeille(chemins, hwnd)
		if err != nil {
			b.Erreur = err.Error()
		}
		b.Annule = annule
		for _, l := range a.Lignes {
			compter(l, !existe(l.Chemin), raisonRestant(annule))
		}

	case OpArchiver:
		complet := true
		if len(a.Lignes) > 0 {
			z, err := zip.OpenReader(e.Archive)
			if err != nil && !errors.Is(err, zip.ErrInsecurePath) {
				return b, fmt.Errorf("archive illisible : %w", err)
			}
			for _, l := range a.Lignes {
				err := restaurer(&z.Reader, l.Chemin, l.Vers)
				compter(l, err == nil, fmt.Sprint(err))
				complet = complet && err == nil
			}
			z.Close()
		}
		if a.ArchiveRecyclee && complet {
			annule, err := Corbeille([]string{e.Archive}, hwnd)
			if err != nil {
				b.Erreur = err.Error()
			}
			b.ArchiveRecyclee = !existe(e.Archive)
			if !b.ArchiveRecyclee {
				b.ArchiveGardee = "Windows n'a pas envoyé l'archive à la corbeille"
				b.Annule = annule
			}
		} else if a.ArchiveRecyclee {
			b.ArchiveGardee = "tout n'a pas pu être restauré : l'archive reste en place"
		}
	}

	b.Resume = resumerAnnulation(b)
	return b, nil
}

// restaurer extrait de l'archive l'élément nommé haut (un fichier, ou un
// dossier et tout son contenu) vers vers. L'extraction se fait sous un nom
// provisoire, renommé seulement à la fin : en cas d'erreur, rien ne reste.
func restaurer(z *zip.Reader, haut, vers string) error {
	var fichiers []*zip.File
	for _, f := range z.File {
		if f.Name == haut || f.Name == haut+"/" || strings.HasPrefix(f.Name, haut+"/") {
			fichiers = append(fichiers, f)
		}
	}
	if len(fichiers) == 0 {
		return errors.New("absent de l'archive")
	}
	if existe(vers) {
		return errors.New("un élément porte déjà ce nom à son emplacement d'origine")
	}
	if err := os.MkdirAll(filepath.Dir(vers), 0o755); err != nil {
		return fmt.Errorf("dossier d'origine impossible à recréer : %w", err)
	}
	tmp := vers + suffixeRestauration
	err := extraire(fichiers, haut, tmp)
	if err == nil {
		if existe(vers) { // apparu pendant l'extraction
			err = errors.New("un élément porte déjà ce nom à son emplacement d'origine")
		} else {
			err = os.Rename(tmp, vers)
		}
	}
	if err != nil {
		os.RemoveAll(tmp)
	}
	return err
}

func extraire(fichiers []*zip.File, haut, tmp string) error {
	// Un seul fichier, à la racine de l'archive
	if len(fichiers) == 1 && fichiers[0].Name == haut {
		return extraireFichier(fichiers[0], tmp)
	}
	if err := os.Mkdir(tmp, 0o755); err != nil {
		return err
	}
	type date struct {
		chemin string
		quand  time.Time
	}
	var dossiers []date
	for _, f := range fichiers {
		rel := strings.TrimPrefix(strings.TrimPrefix(f.Name, haut), "/")
		cible := tmp
		if rel != "" && rel != "/" {
			local := filepath.FromSlash(strings.TrimSuffix(rel, "/"))
			// Une archive retouchée ne doit rien écrire hors du dossier restauré.
			if !filepath.IsLocal(local) {
				return fmt.Errorf("nom suspect dans l'archive : %s", f.Name)
			}
			cible = filepath.Join(tmp, local)
		}
		if strings.HasSuffix(f.Name, "/") {
			if err := os.MkdirAll(cible, 0o755); err != nil {
				return err
			}
			dossiers = append(dossiers, date{cible, f.Modified})
			continue
		}
		if err := os.MkdirAll(filepath.Dir(cible), 0o755); err != nil {
			return err
		}
		if err := extraireFichier(f, cible); err != nil {
			return err
		}
	}
	// Les dates des dossiers en dernier, les plus profonds d'abord : écrire
	// dedans les aurait changées.
	slices.SortFunc(dossiers, func(x, y date) int { return len(y.chemin) - len(x.chemin) })
	for _, d := range dossiers {
		if !d.quand.IsZero() {
			_ = os.Chtimes(d.chemin, d.quand, d.quand)
		}
	}
	return nil
}

// extraireFichier écrit un fichier de l'archive sans jamais écraser ; le CRC
// est contrôlé à la lecture.
func extraireFichier(f *zip.File, cible string) error {
	src, err := f.Open()
	if err != nil {
		return fmt.Errorf("%s : %w", f.Name, err)
	}
	defer src.Close()
	dst, err := os.OpenFile(cible, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return fmt.Errorf("%s : %w", f.Name, err)
	}
	if err := dst.Close(); err != nil {
		return err
	}
	if !f.Modified.IsZero() {
		_ = os.Chtimes(cible, f.Modified, f.Modified)
	}
	return nil
}

func resumerAnnulation(b BilanAnnulation) string {
	n := b.Faits
	s := ""
	if n > 1 {
		s = "s"
	}
	var r string
	switch b.Operation {
	case OpDeplacer:
		r = fmt.Sprintf("%d élément%s remis en place", n, s)
	case OpCopier:
		r = fmt.Sprintf("%d copie%s envoyée%s à la corbeille", n, s, s)
	case OpArchiver:
		switch {
		case n > 1:
			r = fmt.Sprintf("%d originaux restaurés", n)
		case n == 1:
			r = "1 original restauré"
		}
		if b.ArchiveRecyclee {
			if r != "" {
				r += ", "
			}
			r += "archive envoyée à la corbeille"
		}
	}
	if k := len(b.Restants); k > 1 {
		r += fmt.Sprintf(" · %d laissés de côté", k)
	} else if k == 1 {
		r += " · 1 laissé de côté"
	}
	return r
}

// Entree prépare l'inscription d'une opération au journal ; faux si rien
// n'a été fait.
func (b Bilan) Entree() (journal.Entree, bool) {
	if len(b.Traces) == 0 {
		return journal.Entree{}, false
	}
	return journal.Entree{
		Operation: b.Operation, Destination: b.Destination, Archive: b.Archive,
		Elements: slices.Clone(b.Traces), Taille: b.Taille,
	}, true
}
