package scan

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

// Recherche de doublons, en trois passes pour ne lire que le nécessaire :
//  1. les fichiers sont regroupés par taille (aucune lecture) ;
//  2. une empreinte rapide (début et fin du fichier) écarte la plupart des
//     faux candidats ;
//  3. un SHA-256 complet confirme ce qui reste.
// Deux noms pour le même fichier (lien physique) ne sont pas des doublons.

// Taille lue au début et à la fin d'un fichier pour l'empreinte rapide.
const bout = 64 << 10

// MaxGroupes : nombre de groupes renvoyés au plus, les plus rentables d'abord.
const MaxGroupes = 5000

// Étapes d'une recherche de doublons.
const (
	PhaseInventaire = "inventaire"
	PhaseRapide     = "rapide"
	PhaseComplete   = "complete"
)

// SuiviDoublons : où en est la recherche de doublons.
type SuiviDoublons struct {
	Etat        string `json:"etat"`  // EnCours, Fini ou Arrete
	Phase       string `json:"phase"` // inventaire, rapide, complete
	Dossiers    int64  `json:"dossiers"`
	Lus         int64  `json:"lus"`
	Candidats   int64  `json:"candidats"` // fichiers assez gros pour compter
	AHacher     int64  `json:"aHacher"`   // octets à lire dans la phase en cours
	Haches      int64  `json:"haches"`
	Groupes     int    `json:"groupes"`
	Copies      int    `json:"copies"`      // copies en trop (toutes sauf une par groupe)
	Recuperable int64  `json:"recuperable"` // place qu'elles occupent
	Illisibles  int64  `json:"illisibles"`  // fichiers ou dossiers qui n'ont pas pu être lus
	DureeMs     int64  `json:"dureeMs"`
}

// GroupeDoublons : des fichiers au contenu identique.
type GroupeDoublons struct {
	Taille      int64    `json:"taille"`      // taille d'un exemplaire
	Empreinte   string   `json:"empreinte"`   // SHA-256, en hexadécimal
	Recuperable int64    `json:"recuperable"` // en ne gardant qu'un exemplaire
	Fichiers    []Trouve `json:"fichiers"`    // le plus ancien d'abord
}

type candidat struct {
	d       *Dossier
	nom     string
	chemin  string
	taille  int64
	disque  int64
	modif   int64
	rapide  [32]byte
	complet [32]byte
}

// Doublons : une recherche de doublons qui tourne en arrière-plan.
type Doublons struct {
	sc        *Scan
	tailleMin int64
	debut     time.Time
	arreter   context.CancelFunc
	fini      chan struct{}

	dossiers                                    int64
	lus, candidats, aHacher, haches, illisibles atomic.Int64
	mu                                          sync.Mutex
	phase, etat                                 string
	duree                                       time.Duration
	groupes                                     [][]*candidat
}

// ChercherDoublons lance la recherche des fichiers identiques dans le dossier
// id (0 : toute l'analyse), en ignorant ceux de moins de tailleMin octets.
func (s *Scan) ChercherDoublons(id int32, tailleMin int64) (*Doublons, error) {
	taches, err := s.dossiersSous(id, nil)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	d := &Doublons{
		sc: s, tailleMin: max(1, tailleMin), debut: time.Now(), arreter: cancel,
		fini: make(chan struct{}), dossiers: int64(len(taches)), phase: PhaseInventaire, etat: EnCours,
	}
	go d.executer(ctx, taches)
	return d, nil
}

func (d *Doublons) etape(p string, aHacher int64) {
	d.mu.Lock()
	d.phase = p
	d.mu.Unlock()
	d.haches.Store(0)
	d.aHacher.Store(aHacher)
}

func (d *Doublons) executer(ctx context.Context, taches []tacheRecherche) {
	defer func() {
		d.mu.Lock()
		d.duree = time.Since(d.debut)
		d.etat = Fini
		if ctx.Err() != nil {
			d.etat = Arrete
		}
		d.mu.Unlock()
		d.arreter()
		close(d.fini)
	}()

	// 1. Inventaire, regroupé par taille
	tous := d.inventorier(ctx, taches)
	parTaille := map[int64][]*candidat{}
	for _, c := range tous {
		parTaille[c.taille] = append(parTaille[c.taille], c)
	}
	var groupes [][]*candidat
	for _, g := range parTaille {
		if len(g) > 1 {
			groupes = append(groupes, g)
		}
	}
	if ctx.Err() != nil {
		return
	}

	// 2. Empreinte rapide : le début et la fin. Un petit fichier est lu en
	// entier, et son empreinte rapide est déjà la complète.
	var aLire int64
	for _, g := range groupes {
		for _, c := range g {
			aLire += min(c.taille, 2*bout)
		}
	}
	d.etape(PhaseRapide, aLire)
	groupes = d.hacher(ctx, groupes, false)
	if ctx.Err() != nil {
		return
	}

	// 3. SHA-256 complet des gros fichiers restants
	aLire = 0
	for _, g := range groupes {
		if g[0].taille > 2*bout {
			aLire += g[0].taille * int64(len(g))
		}
	}
	d.etape(PhaseComplete, aLire)
	groupes = d.hacher(ctx, groupes, true)
	if ctx.Err() != nil {
		return
	}

	// Liens physiques : deux noms, un seul fichier sur le disque.
	var final [][]*candidat
	for _, g := range groupes {
		if g = sansLiensPhysiques(g); len(g) > 1 {
			final = append(final, g)
		}
	}
	d.mu.Lock()
	d.groupes = final
	d.mu.Unlock()
}

// inventorier relit les dossiers en parallèle et garde les fichiers assez gros.
func (d *Doublons) inventorier(ctx context.Context, taches []tacheRecherche) []*candidat {
	s := d.sc
	var mu sync.Mutex
	var tous []*candidat
	file := make(chan tacheRecherche, 256)
	var wg sync.WaitGroup
	for range Parallelisme() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range file {
				if ctx.Err() == nil {
					var locaux []*candidat
					err := listerDossier(t.chemin, func(e entree) {
						if e.Dossier || e.Lien || e.Taille < d.tailleMin || !s.filtre.garderFichier(e) {
							return
						}
						locaux = append(locaux, &candidat{d: t.d, nom: e.Nom, chemin: filepath.Join(t.chemin, e.Nom), taille: e.Taille, disque: e.Disque, modif: e.Modif})
					})
					if err != nil {
						d.illisibles.Add(1)
					}
					d.candidats.Add(int64(len(locaux)))
					mu.Lock()
					tous = append(tous, locaux...)
					mu.Unlock()
				}
				d.lus.Add(1)
			}
		}()
	}
	for _, t := range taches {
		if ctx.Err() != nil {
			break
		}
		file <- t
	}
	close(file)
	wg.Wait()
	return tous
}

// Lectures en parallèle pendant le hachage : assez pour un SSD, pas trop
// pour un disque dur qui n'aime pas sauter d'un fichier à l'autre.
const lecteursHachage = 4

// hacher calcule l'empreinte (rapide ou complète) des candidats et renvoie
// les groupes qui restent identiques. Un fichier illisible ou qui a changé
// de taille depuis l'inventaire est écarté.
func (d *Doublons) hacher(ctx context.Context, groupes [][]*candidat, complet bool) [][]*candidat {
	var mu sync.Mutex
	ecartes := map[*candidat]bool{}
	file := make(chan *candidat, 64)
	var wg sync.WaitGroup
	for range lecteursHachage {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tampon := make([]byte, 1<<20)
			for c := range file {
				if ctx.Err() != nil {
					continue
				}
				var err error
				if complet {
					err = d.empreinteComplete(ctx, c, tampon)
				} else {
					err = d.empreinteRapide(c)
				}
				if err != nil {
					if !errors.Is(err, context.Canceled) {
						d.illisibles.Add(1)
					}
					mu.Lock()
					ecartes[c] = true
					mu.Unlock()
				}
			}
		}()
	}
	for _, g := range groupes {
		for _, c := range g {
			if complet && c.taille <= 2*bout {
				continue // déjà lu en entier à l'étape rapide
			}
			if ctx.Err() != nil {
				break
			}
			file <- c
		}
	}
	close(file)
	wg.Wait()

	var out [][]*candidat
	for _, g := range groupes {
		parCle := map[[32]byte][]*candidat{}
		var ordre [][32]byte
		for _, c := range g {
			if ecartes[c] {
				continue
			}
			cle := c.rapide
			if complet {
				cle = c.complet
			}
			if _, vu := parCle[cle]; !vu {
				ordre = append(ordre, cle)
			}
			parCle[cle] = append(parCle[cle], c)
		}
		for _, cle := range ordre {
			if len(parCle[cle]) > 1 {
				out = append(out, parCle[cle])
			}
		}
	}
	return out
}

var errChange = errors.New("le fichier a changé pendant la recherche")

func (d *Doublons) empreinteRapide(c *candidat) error {
	f, err := os.Open(c.chemin)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if c.taille <= 2*bout {
		n, err := io.Copy(h, f)
		if err != nil {
			return err
		}
		if n != c.taille {
			return errChange
		}
		d.haches.Add(n)
		copy(c.rapide[:], h.Sum(nil))
		c.complet = c.rapide
		return nil
	}
	tampon := make([]byte, bout)
	if _, err := io.ReadFull(f, tampon); err != nil {
		return err
	}
	h.Write(tampon)
	if _, err := f.ReadAt(tampon, c.taille-bout); err != nil {
		return err
	}
	h.Write(tampon)
	d.haches.Add(2 * bout)
	copy(c.rapide[:], h.Sum(nil))
	return nil
}

func (d *Doublons) empreinteComplete(ctx context.Context, c *candidat, tampon []byte) error {
	f, err := os.Open(c.chemin)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	var n int64
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		k, err := f.Read(tampon)
		if k > 0 {
			h.Write(tampon[:k])
			n += int64(k)
			d.haches.Add(int64(k))
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	if n != c.taille {
		return errChange
	}
	copy(c.complet[:], h.Sum(nil))
	return nil
}

// sansLiensPhysiques garde un seul nom par fichier réel : le premier chemin
// dans l'ordre alphabétique, pour que le résultat ne dépende pas du hasard
// de la lecture parallèle.
func sansLiensPhysiques(g []*candidat) []*candidat {
	g = slices.Clone(g)
	slices.SortFunc(g, func(a, b *candidat) int { return cmp.Compare(a.chemin, b.chemin) })
	var garde []*candidat
	var infos []os.FileInfo
	for _, c := range g {
		info, err := os.Stat(c.chemin)
		if err != nil {
			continue
		}
		if slices.ContainsFunc(infos, func(i os.FileInfo) bool { return os.SameFile(i, info) }) {
			continue
		}
		infos = append(infos, info)
		garde = append(garde, c)
	}
	return garde
}

// Arreter interrompt la recherche ; rien n'est gardé d'une recherche arrêtée
// avant la fin, la vérification n'étant pas complète.
func (d *Doublons) Arreter() { d.arreter() }

// Attendre bloque jusqu'à la fin.
func (d *Doublons) Attendre() { <-d.fini }

// Suivi renvoie l'avancement, et le bilan une fois fini.
func (d *Doublons) Suivi() SuiviDoublons {
	d.mu.Lock()
	su := SuiviDoublons{Etat: d.etat, Phase: d.phase, DureeMs: d.duree.Milliseconds()}
	if d.etat == EnCours {
		su.DureeMs = time.Since(d.debut).Milliseconds()
	}
	for _, g := range d.groupes {
		su.Groupes++
		su.Copies += len(g) - 1
		su.Recuperable += g[0].taille * int64(len(g)-1)
	}
	d.mu.Unlock()
	su.Dossiers, su.Lus, su.Candidats = d.dossiers, d.lus.Load(), d.candidats.Load()
	su.AHacher, su.Haches, su.Illisibles = d.aHacher.Load(), d.haches.Load(), d.illisibles.Load()
	return su
}

// Groupes renvoie les groupes de doublons, les plus rentables d'abord. Ce
// que le ménage a retiré depuis n'y figure plus, et un groupe réduit à un
// seul exemplaire disparaît.
func (d *Doublons) Groupes() ([]GroupeDoublons, error) {
	d.mu.Lock()
	groupes := d.groupes
	d.mu.Unlock()

	s := d.sc
	s.arbre.RLock()
	defer s.arbre.RUnlock()
	chemins := map[int32]string{}
	var out []GroupeDoublons
	for _, g := range groupes {
		var fichiers []Trouve
		for _, c := range g {
			if _, err := s.noeud(c.d.id); err != nil {
				continue
			}
			if s.fichiersRetires[cleCible(Cible{Type: "fichier", ID: c.d.id, Nom: c.nom})] {
				continue
			}
			dossier, ok := chemins[c.d.id]
			if !ok {
				dossier, _ = s.chemin(c.d.id)
				chemins[c.d.id] = dossier
			}
			fichiers = append(fichiers, Trouve{
				Entree:   Entree{ID: c.d.id, Type: "fichier", Nom: c.nom, Taille: c.taille, Disque: c.disque, Modif: c.modif, Fichiers: 1},
				Dossier:  dossier,
				Longueur: longueur(c.chemin),
			})
		}
		if len(fichiers) < 2 {
			continue
		}
		slices.SortFunc(fichiers, func(a, b Trouve) int {
			if c := cmp.Compare(a.Modif, b.Modif); c != 0 {
				return c
			}
			return cmp.Compare(filepath.Join(a.Dossier, a.Nom), filepath.Join(b.Dossier, b.Nom))
		})
		out = append(out, GroupeDoublons{
			Taille: g[0].taille, Empreinte: hex.EncodeToString(g[0].complet[:]),
			Recuperable: g[0].taille * int64(len(fichiers)-1), Fichiers: fichiers,
		})
	}
	slices.SortStableFunc(out, func(a, b GroupeDoublons) int {
		if c := cmp.Compare(b.Recuperable, a.Recuperable); c != 0 {
			return c
		}
		return cmp.Compare(a.Empreinte, b.Empreinte)
	})
	if len(out) > MaxGroupes {
		out = out[:MaxGroupes]
	}
	return out, nil
}
