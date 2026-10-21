// Package reglages garde les préférences d'Aria Scope d'un lancement à
// l'autre, dans %APPDATA%\AriaScope\reglages.json (~/.config/AriaScope sous Linux).
package reglages

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// Reglages : les préférences de l'utilisateur.
type Reglages struct {
	// AnalyseAuto : analyser tout le PC dès l'ouverture de l'appli.
	AnalyseAuto bool `json:"analyseAuto"`
}

// Defaut : les réglages d'un premier lancement.
func Defaut() Reglages { return Reglages{AnalyseAuto: true} }

// Magasin lit et écrit le fichier de réglages.
type Magasin struct {
	mu     sync.Mutex
	chemin string
}

// Ouvrir prépare le magasin dans le dossier de configuration de l'utilisateur.
func Ouvrir() (*Magasin, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return Dans(filepath.Join(base, "AriaScope")), nil
}

// Dans prépare un magasin dans un dossier précis (tests).
func Dans(dossier string) *Magasin {
	return &Magasin{chemin: filepath.Join(dossier, "reglages.json")}
}

// Lire renvoie les réglages ; un fichier absent ou illisible donne les
// réglages par défaut (une clé absente garde sa valeur par défaut).
func (m *Magasin) Lire() Reglages {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := Defaut()
	if data, err := os.ReadFile(m.chemin); err == nil {
		_ = json.Unmarshal(data, &r)
	}
	return r
}

// Modifier applique fn aux réglages et les enregistre.
func (m *Magasin) Modifier(fn func(*Reglages)) (Reglages, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := Defaut()
	if data, err := os.ReadFile(m.chemin); err == nil {
		_ = json.Unmarshal(data, &r)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return r, err
	}
	fn(&r)
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return r, err
	}
	if err := os.MkdirAll(filepath.Dir(m.chemin), 0o700); err != nil {
		return r, err
	}
	// Écriture atomique : un fichier temporaire, puis renommage.
	tmp := m.chemin + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return r, err
	}
	return r, os.Rename(tmp, m.chemin)
}
