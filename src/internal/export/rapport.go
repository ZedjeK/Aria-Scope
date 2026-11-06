package export

import (
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"math"
	"path/filepath"
	"strings"
	"time"

	"aria-scope/internal/scan"
)

//go:embed rapport.html
var modeleRapport string

var rapportTmpl = template.Must(template.New("rapport").Funcs(template.FuncMap{
	"lisible": Lisible,
	"nombre":  nombre,
	"date": func(sec int64) string {
		if sec <= 0 {
			return "—"
		}
		return time.Unix(sec, 0).Format("02/01/2006")
	},
	"pct": func(p float64) string { return decimal(p*100, 1) + " %" },
	// largeurs et coordonnées pour le CSS et le SVG : point décimal
	"barre": func(p float64) string { return fmt.Sprintf("%.2f", math.Max(0.4, p*100)) },
	"f1":    func(f float64) string { return fmt.Sprintf("%.1f", f) },
	"inc":   func(i int) int { return i + 1 },
	"int64": func(i int) int64 { return int64(i) },
}).Parse(modeleRapport))

// nombre : 12 345 678, avec des espaces fines insécables.
func nombre(n int64) string {
	s := fmt.Sprint(n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteString(" ")
		}
		b.WriteRune(r)
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// ---------- Données du rapport ----------

type ligneDossier struct {
	scan.LigneArbre
	Valeur     int64
	PartRacine float64
}

type ligneValeur struct {
	Nom      string
	Detail   string
	Fichiers int64
	Valeur   int64
	Part     float64 // du total
}

type blocCarte struct {
	X, Y, W, H float64
	Couleur    string
	Nom        string
	Taille     string
	Etiquette  bool // assez grand pour un nom
	Titre      bool // dossier de premier niveau, avec son nom en tête
	Opacite    float64
}

type groupeRapport struct {
	Copies      int
	Taille      int64
	Recuperable int64
	Nom         string
	Dossiers    []string
}

type rapport struct {
	Version      string
	Genere       string
	Chemin       string // dossier du rapport
	Analyse      string // racine de l'analyse
	Partiel      bool   // rapport d'un sous-dossier
	ModeDisque   bool
	Racine       scan.LigneArbre
	Valeur       int64
	Duree        string
	Filtres      string
	Ecartes      int64
	Carte        []blocCarte
	CarteW       float64
	CarteH       float64
	Dossiers     []ligneDossier
	Familles     []ligneValeur
	Extensions   []ligneValeur
	Ages         []ligneValeur
	Gros         []scan.GrosFichier
	GrosLimite   int
	Doublons     []groupeRapport
	DblGroupes   int
	DblCopies    int
	DblPlace     int64
	AvecDoublons bool
}

// Teintes de la carte, comme dans l'application.
var teintes = []string{"#7df9ff", "#5c8bff", "#b18cff", "#3fe0c5", "#ff6fae", "#59c3ff", "#b8f35a", "#a9b8ff"}

const (
	grisFichiers = "#93a8c8"
	grisReste    = "#3b5378"
	maxGros      = 50
	maxDossiers  = 40
	maxGroupes   = 20
)

// RapportHTML écrit un rapport autonome (un seul fichier, sans rien à
// télécharger) sur le dossier o.Dans de l'analyse sc : résumé, carte, plus
// gros dossiers, types, âge, plus gros fichiers, et les doublons s'ils ont
// été cherchés. Il s'imprime proprement (et donc en PDF).
func RapportHTML(w io.Writer, sc *scan.Scan, d *scan.Doublons, o Options, version string, maintenant time.Time) error {
	mode := o.Mode
	r := rapport{
		Version: version, Genere: maintenant.Format("02/01/2006 à 15:04"),
		Analyse: sc.Racine(), Partiel: o.Dans != 0, ModeDisque: mode == scan.ModeDisque,
		CarteW: 1000, CarteH: 520, GrosLimite: maxGros,
	}
	p := sc.Progression()
	r.Duree = decimal(float64(p.DureeMs)/1000, 1) + " s"
	r.Ecartes = p.Ecartes
	if f := sc.Filtres(); f.Actifs() {
		r.Filtres = decrireFiltres(f)
	}

	// Résumé et dossiers sur deux niveaux
	var lignes []scan.LigneArbre
	if err := sc.ParcourirArbre(o.Dans, 2, mode, func(l scan.LigneArbre) error {
		lignes = append(lignes, l)
		return nil
	}); err != nil {
		return err
	}
	r.Racine = lignes[0]
	r.Chemin = r.Racine.Chemin
	r.Valeur = poids(r.Racine.Taille, r.Racine.Disque, mode)
	for _, l := range lignes[1:] {
		if len(r.Dossiers) >= maxDossiers {
			break
		}
		v := poids(l.Taille, l.Disque, mode)
		pr := 0.0
		if r.Valeur > 0 {
			pr = float64(v) / float64(r.Valeur)
		}
		// Le second niveau seulement pour ce qui pèse (1 % de l'ensemble)
		if l.Niveau == 2 && pr < 0.01 {
			continue
		}
		r.Dossiers = append(r.Dossiers, ligneDossier{LigneArbre: l, Valeur: v, PartRacine: pr})
	}

	// Carte
	if b, err := sc.Carte(o.Dans, mode); err == nil {
		r.Carte = carte(b, r.CarteW, r.CarteH)
	}

	// Types et extensions
	if fams, err := sc.Types(o.Dans, mode); err == nil {
		var total int64
		for _, f := range fams {
			total += poids(f.Taille, f.Disque, mode)
		}
		var exts []ligneValeur
		for _, f := range fams {
			v := poids(f.Taille, f.Disque, mode)
			r.Familles = append(r.Familles, ligneValeur{Nom: f.Nom, Fichiers: f.Fichiers, Valeur: v, Part: part(v, total)})
			for _, e := range f.Extensions {
				nom := "." + e.Ext
				if e.Ext == "" {
					nom = "sans extension"
				}
				ve := poids(e.Taille, e.Disque, mode)
				exts = append(exts, ligneValeur{Nom: nom, Detail: f.Nom, Fichiers: e.Fichiers, Valeur: ve, Part: part(ve, total)})
			}
		}
		sortValeurs(exts)
		r.Extensions = exts[:min(len(exts), 15)]
	}

	// Âge
	if ages, err := sc.Ages(o.Dans); err == nil {
		var total int64
		for _, a := range ages {
			total += poids(a.Taille, a.Disque, mode)
		}
		for _, a := range ages {
			v := poids(a.Taille, a.Disque, mode)
			r.Ages = append(r.Ages, ligneValeur{Nom: a.Nom, Fichiers: a.Fichiers, Valeur: v, Part: part(v, total)})
		}
	}

	// Plus gros fichiers (relevés sur toute l'analyse : on garde ceux du dossier)
	if gros, err := sc.GrosFichiers(mode, scan.Tri{}); err == nil {
		for _, g := range gros {
			if len(r.Gros) >= maxGros {
				break
			}
			if !r.Partiel || dedans(g.Dossier, r.Chemin) {
				r.Gros = append(r.Gros, g)
			}
		}
	}

	// Doublons, s'ils ont été cherchés sur cette analyse
	if d != nil {
		if su := d.Suivi(); su.Etat == scan.Fini {
			if groupes, err := d.Groupes(); err == nil {
				r.AvecDoublons = true
				for _, g := range groupes {
					r.DblGroupes++
					r.DblCopies += len(g.Fichiers) - 1
					r.DblPlace += g.Recuperable
					if len(r.Doublons) < maxGroupes {
						gr := groupeRapport{Copies: len(g.Fichiers), Taille: g.Taille, Recuperable: g.Recuperable, Nom: g.Fichiers[0].Nom}
						for _, f := range g.Fichiers {
							gr.Dossiers = append(gr.Dossiers, f.Dossier)
						}
						r.Doublons = append(r.Doublons, gr)
					}
				}
			}
		}
	}

	return rapportTmpl.Execute(w, r)
}

func part(v, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(v) / float64(total)
}

func sortValeurs(l []ligneValeur) {
	for i := 1; i < len(l); i++ {
		for j := i; j > 0 && l[j].Valeur > l[j-1].Valeur; j-- {
			l[j], l[j-1] = l[j-1], l[j]
		}
	}
}

func dedans(chemin, dossier string) bool {
	rel, err := filepath.Rel(dossier, chemin)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func decrireFiltres(f scan.Filtres) string {
	var p []string
	if len(f.Inclure) > 0 {
		p = append(p, "seulement "+strings.Join(f.Inclure, ", "))
	}
	if len(f.Exclure) > 0 {
		p = append(p, "sans "+strings.Join(f.Exclure, ", "))
	}
	if f.TailleMin > 0 {
		p = append(p, "fichiers de "+Lisible(f.TailleMin)+" ou plus")
	}
	if f.Regex != "" {
		p = append(p, "noms correspondant à "+f.Regex)
	}
	if f.AgeMinJours > 0 {
		p = append(p, fmt.Sprintf("fichiers modifiés il y a %d jours ou plus", f.AgeMinJours))
	}
	if len(p) == 0 {
		return "filtres actifs"
	}
	return strings.Join(p, " · ")
}

// ---------- Carte (treemap « squarified ») ----------

type rect struct{ x, y, w, h float64 }

// squarifier découpe le rectangle en aires proportionnelles aux valeurs,
// les plus carrées possible (Bruls, Huizing, van Wijk), comme dans l'appli.
func squarifier(valeurs []float64, x, y, w, h float64) []rect {
	out := make([]rect, 0, len(valeurs))
	var total float64
	for _, v := range valeurs {
		total += v
	}
	if total <= 0 || w <= 0 || h <= 0 {
		for range valeurs {
			out = append(out, rect{x, y, 0, 0})
		}
		return out
	}
	echelle := w * h / total
	aires := make([]float64, len(valeurs))
	for i, v := range valeurs {
		aires[i] = v * echelle
	}
	pire := func(rangee []float64, cote float64) float64 {
		var s, mn, mx float64
		mn = math.Inf(1)
		for _, a := range rangee {
			s += a
			mn = math.Min(mn, a)
			mx = math.Max(mx, a)
		}
		s2, c2 := s*s, cote*cote
		return math.Max(c2*mx/s2, s2/(c2*mn))
	}
	for i := 0; i < len(aires); {
		cote := math.Min(w, h)
		j := i + 1
		courant := pire(aires[i:j], cote)
		for j < len(aires) && aires[j] > 0 {
			p := pire(aires[i:j+1], cote)
			if p > courant {
				break
			}
			courant = p
			j++
		}
		var somme float64
		for _, a := range aires[i:j] {
			somme += a
		}
		if w >= h {
			l := somme / h
			yy := y
			for _, a := range aires[i:j] {
				out = append(out, rect{x, yy, l, a / l})
				yy += a / l
			}
			x += l
			w -= l
		} else {
			ht := somme / w
			xx := x
			for _, a := range aires[i:j] {
				out = append(out, rect{xx, y, a / ht, ht})
				xx += a / ht
			}
			y += ht
			h -= ht
		}
		i = j
	}
	return out
}

// carte dessine deux niveaux : les dossiers du premier niveau, et dedans
// ce qu'ils contiennent.
func carte(b scan.Bloc, w, h float64) []blocCarte {
	var out []blocCarte
	enfants := b.Enfants
	valeurs := make([]float64, len(enfants))
	for i, e := range enfants {
		valeurs[i] = float64(e.Valeur)
	}
	couleur := func(i int, e scan.Bloc) string {
		switch e.Type {
		case "fichiers":
			return grisFichiers
		case "reste":
			return grisReste
		}
		return teintes[i%len(teintes)]
	}
	for i, rc := range squarifier(valeurs, 0, 0, w, h) {
		e := enfants[i]
		c := couleur(i, e)
		nom := e.Nom
		switch e.Type {
		case "fichiers":
			nom = "fichiers"
		case "reste":
			nom = fmt.Sprintf("%d autres", e.Nombre)
		}
		titre := rc.w > 60 && rc.h > 40 && len(e.Enfants) > 0
		out = append(out, blocCarte{X: rc.x, Y: rc.y, W: rc.w, H: rc.h, Couleur: c, Nom: nom, Taille: Lisible(e.Valeur),
			Etiquette: rc.w > 64 && rc.h > 30, Titre: titre, Opacite: 0.16})
		if !titre {
			continue
		}
		// Le contenu, sous un bandeau de 18 px pour le nom
		const marge, bandeau = 2.0, 18.0
		sous := e.Enfants
		sv := make([]float64, len(sous))
		for k, s := range sous {
			sv[k] = float64(s.Valeur)
		}
		for k, sr := range squarifier(sv, rc.x+marge, rc.y+bandeau, rc.w-2*marge, rc.h-bandeau-marge) {
			s := sous[k]
			sc := c
			if s.Type == "fichiers" {
				sc = grisFichiers
			} else if s.Type == "reste" {
				sc = grisReste
			}
			sn := s.Nom
			if s.Type == "fichiers" {
				sn = "fichiers"
			} else if s.Type == "reste" {
				sn = fmt.Sprintf("%d autres", s.Nombre)
			}
			out = append(out, blocCarte{X: sr.x, Y: sr.y, W: sr.w, H: sr.h, Couleur: sc, Nom: sn, Taille: Lisible(s.Valeur),
				Etiquette: sr.w > 64 && sr.h > 30, Opacite: 0.42})
		}
	}
	return out
}
