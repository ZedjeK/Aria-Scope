// Package journal garde la trace des opérations de ménage (corbeille,
// déplacement, copie, archive) d'un lancement à l'autre, pour pouvoir les
// retrouver et les annuler. Fichier : %APPDATA%\AriaScope\journal.json
// (~/.config/AriaScope sous Linux).
package journal

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Nombre d'opérations gardées ; les plus anciennes s'effacent.
const Max = 500

// Element : un élément traité par une opération.
type Element struct {
	Origine string `json:"origine"`
	// Arrivee : chemin d'arrivée (déplacement, copie) ou nom de l'élément
	// dans l'archive (archive). Vide pour la corbeille.
	Arrivee string `json:"arrivee,omitempty"`
	Type    string `json:"type"` // « dossier » ou « fichier »
	Taille  int64  `json:"taille"`
	// Recycle : archive seulement, l'original est parti à la corbeille
	// une fois l'archive vérifiée.
	Recycle bool `json:"recycle,omitempty"`
}

// Entree : une opération du journal.
type Entree struct {
	ID          string    `json:"id"`
	Date        time.Time `json:"date"`
	Operation   string    `json:"operation"`
	Destination string    `json:"destination,omitempty"`
	Archive     string    `json:"archive,omitempty"` // chemin du .zip créé
	Elements    []Element `json:"elements"`
	Taille      int64     `json:"taille"`
	// Annulee : quand l'opération a été annulée depuis Aria Scope.
	Annulee    *time.Time `json:"annulee,omitempty"`
	Annulation string     `json:"annulation,omitempty"` // ce que l'annulation a fait
}

// Journal lit et écrit le fichier du journal.
type Journal struct {
	mu      sync.Mutex
	chemin  string
	entrees []Entree // la plus récente en premier
	charge  bool
}

// Ouvrir prépare le journal dans le dossier de configuration de l'utilisateur.
func Ouvrir() (*Journal, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return Dans(filepath.Join(base, "AriaScope")), nil
}

// Dans prépare un journal dans un dossier précis (tests).
func Dans(dossier string) *Journal {
	return &Journal{chemin: filepath.Join(dossier, "journal.json")}
}

// charger lit le fichier une fois. Un fichier abîmé est mis de côté
// (journal.json.abime) plutôt qu'écrasé, et le journal repart à vide.
func (j *Journal) charger() {
	if j.charge {
		return
	}
	j.charge = true
	data, err := os.ReadFile(j.chemin)
	if err != nil {
		return
	}
	var e []Entree
	if err := json.Unmarshal(data, &e); err != nil {
		_ = os.Rename(j.chemin, j.chemin+".abime")
		return
	}
	j.entrees = e
}

func (j *Journal) enregistrer() error {
	data, err := json.MarshalIndent(j.entrees, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(j.chemin), 0o700); err != nil {
		return err
	}
	// Écriture atomique : un fichier temporaire, puis renommage.
	tmp := j.chemin + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, j.chemin)
}

// Ajouter inscrit une opération en tête du journal et renvoie son identifiant.
func (j *Journal) Ajouter(e Entree) (string, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.charger()
	if e.Date.IsZero() {
		e.Date = time.Now()
	}
	// Identifiant lisible et unique, même pour deux opérations dans la même seconde.
	e.ID = e.Date.Format("20060102-150405.000")
	for n := 2; j.existe(e.ID); n++ {
		e.ID = fmt.Sprintf("%s-%d", e.Date.Format("20060102-150405.000"), n)
	}
	if e.Elements == nil {
		e.Elements = []Element{}
	}
	j.entrees = append([]Entree{e}, j.entrees...)
	if len(j.entrees) > Max {
		j.entrees = j.entrees[:Max]
	}
	return e.ID, j.enregistrer()
}

func (j *Journal) existe(id string) bool {
	for _, e := range j.entrees {
		if e.ID == id {
			return true
		}
	}
	return false
}

// Lister renvoie les opérations, la plus récente en premier.
func (j *Journal) Lister() []Entree {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.charger()
	out := make([]Entree, len(j.entrees))
	copy(out, j.entrees)
	return out
}

// Trouver renvoie une opération par son identifiant.
func (j *Journal) Trouver(id string) (Entree, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.charger()
	for _, e := range j.entrees {
		if e.ID == id {
			return e, true
		}
	}
	return Entree{}, false
}

// Marquer note qu'une opération a été annulée, et ce que l'annulation a fait.
func (j *Journal) Marquer(id string, quand time.Time, resume string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.charger()
	for i := range j.entrees {
		if j.entrees[i].ID == id {
			j.entrees[i].Annulee = &quand
			j.entrees[i].Annulation = resume
			return j.enregistrer()
		}
	}
	return errors.New("opération introuvable dans le journal")
}

// Present indique si le fichier du journal existe (tests).
func (j *Journal) Present() bool {
	_, err := os.Stat(j.chemin)
	return !errors.Is(err, fs.ErrNotExist)
}
