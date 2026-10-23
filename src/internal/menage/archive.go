package menage

import (
	"archive/zip"
	"bufio"
	"compress/flate"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unicode"

	"aria-scope/internal/scan"
)

// Place comptée pour les en-têtes d'un fichier dans le zip (local, central,
// zip64, nom compris) : large, pour ne jamais manquer de place en route.
const enteteZip = 256

// Suffixe du fichier en cours d'écriture : renommé seulement une fois relu.
const suffixePart = ".part"

// Niveau de compression : mesuré sur des sources et des binaires, le niveau 1
// donne 36 % de la taille d'origine contre 32 % au niveau 6 (le défaut),
// mais va 3 fois plus vite. Sur des gigaoctets, la vitesse l'emporte.
var niveauCompression = flate.BestSpeed

// Fichiers déjà compressés : stockés tels quels, les recompresser ne fait
// que perdre du temps.
var dejaCompresses = map[string]bool{
	".zip": true, ".7z": true, ".rar": true, ".gz": true, ".tgz": true, ".bz2": true, ".xz": true, ".zst": true, ".cab": true,
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".heic": true, ".avif": true,
	".mp3": true, ".aac": true, ".ogg": true, ".opus": true, ".flac": true, ".m4a": true,
	".mp4": true, ".mkv": true, ".avi": true, ".mov": true, ".webm": true, ".wmv": true, ".m4v": true,
	".docx": true, ".xlsx": true, ".pptx": true, ".odt": true, ".ods": true, ".epub": true, ".jar": true, ".apk": true,
}

// OptionsArchive : ce que l'on choisit dans la fenêtre d'aperçu.
type OptionsArchive struct {
	Nom       string `json:"nom"`       // nom du .zip ; vide : proposé
	Corbeille bool   `json:"corbeille"` // originaux à la corbeille une fois l'archive vérifiée
}

// Les étapes d'un archivage, dans l'ordre.
const (
	PhaseEcriture     = "ecriture"
	PhaseVerification = "verification"
	PhaseCorbeille    = "corbeille"
	PhaseFini         = "fini"
)

// Suivi : où en est l'archivage, pour la barre de progression.
type Suivi struct {
	Phase         string `json:"phase"`
	Archive       string `json:"archive"`
	Fichiers      int64  `json:"fichiers"`
	FichiersFaits int64  `json:"fichiersFaits"`
	Octets        int64  `json:"octets"`
	OctetsFaits   int64  `json:"octetsFaits"`
	Courant       string `json:"courant"`
	Bilan         *Bilan `json:"bilan"` // présent une fois fini
}

// Archivage : un archivage en cours, qui tourne en arrière-plan.
type Archivage struct {
	mu       sync.Mutex
	suivi    Suivi
	annuler  atomic.Bool
	fichiers atomic.Int64
	octets   atomic.Int64
	fini     chan struct{}
}

var errAnnule = errors.New("archivage annulé")

// Suivi renvoie l'état de l'archivage.
func (a *Archivage) Suivi() Suivi {
	a.mu.Lock()
	defer a.mu.Unlock()
	s := a.suivi
	s.FichiersFaits = a.fichiers.Load()
	s.OctetsFaits = a.octets.Load()
	return s
}

// Annuler arrête l'écriture ou la vérification ; rien n'est laissé sur le
// disque. Sans effet une fois les originaux en route vers la corbeille.
func (a *Archivage) Annuler() { a.annuler.Store(true) }

// Fini se ferme quand l'archivage est terminé.
func (a *Archivage) Fini() <-chan struct{} { return a.fini }

// Attendre bloque jusqu'à la fin et renvoie le bilan.
func (a *Archivage) Attendre() Bilan {
	<-a.fini
	return *a.Suivi().Bilan
}

// pointTest : appelé à chaque changement d'étape (tests uniquement).
var pointTest = func(phase string) {}

func (a *Archivage) etape(phase, courant string) {
	pointTest(phase)
	a.mu.Lock()
	a.suivi.Phase, a.suivi.Courant = phase, courant
	a.mu.Unlock()
	a.fichiers.Store(0)
	a.octets.Store(0)
}

func (a *Archivage) courant(c string) {
	a.mu.Lock()
	a.suivi.Courant = c
	a.mu.Unlock()
}

// ApercevoirArchive récapitule l'archive à créer, sans rien toucher. nom :
// celui que l'on a tapé, vide pour en proposer un.
func ApercevoirArchive(sc *scan.Scan, cibles []scan.Cible, dest, nom string) (Apercu, error) {
	p, err := preparerArchive(sc, cibles, dest, nom)
	return p.apercu, err
}

func preparerArchive(sc *scan.Scan, cibles []scan.Cible, dest, nom string) (prepa, error) {
	p, err := preparer(sc, OpArchiver, cibles, dest)
	if err != nil {
		return p, err
	}
	if strings.TrimSpace(nom) == "" {
		nom = nomPropose(p.chemins, time.Now())
	} else if nom, err = nomArchive(nom); err != nil {
		return p, err
	}
	p.apercu.Archive = nomLibre(p.apercu.Destination, nom)
	return p, nil
}

// nomPropose : le nom de l'élément s'il est seul, sinon une archive datée.
func nomPropose(chemins []string, maintenant time.Time) string {
	if len(chemins) == 1 {
		base := filepath.Base(chemins[0])
		if info, err := os.Stat(chemins[0]); err == nil && !info.IsDir() {
			if sans := strings.TrimSuffix(base, filepath.Ext(base)); sans != "" {
				base = sans
			}
		}
		if n, err := nomArchive(base); err == nil {
			return n
		}
	}
	return "Archive " + maintenant.Format("2006-01-02") + ".zip"
}

var nomsReserves = map[string]bool{"CON": true, "PRN": true, "AUX": true, "NUL": true}

func init() {
	for i := '1'; i <= '9'; i++ {
		nomsReserves["COM"+string(i)] = true
		nomsReserves["LPT"+string(i)] = true
	}
}

// nomArchive contrôle un nom tapé à la main et lui ajoute « .zip » au besoin.
// Les règles sont celles de Windows, partout : l'archive doit pouvoir voyager.
func nomArchive(nom string) (string, error) {
	nom = strings.TrimSpace(nom)
	if !strings.HasSuffix(strings.ToLower(nom), ".zip") {
		nom += ".zip"
	}
	base := strings.TrimRight(strings.TrimSuffix(nom, nom[len(nom)-4:]), ". ")
	if base == "" {
		return "", errors.New("donne un nom à l'archive")
	}
	for _, r := range base {
		if strings.ContainsRune(`<>:"/\|?*`, r) || unicode.IsControl(r) {
			return "", fmt.Errorf("le caractère %q n'est pas permis dans un nom de fichier", r)
		}
	}
	if racine, _, _ := strings.Cut(base, "."); nomsReserves[strings.ToUpper(strings.TrimSpace(racine))] {
		return "", fmt.Errorf("« %s » est un nom réservé par Windows", base)
	}
	if len(base) > 200 {
		return "", errors.New("nom d'archive trop long")
	}
	return base + ".zip", nil
}

// nomLibre : nom.zip, ou nom (2).zip, (3)… si déjà pris. Jamais d'écrasement.
func nomLibre(dest, nom string) string {
	base := strings.TrimSuffix(nom, filepath.Ext(nom))
	for i := 1; ; i++ {
		c := nom
		if i > 1 {
			c = fmt.Sprintf("%s (%d).zip", base, i)
		}
		if !existe(filepath.Join(dest, c)) && !existe(filepath.Join(dest, c+suffixePart)) {
			return c
		}
	}
}

func existe(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

// Archiver lance l'archivage en arrière-plan et rend la main aussitôt.
// L'archive est écrite sous un nom provisoire, relue en entier, puis
// renommée ; si on le demande, les originaux partent ensuite à la corbeille,
// sauf ceux qui sont protégés, ont changé pendant l'archivage ou n'ont pas
// pu être lus en entier.
func Archiver(sc *scan.Scan, cibles []scan.Cible, dest string, opts OptionsArchive, hwnd uintptr) (*Archivage, error) {
	if !opMu.TryLock() {
		return nil, errors.New("une opération de ménage est déjà en cours")
	}
	p, err := preparerArchive(sc, cibles, dest, opts.Nom)
	if err == nil && p.apercu.Manque > 0 {
		err = fmt.Errorf("pas assez de place à l'arrivée : il manque %d octets", p.apercu.Manque)
	}
	if err == nil && len(p.cibles) == 0 {
		err = errors.New("rien dans cette sélection ne peut être archivé")
	}
	if err != nil {
		opMu.Unlock()
		return nil, err
	}
	a := &Archivage{fini: make(chan struct{})}
	a.suivi = Suivi{
		Phase: PhaseEcriture, Archive: filepath.Join(p.apercu.Destination, p.apercu.Archive),
		Fichiers: p.apercu.Fichiers, Octets: p.apercu.Taille,
	}
	go func() {
		defer opMu.Unlock()
		defer close(a.fini)
		pointTest(PhaseEcriture)
		b := a.executer(sc, p, opts, hwnd)
		a.mu.Lock()
		a.suivi.Phase, a.suivi.Courant, a.suivi.Bilan = PhaseFini, "", &b
		a.mu.Unlock()
	}()
	return a, nil
}

// fichierArchive : ce qui a été mis dans le zip, pour la vérification et
// pour savoir si l'original a changé depuis.
type fichierArchive struct {
	nom     string // dans le zip
	chemin  string // sur le disque
	element int
	dossier bool
	taille  int64 // octets lus
	modif   time.Time
}

func (a *Archivage) executer(sc *scan.Scan, p prepa, opts OptionsArchive, hwnd uintptr) Bilan {
	ap := p.apercu
	b := nouveauBilan(OpArchiver, ap)
	final := filepath.Join(ap.Destination, ap.Archive)
	part := final + suffixePart

	contenu, sautes, err := a.ecrire(part, p)
	b.Sautes = sautes
	if err == nil {
		a.etape(PhaseVerification, "")
		err = a.verifier(part, contenu)
	}
	if err == nil {
		// Un fichier du même nom a pu apparaître pendant l'écriture.
		if existe(final) {
			final = filepath.Join(ap.Destination, nomLibre(ap.Destination, ap.Archive))
		}
		err = os.Rename(part, final)
	}
	if err != nil {
		os.Remove(part)
		b.Sautes = []Refus{}
		if errors.Is(err, errAnnule) {
			b.Annule = true
		} else {
			b.Erreur = err.Error()
		}
		return b
	}

	b.Archive = final
	b.Verifie = true
	if info, err := os.Stat(final); err == nil {
		b.Compresse = info.Size()
	}
	b.Faits = len(p.cibles)
	if r, err := sc.Resumer(p.cibles); err == nil {
		b.Taille, b.Disque = r.Taille, r.Disque
	}
	b.Perime = ap.DansAnalyse

	if opts.Corbeille {
		a.etape(PhaseCorbeille, "")
		a.recycler(sc, p, contenu, sautes, hwnd, &b)
	}
	return b
}

// ecrire crée l'archive dans part. Une erreur d'écriture, ou une lecture qui
// échoue en plein fichier, annule tout ; un fichier qui ne s'ouvre pas est
// simplement sauté et signalé.
func (a *Archivage) ecrire(part string, p prepa) (contenu []fichierArchive, sautes []Refus, err error) {
	sautes = []Refus{}
	f, err := os.OpenFile(part, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return nil, sautes, fmt.Errorf("impossible de créer l'archive : %w", err)
	}
	defer func() {
		if f != nil {
			f.Close()
		}
	}()
	tampon := bufio.NewWriterSize(f, 1<<20)
	zw := zip.NewWriter(tampon)
	zw.RegisterCompressor(zip.Deflate, func(w io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(w, niveauCompression)
	})
	lecture := make([]byte, 1<<20)
	pris := map[string]bool{}

	sauter := func(chemin string, raison string) { sautes = append(sautes, Refus{Chemin: chemin, Raison: raison}) }

	ajouterFichier := func(chemin, nom string, element int) error {
		src, err := os.Open(chemin)
		if err != nil {
			sauter(chemin, raisonLecture(err))
			return nil
		}
		defer src.Close()
		info, err := src.Stat()
		if err != nil || !info.Mode().IsRegular() {
			sauter(chemin, "pas un fichier ordinaire")
			return nil
		}
		h, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		h.Name = nom
		h.Method = zip.Deflate
		if dejaCompresses[strings.ToLower(filepath.Ext(nom))] {
			h.Method = zip.Store
		}
		w, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		a.courant(chemin)
		n, err := io.CopyBuffer(w, &lecteur{r: src, a: a}, lecture)
		if err != nil {
			if errors.Is(err, errAnnule) {
				return err
			}
			// L'entrée est à moitié écrite : l'archive ne serait pas fiable.
			return fmt.Errorf("lecture interrompue de %s : %w", chemin, err)
		}
		a.fichiers.Add(1)
		contenu = append(contenu, fichierArchive{nom: nom, chemin: chemin, element: element, taille: n, modif: info.ModTime()})
		return nil
	}
	ajouterDossier := func(chemin, nom string, element int, info fs.FileInfo) error {
		h := &zip.FileHeader{Name: nom + "/", Method: zip.Store, Modified: info.ModTime()}
		h.SetMode(info.Mode())
		if _, err := zw.CreateHeader(h); err != nil {
			return err
		}
		contenu = append(contenu, fichierArchive{nom: nom + "/", chemin: chemin, element: element, dossier: true, modif: info.ModTime()})
		return nil
	}

	for i, chemin := range p.chemins {
		haut := nomUnique(pris, filepath.Base(chemin))
		info, err := os.Lstat(chemin)
		if err != nil {
			sauter(chemin, raisonLecture(err))
			continue
		}
		if !info.IsDir() {
			if !info.Mode().IsRegular() {
				sauter(chemin, "lien ou fichier spécial, non archivé")
				continue
			}
			if err := ajouterFichier(chemin, haut, i); err != nil {
				return nil, sautes, err
			}
			continue
		}
		err = filepath.WalkDir(chemin, func(c string, d fs.DirEntry, err error) error {
			if a.annuler.Load() {
				return errAnnule
			}
			if err != nil {
				sauter(c, raisonLecture(err))
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			rel, err := filepath.Rel(chemin, c)
			if err != nil {
				return err
			}
			nom := haut
			if rel != "." {
				nom = haut + "/" + filepath.ToSlash(rel)
			}
			switch {
			case d.IsDir():
				info, err := d.Info()
				if err != nil {
					sauter(c, raisonLecture(err))
					return fs.SkipDir
				}
				return ajouterDossier(c, nom, i, info)
			case d.Type().IsRegular():
				return ajouterFichier(c, nom, i)
			default:
				// Lien symbolique, jonction… : comme l'analyse, on ne les suit pas.
				sauter(c, "lien ou fichier spécial, non archivé")
				return nil
			}
		})
		if err != nil {
			return nil, sautes, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, sautes, err
	}
	if err := tampon.Flush(); err != nil {
		return nil, sautes, err
	}
	if err := f.Sync(); err != nil {
		return nil, sautes, err
	}
	err = f.Close()
	f = nil
	return contenu, sautes, err
}

// nomUnique : deux éléments du même nom venus de dossiers différents ne
// doivent pas se mélanger dans l'archive.
func nomUnique(pris map[string]bool, nom string) string {
	ext := filepath.Ext(nom)
	base := strings.TrimSuffix(nom, ext)
	for i := 1; ; i++ {
		c := nom
		if i > 1 {
			c = fmt.Sprintf("%s (%d)%s", base, i, ext)
		}
		if k := strings.ToLower(c); !pris[k] {
			pris[k] = true
			return c
		}
	}
}

// verifier relit toute l'archive : chaque fichier doit se décompresser avec
// le bon CRC et la taille exacte de ce qui a été lu sur le disque.
func (a *Archivage) verifier(part string, contenu []fichierArchive) error {
	r, err := zip.OpenReader(part)
	if err != nil && !errors.Is(err, zip.ErrInsecurePath) {
		return fmt.Errorf("vérification : archive illisible : %w", err)
	}
	defer r.Close()
	if len(r.File) != len(contenu) {
		return fmt.Errorf("vérification : %d entrées attendues, %d trouvées", len(contenu), len(r.File))
	}
	tampon := make([]byte, 1<<20)
	for i, f := range r.File {
		attendu := contenu[i]
		if f.Name != attendu.nom {
			return fmt.Errorf("vérification : entrée inattendue %s", f.Name)
		}
		if attendu.dossier {
			continue
		}
		a.courant(attendu.chemin)
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("vérification de %s : %w", f.Name, err)
		}
		n, err := io.CopyBuffer(io.Discard, &lecteur{r: rc, a: a}, tampon)
		rc.Close()
		if errors.Is(err, errAnnule) {
			return err
		}
		if err != nil {
			return fmt.Errorf("vérification de %s : %w", f.Name, err)
		}
		if n != attendu.taille {
			return fmt.Errorf("vérification de %s : %d octets au lieu de %d", f.Name, n, attendu.taille)
		}
		a.fichiers.Add(1)
	}
	return nil
}

// recycler envoie à la corbeille les originaux dont tout le contenu est dans
// l'archive vérifiée, tel qu'il est encore sur le disque.
func (a *Archivage) recycler(sc *scan.Scan, p prepa, contenu []fichierArchive, sautes []Refus, hwnd uintptr, b *Bilan) {
	raisons := map[int]string{}
	for i, r := range p.gardes {
		raisons[i] = r
	}
	for _, s := range sautes {
		for i, c := range p.chemins {
			if _, deja := raisons[i]; !deja && dedans(s.Chemin, c) {
				raisons[i] = "contient des fichiers qui n'ont pas pu être archivés"
			}
		}
	}
	for _, f := range contenu {
		if _, deja := raisons[f.element]; deja {
			continue
		}
		info, err := os.Lstat(f.chemin)
		switch {
		case err != nil:
			raisons[f.element] = "a changé pendant l'archivage"
		case f.dossier:
			// La date d'un dossier change dès qu'on y ajoute ou retire quelque chose.
			if !info.IsDir() || !info.ModTime().Equal(f.modif) {
				raisons[f.element] = "a changé pendant l'archivage"
			}
		case info.Size() != f.taille || !info.ModTime().Equal(f.modif):
			raisons[f.element] = "a changé pendant l'archivage"
		}
	}

	var cibles []scan.Cible
	for i, c := range p.cibles {
		if r, garde := raisons[i]; garde {
			b.Gardes = append(b.Gardes, Refus{Chemin: p.chemins[i], Raison: r})
		} else {
			cibles = append(cibles, c)
		}
	}
	if len(cibles) == 0 {
		return
	}
	retraits, err := sc.PreparerRetrait(cibles)
	if err != nil {
		b.Erreur = err.Error()
		return
	}
	chemins := make([]string, len(retraits))
	for i, r := range retraits {
		chemins[i] = r.Chemin
	}
	annule, err := Corbeille(chemins, hwnd)
	if err != nil {
		b.Erreur = err.Error()
	}
	partis := map[string]bool{}
	for _, f := range sc.Retirer(retraits) {
		b.Recycles++
		b.Libere += f.Taille
		b.LibereDisque += f.Disque
		partis[f.Chemin] = true
	}
	for _, r := range retraits {
		if !partis[r.Chemin] {
			raison := "Windows ne l'a pas envoyé à la corbeille"
			if annule {
				raison = "envoi à la corbeille interrompu"
			}
			b.Gardes = append(b.Gardes, Refus{Chemin: r.Chemin, Raison: raison})
		}
	}
}

// lecteur compte les octets pour la progression et s'arrête si on annule.
type lecteur struct {
	r io.Reader
	a *Archivage
}

func (l *lecteur) Read(p []byte) (int, error) {
	if l.a.annuler.Load() {
		return 0, errAnnule
	}
	n, err := l.r.Read(p)
	l.a.octets.Add(int64(n))
	return n, err
}

// raisonLecture traduit pourquoi un fichier n'a pas pu être lu.
func raisonLecture(err error) string {
	var errno syscall.Errno
	switch {
	case errors.Is(err, fs.ErrPermission):
		return "accès refusé"
	case errors.Is(err, fs.ErrNotExist):
		return "a disparu entre-temps"
	case runtime.GOOS == "windows" && errors.As(err, &errno) && (errno == 32 || errno == 33):
		return "ouvert et verrouillé par un autre programme"
	}
	return "illisible : " + err.Error()
}
