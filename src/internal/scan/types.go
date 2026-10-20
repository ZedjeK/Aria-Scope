package scan

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

// Familles de fichiers, d'après leur extension.
var familles = map[string][]string{
	"Vidéos":                      {"mp4", "mkv", "avi", "mov", "wmv", "flv", "webm", "m4v", "mpg", "mpeg", "ts", "m2ts", "vob", "3gp"},
	"Images":                      {"jpg", "jpeg", "png", "gif", "bmp", "tif", "tiff", "webp", "heic", "heif", "avif", "raw", "cr2", "cr3", "nef", "arw", "dng", "psd", "svg", "ico"},
	"Audio":                       {"mp3", "flac", "wav", "aac", "ogg", "opus", "m4a", "wma", "aiff", "alac", "mid", "midi"},
	"Archives":                    {"zip", "rar", "7z", "tar", "gz", "tgz", "bz2", "xz", "zst", "cab", "lz4", "lzma", "whl", "nupkg"},
	"Images disque et VM":         {"iso", "img", "vhd", "vhdx", "vmdk", "vdi", "qcow2", "wim", "esd"},
	"Documents":                   {"pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx", "odt", "ods", "odp", "rtf", "txt", "md", "epub", "csv"},
	"Code":                        {"go", "js", "mjs", "ts", "jsx", "tsx", "py", "pyc", "c", "h", "cpp", "hpp", "cs", "java", "class", "rs", "rb", "php", "html", "css", "scss", "json", "xml", "yaml", "yml", "toml", "sh", "ps1", "bat", "lua", "sql", "svelte", "vue", "kt", "swift"},
	"Programmes et bibliothèques": {"exe", "dll", "sys", "msi", "so", "dylib", "bin", "jar", "apk", "appx", "msix", "ocx", "efi", "a", "lib"},
	"Jeux":                        {"pak", "vpk", "upk", "uasset", "umap", "ubulk", "bsa", "ba2", "esm", "esp", "forge", "bnk", "wem", "big", "pck", "ucas", "utoc"},
	"3D et impression 3D":         {"stl", "obj", "3mf", "ctb", "goo", "cbddlp", "photon", "pwmx", "lys", "chitubox", "fbx", "blend", "gcode", "step", "stp"},
	"Bases de données":            {"db", "sqlite", "sqlite3", "mdb", "accdb", "ldf", "mdf", "ndf", "ost", "pst", "edb"},
	"Polices":                     {"ttf", "otf", "woff", "woff2", "fon"},
	"Temporaires et journaux":     {"tmp", "temp", "log", "dmp", "etl", "bak", "old", "cache", "crdownload", "part"},
}

const (
	familleAutre = "Autres"
	familleSans  = "Sans extension"
)

var familleDe = func() map[string]string {
	m := map[string]string{}
	for f, exts := range familles {
		for _, e := range exts {
			m[e] = f
		}
	}
	return m
}()

// extension renvoie l'extension en minuscules, sans le point ; "" pour un
// fichier qui n'en a pas (y compris « .gitignore » ou « LISEZMOI »).
func extension(nom string) string {
	e := filepath.Ext(nom)
	if len(e) < 2 || len(e) == len(nom) || len(e) > 13 || strings.ContainsAny(e, " ") {
		return ""
	}
	return strings.ToLower(e[1:])
}

type compteur struct{ taille, disque, fichiers int64 }

func (c *compteur) ajouter(e entree) {
	c.taille += e.Taille
	c.disque += e.Disque
	c.fichiers++
}

// statsTypes cumule les fichiers par extension pour toute l'analyse. Chaque
// dossier compte d'abord chez lui, puis fusionne d'un coup : un verrou par
// dossier, pas par fichier.
type statsTypes struct {
	mu   sync.Mutex
	exts map[string]compteur
}

func (st *statsTypes) fusionner(local map[string]*compteur) {
	if len(local) == 0 {
		return
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.exts == nil {
		st.exts = map[string]compteur{}
	}
	for k, v := range local {
		c := st.exts[k]
		c.taille += v.taille
		c.disque += v.disque
		c.fichiers += v.fichiers
		st.exts[k] = c
	}
}

// ---------- Résultat ----------

// TypeExt : une extension et ce qu'elle pèse.
type TypeExt struct {
	Ext      string `json:"ext"`
	Taille   int64  `json:"taille"`
	Disque   int64  `json:"disque"`
	Fichiers int64  `json:"fichiers"`
}

// Famille regroupe des extensions (Vidéos, Archives…).
type Famille struct {
	Nom        string    `json:"nom"`
	Taille     int64     `json:"taille"`
	Disque     int64     `json:"disque"`
	Fichiers   int64     `json:"fichiers"`
	Extensions []TypeExt `json:"extensions"` // les plus lourdes d'abord, 30 au plus
}

const extParFamille = 30

func repartir(exts map[string]compteur, mode string) []Famille {
	parNom := map[string]*Famille{}
	for ext, c := range exts {
		nom := familleSans
		if ext != "" {
			nom = familleDe[ext]
			if nom == "" {
				nom = familleAutre
			}
		}
		f := parNom[nom]
		if f == nil {
			f = &Famille{Nom: nom}
			parNom[nom] = f
		}
		f.Taille += c.taille
		f.Disque += c.disque
		f.Fichiers += c.fichiers
		f.Extensions = append(f.Extensions, TypeExt{Ext: ext, Taille: c.taille, Disque: c.disque, Fichiers: c.fichiers})
	}
	val := func(t, d int64) int64 {
		if mode == ModeDisque {
			return d
		}
		return t
	}
	out := make([]Famille, 0, len(parNom))
	for _, f := range parNom {
		slices.SortFunc(f.Extensions, func(a, b TypeExt) int {
			if c := cmp.Compare(val(b.Taille, b.Disque), val(a.Taille, a.Disque)); c != 0 {
				return c
			}
			return cmp.Compare(a.Ext, b.Ext)
		})
		if len(f.Extensions) > extParFamille {
			f.Extensions = f.Extensions[:extParFamille]
		}
		out = append(out, *f)
	}
	slices.SortFunc(out, func(a, b Famille) int {
		if c := cmp.Compare(val(b.Taille, b.Disque), val(a.Taille, a.Disque)); c != 0 {
			return c
		}
		return cmp.Compare(a.Nom, b.Nom)
	})
	return out
}

// Types renvoie la répartition par famille de fichiers du dossier id.
// Pour la racine, ce sont les chiffres relevés pendant le scan ; pour une
// branche, on relit ses dossiers à la demande.
func (s *Scan) Types(id int32, mode string) ([]Famille, error) {
	d, err := s.noeud(id)
	if err != nil {
		return nil, err
	}
	if id == 0 {
		s.types.mu.Lock()
		defer s.types.mu.Unlock()
		return repartir(s.types.exts, mode), nil
	}
	chemin, err := s.Chemin(id)
	if err != nil {
		return nil, err
	}
	var st statsTypes
	s.relireBranche(d, chemin, func(local map[string]*compteur) { st.fusionner(local) })
	return repartir(st.exts, mode), nil
}

// relireBranche relit en parallèle tous les dossiers sous d (connus grâce au
// scan, sans redécouvrir l'arborescence) et confie les fichiers de chacun à fn.
func (s *Scan) relireBranche(d *Dossier, chemin string, fn func(map[string]*compteur)) {
	type tache struct {
		d      *Dossier
		chemin string
	}
	taches := make(chan tache, 256)
	var wg sync.WaitGroup
	for range Parallelisme() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range taches {
				local := map[string]*compteur{}
				_ = listerDossier(t.chemin, func(e entree) {
					if e.Dossier || e.Lien {
						return
					}
					ext := extension(e.Nom)
					c := local[ext]
					if c == nil {
						c = &compteur{}
						local[ext] = c
					}
					c.ajouter(e)
				})
				fn(local)
			}
		}()
	}
	pile := []tache{{d, chemin}}
	for len(pile) > 0 {
		t := pile[len(pile)-1]
		pile = pile[:len(pile)-1]
		taches <- t
		for _, c := range t.d.enfants {
			pile = append(pile, tache{c, filepath.Join(t.chemin, c.nom)})
		}
	}
	close(taches)
	wg.Wait()
}
