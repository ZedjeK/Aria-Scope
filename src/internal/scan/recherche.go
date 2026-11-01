package scan

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"regexp/syntax"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf16"
)

// MaxTrouves : nombre de résultats gardés ; au-delà, seuls les plus lourds
// restent (le total trouvé est toujours compté).
const MaxTrouves = 10000

var plafond = MaxTrouves // réduit dans les tests

// Ce que l'on cherche.
const (
	QuoiFichiers = "fichiers"
	QuoiDossiers = "dossiers"
	QuoiTout     = "tout"
)

// Criteres : ce que doit vérifier un résultat. Un critère vide ne restreint rien.
type Criteres struct {
	// Nom : des motifs séparés par « ; ». Avec joker (« *.mkv », « IMG_????.jpg »)
	// le nom entier doit correspondre ; sans joker, il suffit qu'il contienne
	// le texte. Insensible à la casse.
	Nom string `json:"nom"`
	// Regex : Nom est une expression régulière (insensible à la casse).
	Regex      bool     `json:"regex"`
	Quoi       string   `json:"quoi"`       // fichiers (défaut), dossiers, tout
	TailleMin  int64    `json:"tailleMin"`  // octets
	TailleMax  int64    `json:"tailleMax"`  // octets, 0 : pas de limite
	ModifApres int64    `json:"modifApres"` // secondes Unix, 0 : pas de limite
	ModifAvant int64    `json:"modifAvant"` // secondes Unix, 0 : pas de limite
	Familles   []string `json:"familles"`   // familles de types (fichiers seulement)
	// CheminMin : longueur minimale du chemin complet, en caractères comme
	// les compte Windows (UTF-16). 260 : la limite historique MAX_PATH.
	CheminMin int `json:"cheminMin"`
}

type critere struct {
	quoi              string
	motifs, contenus  []string // en minuscules
	regex             *regexp.Regexp
	tailleMin, taille int64 // taille : maximum, 0 sans limite
	apres, avant      int64
	familles          map[string]bool
	cheminMin         int
}

func (c Criteres) compiler() (*critere, error) {
	k := &critere{
		quoi: c.Quoi, tailleMin: max(0, c.TailleMin), taille: max(0, c.TailleMax),
		apres: max(0, c.ModifApres), avant: max(0, c.ModifAvant), cheminMin: max(0, c.CheminMin),
	}
	switch k.quoi {
	case "":
		k.quoi = QuoiFichiers
	case QuoiFichiers, QuoiDossiers, QuoiTout:
	default:
		return nil, fmt.Errorf("type de recherche inconnu : %s", c.Quoi)
	}
	if k.taille > 0 && k.taille < k.tailleMin {
		return nil, errors.New("la taille maximale est plus petite que la taille minimale")
	}
	if k.avant > 0 && k.avant < k.apres {
		return nil, errors.New("la période se termine avant de commencer")
	}
	nom := strings.TrimSpace(c.Nom)
	if c.Regex && nom != "" {
		if _, err := syntax.Parse(nom, syntax.Perl); err != nil {
			return nil, erreurRegex(err)
		}
		r, err := regexp.Compile("(?i)" + nom)
		if err != nil {
			return nil, erreurRegex(err)
		}
		k.regex = r
	} else {
		for _, m := range strings.Split(nom, ";") {
			m = strings.ToLower(strings.TrimSpace(m))
			switch {
			case m == "":
			case strings.ContainsAny(m, "*?["):
				if _, err := filepath.Match(m, ""); err != nil {
					return nil, fmt.Errorf("motif « %s » invalide", m)
				}
				k.motifs = append(k.motifs, m)
			default:
				k.contenus = append(k.contenus, m)
			}
		}
	}
	if len(c.Familles) > 0 {
		k.familles = map[string]bool{}
		for _, f := range c.Familles {
			k.familles[f] = true
		}
	}
	return k, nil
}

// nomOK : le nom correspond à l'un des motifs (aucun motif : tout correspond).
func (k *critere) nomOK(nom string) bool {
	if k.regex != nil {
		return k.regex.MatchString(nom)
	}
	if len(k.motifs) == 0 && len(k.contenus) == 0 {
		return true
	}
	bas := strings.ToLower(nom)
	for _, c := range k.contenus {
		if strings.Contains(bas, c) {
			return true
		}
	}
	return correspond(k.motifs, bas)
}

func (k *critere) tailleOK(t int64) bool {
	return t >= k.tailleMin && (k.taille == 0 || t <= k.taille)
}

func (k *critere) dateOK(m int64) bool {
	if k.apres == 0 && k.avant == 0 {
		return true
	}
	return m > 0 && m >= k.apres && (k.avant == 0 || m <= k.avant)
}

// longueur : nombre de caractères UTF-16 de s, comme les compte Windows.
func longueur(s string) int {
	n := 0
	for _, r := range s {
		n += utf16.RuneLen(r)
	}
	return n
}

// Trouve : une ligne de résultat. Comme ailleurs, l'ID d'un fichier est
// celui de son dossier.
type Trouve struct {
	Entree
	Dossier  string `json:"dossier"`  // dossier qui le contient
	Longueur int    `json:"longueur"` // longueur du chemin complet
}

type trouve struct {
	d        *Dossier // le dossier trouvé, ou celui qui contient le fichier
	fichier  bool
	nom      string
	taille   int64
	disque   int64
	modif    int64
	longueur int
}

// tasTrouves : le plus léger en tête, pour l'évincer quand le plafond est atteint.
type tasTrouves []trouve

func (t tasTrouves) Len() int           { return len(t) }
func (t tasTrouves) Less(i, j int) bool { return t[i].taille < t[j].taille }
func (t tasTrouves) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t *tasTrouves) Push(x any)        { *t = append(*t, x.(trouve)) }
func (t *tasTrouves) Pop() any {
	old := *t
	x := old[len(old)-1]
	*t = old[:len(old)-1]
	return x
}

// SuiviRecherche : où en est une recherche.
type SuiviRecherche struct {
	Etat       string `json:"etat"`     // EnCours, Fini ou Arrete
	Dossiers   int64  `json:"dossiers"` // dossiers à relire
	Lus        int64  `json:"lus"`
	Trouves    int64  `json:"trouves"` // toutes les correspondances, même au-delà du plafond
	Gardes     int    `json:"gardes"`  // celles qui sont gardées (MaxTrouves au plus)
	Taille     int64  `json:"taille"`  // poids de toutes les correspondances
	Disque     int64  `json:"disque"`
	Illisibles int64  `json:"illisibles"`
	DureeMs    int64  `json:"dureeMs"`
}

// Recherche : une recherche lancée sur l'analyse, qui tourne en arrière-plan.
type Recherche struct {
	sc      *Scan
	k       *critere
	debut   time.Time
	arreter context.CancelFunc
	fini    chan struct{}

	dossiers                                 int64
	lus, trouves, taille, disque, illisibles atomic.Int64
	mu                                       sync.Mutex
	tas                                      tasTrouves
	etat                                     string
	duree                                    time.Duration
}

type tacheRecherche struct {
	d      *Dossier
	chemin string
	long   int // longueur du chemin du dossier
}

// dossiersSous dresse la liste des dossiers connus sous id (lui compris),
// avec leur chemin, en sautant ce que le ménage a retiré ; visite (facultatif)
// voit chacun passer. L'arbre n'est verrouillé que le temps de ce relevé : un
// ménage peut avoir lieu pendant la relecture qui suit.
func (s *Scan) dossiersSous(id int32, visite func(t tacheRecherche, depart *Dossier)) ([]tacheRecherche, error) {
	s.arbre.RLock()
	defer s.arbre.RUnlock()
	depart, err := s.noeud(id)
	if err != nil {
		return nil, err
	}
	var pile, taches []tacheRecherche
	if id == 0 && s.pc {
		for _, c := range depart.enfants {
			pile = append(pile, tacheRecherche{c, c.nom, longueur(c.nom)})
		}
	} else {
		ch, err := s.chemin(id)
		if err != nil {
			return nil, err
		}
		pile = append(pile, tacheRecherche{depart, ch, longueur(ch)})
	}
	for len(pile) > 0 {
		t := pile[len(pile)-1]
		pile = pile[:len(pile)-1]
		if t.d.retire {
			continue
		}
		taches = append(taches, t)
		if visite != nil {
			visite(t, depart)
		}
		for _, e := range t.d.enfants {
			pile = append(pile, tacheRecherche{e, filepath.Join(t.chemin, e.nom), t.long + 1 + longueur(e.nom)})
		}
	}
	return taches, nil
}

// Chercher lance une recherche dans le dossier id (0 : toute l'analyse).
// Les dossiers connus de l'analyse sont relus en parallèle ; les filtres de
// l'analyse s'appliquent aussi, pour que les chiffres restent cohérents.
func (s *Scan) Chercher(id int32, c Criteres) (*Recherche, error) {
	k, err := c.compiler()
	if err != nil {
		return nil, err
	}

	var dossiersTrouves []trouve
	taches, err := s.dossiersSous(id, func(t tacheRecherche, depart *Dossier) {
		if k.quoi != QuoiFichiers && t.d != depart && k.familles == nil &&
			k.nomOK(t.d.nom) && k.tailleOK(t.d.taille) && k.dateOK(t.d.modif) && t.long >= k.cheminMin {
			dossiersTrouves = append(dossiersTrouves, trouve{d: t.d, nom: t.d.nom, taille: t.d.taille, disque: t.d.disque, modif: t.d.modif, longueur: t.long})
		}
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	r := &Recherche{sc: s, k: k, debut: time.Now(), arreter: cancel, fini: make(chan struct{}), etat: EnCours}
	for _, t := range dossiersTrouves {
		r.ajouter(t)
	}
	if k.quoi == QuoiDossiers {
		r.dossiers = 0
		r.terminer(ctx)
		return r, nil
	}
	r.dossiers = int64(len(taches))

	file := make(chan tacheRecherche, 256)
	var wg sync.WaitGroup
	for range Parallelisme() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range file {
				if ctx.Err() == nil {
					r.lire(t)
				}
				r.lus.Add(1)
			}
		}()
	}
	go func() {
		for _, t := range taches {
			if ctx.Err() != nil {
				break
			}
			file <- t
		}
		close(file)
		wg.Wait()
		r.terminer(ctx)
	}()
	return r, nil
}

// lire relit un dossier et retient les fichiers qui correspondent.
func (r *Recherche) lire(t tacheRecherche) {
	k, s := r.k, r.sc
	err := listerDossier(t.chemin, func(e entree) {
		if e.Dossier || e.Lien || !s.filtre.garderFichier(e) {
			return
		}
		if !k.tailleOK(e.Taille) || !k.dateOK(e.Modif) || !k.nomOK(e.Nom) {
			return
		}
		long := t.long + 1 + longueur(e.Nom)
		if long < k.cheminMin {
			return
		}
		if k.familles != nil && !k.familles[famille(extension(e.Nom))] {
			return
		}
		r.ajouter(trouve{d: t.d, fichier: true, nom: e.Nom, taille: e.Taille, disque: e.Disque, modif: e.Modif, longueur: long})
	})
	if err != nil {
		r.illisibles.Add(1)
	}
}

// famille : la famille d'une extension, comme dans l'onglet des types.
func famille(ext string) string {
	if ext == "" {
		return familleSans
	}
	if f, ok := familleDe[ext]; ok {
		return f
	}
	return familleAutre
}

func (r *Recherche) ajouter(t trouve) {
	r.trouves.Add(1)
	r.taille.Add(t.taille)
	r.disque.Add(t.disque)
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.tas) < plafond {
		heap.Push(&r.tas, t)
		return
	}
	if t.taille > r.tas[0].taille {
		r.tas[0] = t
		heap.Fix(&r.tas, 0)
	}
}

func (r *Recherche) terminer(ctx context.Context) {
	r.mu.Lock()
	r.duree = time.Since(r.debut)
	r.etat = Fini
	if ctx.Err() != nil {
		r.etat = Arrete
	}
	r.mu.Unlock()
	r.arreter()
	close(r.fini)
}

// Arreter interrompt la recherche ; ce qui a été trouvé reste consultable.
func (r *Recherche) Arreter() { r.arreter() }

// Attendre bloque jusqu'à la fin de la recherche.
func (r *Recherche) Attendre() { <-r.fini }

// Suivi renvoie l'avancement.
func (r *Recherche) Suivi() SuiviRecherche {
	r.mu.Lock()
	etat, duree, gardes := r.etat, r.duree, len(r.tas)
	r.mu.Unlock()
	if etat == EnCours {
		duree = time.Since(r.debut)
	}
	return SuiviRecherche{
		Etat: etat, Dossiers: r.dossiers, Lus: r.lus.Load(),
		Trouves: r.trouves.Load(), Gardes: gardes,
		Taille: r.taille.Load(), Disque: r.disque.Load(),
		Illisibles: r.illisibles.Load(), DureeMs: duree.Milliseconds(),
	}
}

// Resultats renvoie ce qui a été trouvé, trié selon t. Ce qui a été retiré
// depuis par le ménage (corbeille, déplacement) n'y figure plus.
func (r *Recherche) Resultats(mode string, t Tri) ([]Trouve, error) {
	r.mu.Lock()
	tous := make([]trouve, len(r.tas))
	copy(tous, r.tas)
	r.mu.Unlock()

	s := r.sc
	s.arbre.RLock()
	defer s.arbre.RUnlock()
	lignes := make([]Entree, 0, len(tous))
	longueurs := map[string]int{}
	cle := func(l Entree) string { return l.Type + ":" + fmt.Sprint(l.ID) + "/" + l.Nom }
	for _, x := range tous {
		if _, err := s.noeud(x.d.id); err != nil {
			continue
		}
		var l Entree
		if x.fichier {
			if s.fichiersRetires[cleCible(Cible{Type: "fichier", ID: x.d.id, Nom: x.nom})] {
				continue
			}
			l = Entree{ID: x.d.id, Type: "fichier", Nom: x.nom, Taille: x.taille, Disque: x.disque, Modif: x.modif, Fichiers: 1}
		} else {
			// Un dossier : ses chiffres à jour, après un éventuel ménage dedans.
			l = entreeDossier(x.d)
		}
		lignes = append(lignes, l)
		longueurs[cle(l)] = x.longueur
	}
	t.trier(lignes, mode)

	chemins := map[int32]string{}
	cheminDe := func(id int32) string {
		c, ok := chemins[id]
		if !ok {
			c, _ = s.chemin(id)
			chemins[id] = c
		}
		return c
	}
	out := make([]Trouve, len(lignes))
	for i, l := range lignes {
		dossier := cheminDe(l.ID)
		if l.Type == "dossier" {
			dossier = filepath.Dir(dossier)
			if d, _ := s.noeud(l.ID); d != nil && d.parent != nil && d.parent.parent == nil && s.pc {
				dossier = "" // un disque, directement sous « Ce PC »
			}
		}
		out[i] = Trouve{Entree: l, Dossier: dossier, Longueur: longueurs[cle(l)]}
	}
	return out, nil
}

// NomsFamilles : les familles de types, dans l'ordre alphabétique, puis
// « Autres » et « Sans extension ».
func NomsFamilles() []string {
	noms := make([]string, 0, len(familles)+2)
	for f := range familles {
		noms = append(noms, f)
	}
	slices.Sort(noms)
	return append(noms, familleAutre, familleSans)
}
