// Package export écrit les résultats d'Aria Scope dans des fichiers à
// partager ou à retravailler : CSV (Excel en français), rapport HTML, classeur
// Excel.
package export

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"aria-scope/internal/scan"
)

// Ecrire crée le fichier chemin par écriture, sous un nom provisoire
// renommé à la fin : en cas d'erreur, rien de bancal ne reste.
func Ecrire(chemin string, ecrire func(w io.Writer) error) error {
	tmp := chemin + ".part"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	b := bufio.NewWriterSize(f, 1<<16)
	err = ecrire(b)
	if err == nil {
		err = b.Flush()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp, chemin)
	}
	if err != nil {
		os.Remove(tmp)
	}
	return err
}

// csv écrit un CSV qu'Excel en français ouvre d'un double-clic : UTF-8 avec
// BOM, « ; » entre les colonnes, fins de ligne Windows.
type csv struct {
	w   io.Writer
	err error
}

func nouveauCSV(w io.Writer, entetes ...string) *csv {
	c := &csv{w: w}
	_, c.err = io.WriteString(w, "\uFEFF")
	c.ligne(entetes...)
	return c
}

func (c *csv) ligne(champs ...string) {
	if c.err != nil {
		return
	}
	var b strings.Builder
	for i, ch := range champs {
		if i > 0 {
			b.WriteByte(';')
		}
		// Une formule (=, +, -, @ en tête) ne doit pas s'exécuter dans Excel :
		// un nom de fichier n'est que du texte.
		if ch != "" && strings.ContainsRune("=+-@", rune(ch[0])) && !estNombre(ch) {
			ch = "'" + ch
		}
		if strings.ContainsAny(ch, ";\"\r\n") {
			ch = `"` + strings.ReplaceAll(ch, `"`, `""`) + `"`
		}
		b.WriteString(ch)
	}
	b.WriteString("\r\n")
	_, c.err = io.WriteString(c.w, b.String())
}

func estNombre(s string) bool {
	_, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	return err == nil
}

// ---------- Mise en forme ----------

func entier(n int64) string { return strconv.FormatInt(n, 10) }

// decimal : avec une virgule, comme Excel en français l'attend.
func decimal(f float64, chiffres int) string {
	return strings.Replace(strconv.FormatFloat(f, 'f', chiffres, 64), ".", ",", 1)
}

var unites = []string{"o", "Ko", "Mo", "Go", "To", "Po"}

// Lisible : trois chiffres significatifs, comme l'interface (9,87 Go).
func Lisible(n int64) string {
	if n <= 0 {
		return "0 o"
	}
	v, i := float64(n), 0
	for v >= 1024 && i < len(unites)-1 {
		v /= 1024
		i++
	}
	d := 0
	switch {
	case i == 0:
	case v < 10:
		d = 2
	case v < 100:
		d = 1
	}
	// Arrondi qui passe à l'unité suivante (1023,9 Ko → 1 024 Ko) : on garde l'affichage simple.
	p := math.Pow(10, float64(d))
	v = math.Round(v*p) / p
	return decimal(v, d) + " " + unites[i]
}

// Date : lue comme une date par Excel en français ; vide si inconnue.
func Date(sec int64) string {
	if sec <= 0 {
		return ""
	}
	return time.Unix(sec, 0).Format("2006-01-02 15:04:05")
}

func pourcent(part float64) string { return decimal(part*100, 1) }

func poids(taille, disque int64, mode string) int64 {
	if mode == scan.ModeDisque {
		return disque
	}
	return taille
}

// ---------- Les vues ----------

// ArbreCSV : les dossiers de l'arbre, parcourus par parcourir.
func ArbreCSV(w io.Writer, mode string, parcourir func(func(scan.LigneArbre) error) error) error {
	c := nouveauCSV(w, "Niveau", "Chemin", "Nom", "Taille (octets)", "Sur disque (octets)", "Taille", "Part du parent (%)", "Fichiers", "Sous-dossiers", "Modifié", "Illisibles")
	err := parcourir(func(l scan.LigneArbre) error {
		c.ligne(strconv.Itoa(l.Niveau), l.Chemin, l.Nom, entier(l.Taille), entier(l.Disque), Lisible(poids(l.Taille, l.Disque, mode)),
			pourcent(l.Part), entier(l.Fichiers), entier(l.Dossiers), Date(l.Modif), entier(l.Erreurs))
		return c.err
	})
	if err != nil {
		return err
	}
	return c.err
}

func complet(dossier, nom string) string {
	if dossier == "" {
		return nom
	}
	return filepath.Join(dossier, nom)
}

// GrosCSV : les plus gros fichiers.
func GrosCSV(w io.Writer, mode string, lignes []scan.GrosFichier) error {
	c := nouveauCSV(w, "Rang", "Nom", "Dossier", "Chemin complet", "Taille (octets)", "Sur disque (octets)", "Taille", "Modifié")
	for i, l := range lignes {
		c.ligne(strconv.Itoa(i+1), l.Nom, l.Dossier, complet(l.Dossier, l.Nom), entier(l.Taille), entier(l.Disque), Lisible(poids(l.Taille, l.Disque, mode)), Date(l.Modif))
	}
	return c.err
}

// TypesCSV : chaque famille, puis ses extensions.
func TypesCSV(w io.Writer, mode string, familles []scan.Famille) error {
	var total int64
	for _, f := range familles {
		total += poids(f.Taille, f.Disque, mode)
	}
	part := func(n int64) string {
		if total <= 0 {
			return "0"
		}
		return pourcent(float64(n) / float64(total))
	}
	c := nouveauCSV(w, "Famille", "Extension", "Fichiers", "Taille (octets)", "Sur disque (octets)", "Taille", "Part (%)")
	for _, f := range familles {
		p := poids(f.Taille, f.Disque, mode)
		c.ligne(f.Nom, "(toute la famille)", entier(f.Fichiers), entier(f.Taille), entier(f.Disque), Lisible(p), part(p))
		for _, e := range f.Extensions {
			ext := "." + e.Ext
			if e.Ext == "" {
				ext = "(sans extension)"
			}
			pe := poids(e.Taille, e.Disque, mode)
			c.ligne(f.Nom, ext, entier(e.Fichiers), entier(e.Taille), entier(e.Disque), Lisible(pe), part(pe))
		}
	}
	return c.err
}

// AgesCSV : la répartition par âge.
func AgesCSV(w io.Writer, mode string, ages []scan.Age) error {
	var total int64
	for _, a := range ages {
		total += poids(a.Taille, a.Disque, mode)
	}
	c := nouveauCSV(w, "Dernière modification", "Fichiers", "Taille (octets)", "Sur disque (octets)", "Taille", "Part (%)")
	for _, a := range ages {
		p := poids(a.Taille, a.Disque, mode)
		part := "0"
		if total > 0 {
			part = pourcent(float64(p) / float64(total))
		}
		c.ligne(a.Nom, entier(a.Fichiers), entier(a.Taille), entier(a.Disque), Lisible(p), part)
	}
	return c.err
}

// RechercheCSV : les résultats d'une recherche.
func RechercheCSV(w io.Writer, mode string, lignes []scan.Trouve) error {
	c := nouveauCSV(w, "Type", "Nom", "Dossier", "Chemin complet", "Taille (octets)", "Sur disque (octets)", "Taille", "Modifié", "Longueur du chemin")
	for _, l := range lignes {
		c.ligne(l.Type, l.Nom, l.Dossier, complet(l.Dossier, l.Nom), entier(l.Taille), entier(l.Disque), Lisible(poids(l.Taille, l.Disque, mode)), Date(l.Modif), strconv.Itoa(l.Longueur))
	}
	return c.err
}

// DoublonsCSV : une ligne par copie, regroupées par empreinte.
func DoublonsCSV(w io.Writer, groupes []scan.GroupeDoublons) error {
	c := nouveauCSV(w, "Groupe", "SHA-256", "Copies", "Taille d'un exemplaire (octets)", "Récupérable (octets)", "Récupérable", "Nom", "Dossier", "Chemin complet", "Modifié")
	for i, g := range groupes {
		for _, f := range g.Fichiers {
			c.ligne(strconv.Itoa(i+1), g.Empreinte, strconv.Itoa(len(g.Fichiers)), entier(g.Taille), entier(g.Recuperable), Lisible(g.Recuperable),
				f.Nom, f.Dossier, complet(f.Dossier, f.Nom), Date(f.Modif))
		}
	}
	return c.err
}

// NomFichier propose un nom de fichier d'export : « Aria Scope - Gros fichiers - 2026-11-01.csv ».
func NomFichier(quoi, ext string, quand time.Time) string {
	return fmt.Sprintf("Aria Scope - %s - %s.%s", quoi, quand.Format("2006-01-02"), ext)
}

// Options : quelle vue exporter, et comment.
type Options struct {
	Vue        string   `json:"vue"` // arbre, gros, types, ages, recherche, doublons
	Mode       string   `json:"mode"`
	Tri        scan.Tri `json:"tri"`
	Dans       int32    `json:"dans"`       // arbre, types, âge : dossier de départ (0 : toute l'analyse)
	Profondeur int      `json:"profondeur"` // arbre : niveaux sous le départ (0 : tous)
}

// Titres des vues, pour les noms de fichiers.
var Titres = map[string]string{
	"arbre": "Arbre", "gros": "Gros fichiers", "types": "Types", "ages": "Âge des fichiers",
	"recherche": "Recherche", "doublons": "Doublons",
}

// VueCSV écrit la vue demandée de l'analyse sc. La recherche et les doublons
// sont ceux de l'analyse en cours (nil s'il n'y en a pas).
func VueCSV(w io.Writer, sc *scan.Scan, r *scan.Recherche, d *scan.Doublons, o Options) error {
	switch o.Vue {
	case "arbre":
		return ArbreCSV(w, o.Mode, func(fn func(scan.LigneArbre) error) error {
			return sc.ParcourirArbre(o.Dans, o.Profondeur, o.Mode, fn)
		})
	case "gros":
		l, err := sc.GrosFichiers(o.Mode, o.Tri)
		if err != nil {
			return err
		}
		return GrosCSV(w, o.Mode, l)
	case "types":
		l, err := sc.Types(o.Dans, o.Mode)
		if err != nil {
			return err
		}
		return TypesCSV(w, o.Mode, l)
	case "ages":
		l, err := sc.Ages(o.Dans)
		if err != nil {
			return err
		}
		return AgesCSV(w, o.Mode, l)
	case "recherche":
		if r == nil {
			return errAucune("recherche")
		}
		l, err := r.Resultats(o.Mode, o.Tri)
		if err != nil {
			return err
		}
		return RechercheCSV(w, o.Mode, l)
	case "doublons":
		if d == nil {
			return errAucune("recherche de doublons")
		}
		g, err := d.Groupes()
		if err != nil {
			return err
		}
		return DoublonsCSV(w, g)
	}
	return fmt.Errorf("vue inconnue : %s", o.Vue)
}

func errAucune(quoi string) error {
	return fmt.Errorf("aucune %s à exporter sur cette analyse", quoi)
}
