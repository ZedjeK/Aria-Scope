package scan

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"regexp/syntax"
	"strings"
)

// Filtres restreint ce que le scan compte. Ils sont appliqués pendant la
// lecture : arbre, carte, gros fichiers et statistiques restent cohérents.
type Filtres struct {
	// Exclure : motifs de noms (dossiers ou fichiers) à ignorer complètement,
	// par exemple « node_modules », « *.iso », « $RECYCLE.BIN ». Un dossier
	// exclu n'est même pas parcouru.
	Exclure []string `json:"exclure"`
	// Inclure : si la liste n'est pas vide, seuls les fichiers dont le nom
	// correspond à l'un des motifs sont comptés (« *.mkv », « *.mp4 »).
	Inclure []string `json:"inclure"`
	// Regex : expression régulière que le nom du fichier doit contenir
	// (insensible à la casse).
	Regex string `json:"regex"`
	// TailleMin : les fichiers plus petits (en octets) sont ignorés.
	TailleMin int64 `json:"tailleMin"`
	// AgeMinJours : les fichiers modifiés il y a moins de N jours sont ignorés.
	AgeMinJours int64 `json:"ageMinJours"`
}

// Actifs indique si au moins un filtre restreint le scan.
func (f Filtres) Actifs() bool {
	return len(f.Exclure) > 0 || len(f.Inclure) > 0 || f.Regex != "" || f.TailleMin > 0 || f.AgeMinJours > 0
}

// filtre est la version compilée de Filtres, prête pour le scan.
type filtre struct {
	exclure, inclure []string // motifs en minuscules
	regex            *regexp.Regexp
	tailleMin        int64
	modifMax         int64 // date de modification la plus récente admise (0 : pas de limite)
	actif            bool
	// elaguer : un filtre porte sur les fichiers eux-mêmes ; on retire alors
	// de l'arbre les dossiers où aucun n'est retenu. Une simple exclusion
	// laisse les dossiers vides à leur place.
	elaguer bool
}

func nettoyer(motifs []string) ([]string, error) {
	var out []string
	for _, m := range motifs {
		m = strings.ToLower(strings.TrimSpace(m))
		if m == "" {
			continue
		}
		if strings.ContainsAny(m, `/\`) {
			return nil, fmt.Errorf("motif « %s » : seuls les noms sont acceptés, pas les chemins", m)
		}
		if _, err := filepath.Match(m, ""); err != nil {
			return nil, fmt.Errorf("motif « %s » invalide", m)
		}
		out = append(out, m)
	}
	return out, nil
}

// compiler vérifie les filtres et les prépare ; ref est la date du scan.
func (f Filtres) compiler(ref int64) (*filtre, error) {
	c := &filtre{tailleMin: max(0, f.TailleMin)}
	var err error
	if c.exclure, err = nettoyer(f.Exclure); err != nil {
		return nil, err
	}
	if c.inclure, err = nettoyer(f.Inclure); err != nil {
		return nil, err
	}
	if r := strings.TrimSpace(f.Regex); r != "" {
		if _, err := syntax.Parse(r, syntax.Perl); err != nil {
			return nil, erreurRegex(err)
		}
		if c.regex, err = regexp.Compile("(?i)" + r); err != nil {
			return nil, erreurRegex(err)
		}
	}
	if f.AgeMinJours > 0 {
		c.modifMax = ref - f.AgeMinJours*86400
	}
	c.elaguer = len(c.inclure) > 0 || c.regex != nil || c.tailleMin > 0 || c.modifMax > 0
	c.actif = c.elaguer || len(c.exclure) > 0
	return c, nil
}

// erreurRegex traduit les erreurs les plus courantes d'une expression régulière.
func erreurRegex(err error) error {
	var se *syntax.Error
	if !errors.As(err, &se) {
		return fmt.Errorf("expression régulière invalide : %v", err)
	}
	quoi := map[syntax.ErrorCode]string{
		syntax.ErrMissingParen:          "parenthèse non fermée",
		syntax.ErrUnexpectedParen:       "parenthèse fermante en trop",
		syntax.ErrMissingBracket:        "crochet non fermé",
		syntax.ErrMissingRepeatArgument: "« * », « + » ou « ? » sans rien devant",
		syntax.ErrInvalidRepeatOp:       "répétition mal placée",
		syntax.ErrInvalidRepeatSize:     "nombre de répétitions invalide",
		syntax.ErrInvalidEscape:         "échappement « \\ » inconnu",
		syntax.ErrTrailingBackslash:     "« \\ » tout à la fin",
		syntax.ErrInvalidCharRange:      "plage de caractères à l'envers",
		syntax.ErrInvalidCharClass:      "classe de caractères inconnue",
		syntax.ErrInvalidPerlOp:         "groupe « (? » non reconnu",
	}[se.Code]
	if quoi == "" {
		quoi = se.Code.String()
	}
	return fmt.Errorf("expression régulière invalide : %s (près de « %s »)", quoi, se.Expr)
}

// normalises renvoie les filtres tels qu'ils seront appliqués (motifs vides
// retirés, en minuscules), pour l'affichage.
func (c *filtre) normalises(f Filtres) Filtres {
	return Filtres{
		Exclure:     c.exclure,
		Inclure:     c.inclure,
		Regex:       strings.TrimSpace(f.Regex),
		TailleMin:   c.tailleMin,
		AgeMinJours: max(0, f.AgeMinJours),
	}
}

func correspond(motifs []string, nom string) bool {
	for _, m := range motifs {
		if ok, _ := filepath.Match(m, nom); ok {
			return true
		}
	}
	return false
}

// garderDossier : le dossier doit-il être parcouru ?
func (c *filtre) garderDossier(nom string) bool {
	return !c.actif || len(c.exclure) == 0 || !correspond(c.exclure, strings.ToLower(nom))
}

// garderFichier : le fichier doit-il être compté ?
func (c *filtre) garderFichier(e entree) bool {
	if !c.actif {
		return true
	}
	if e.Taille < c.tailleMin {
		return false
	}
	if c.modifMax > 0 && e.Modif > c.modifMax {
		return false
	}
	if len(c.exclure) > 0 || len(c.inclure) > 0 {
		nom := strings.ToLower(e.Nom)
		if correspond(c.exclure, nom) {
			return false
		}
		if len(c.inclure) > 0 && !correspond(c.inclure, nom) {
			return false
		}
	}
	return c.regex == nil || c.regex.MatchString(e.Nom)
}

// elaguer retire, sous d, les dossiers où aucun fichier n'a été retenu :
// avec un filtre « *.mkv », l'arbre ne montre que les dossiers qui en ont.
// Les dossiers illisibles restent, pour qu'on sache où le scan n'a pas pu lire.
func elaguer(d *Dossier) bool {
	garde := d.enfants[:0]
	for _, c := range d.enfants {
		if elaguer(c) {
			garde = append(garde, c)
		}
	}
	clear(d.enfants[len(garde):])
	d.enfants = garde
	return len(d.enfants) > 0 || d.nFichiers > 0 || d.illisible
}
