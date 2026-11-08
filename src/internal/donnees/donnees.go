// Package donnees choisit le dossier d'Aria Scope : %USERPROFILE%\.Aria-Scope,
// comme les autres applis Aria. Réglages, journal et profil WebView2 de
// l'interface y sont rangés ; l'exe reste portable.
package donnees

import (
	"os"
	"path/filepath"
)

// Nom est le dossier d'Aria Scope dans le profil de l'utilisateur.
const Nom = ".Aria-Scope"

// ancienNom est le dossier des versions jusqu'à la v0.6, dans %APPDATA%.
const ancienNom = "AriaScope"

// Dossier renvoie %USERPROFILE%\.Aria-Scope, créé si besoin. Au premier
// lancement, l'ancien %APPDATA%\AriaScope y est renommé (deplace). Si le
// renommage échoue, l'ancien dossier sert encore pour cette fois.
func Dossier() (dir string, deplace bool, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false, err
	}
	dir = filepath.Join(home, Nom)
	if conf, err := os.UserConfigDir(); err == nil {
		ancien := filepath.Join(conf, ancienNom)
		deplace, err = deplacer(ancien, dir)
		if err != nil {
			return ancien, false, err
		}
	}
	return dir, deplace, os.MkdirAll(dir, 0o755)
}

// deplacer renomme ancien en dir si ancien est un vrai dossier et que dir
// n'existe pas encore ; si les deux existent, dir est gardé tel quel.
func deplacer(ancien, dir string) (bool, error) {
	st, err := os.Lstat(ancien)
	if err != nil || !st.IsDir() {
		return false, nil
	}
	if _, err := os.Lstat(dir); err == nil {
		return false, nil
	}
	if err := os.Rename(ancien, dir); err != nil {
		return false, err
	}
	return true, nil
}

// Webview est le profil WebView2 de l'interface, rangé avec le reste ("" si
// le dossier est inconnu : Wails garde alors son emplacement par défaut).
func Webview(dir string) string {
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "webview")
}

// NettoyerAncienWebview supprime le profil WebView2 des versions précédentes,
// que Wails rangeait par défaut dans %APPDATA%\<nom de l'exe>. Il n'est effacé
// que s'il a bien la forme d'un profil WebView2 (sous-dossier EBWebView) et
// rien d'autre, pour ne jamais toucher à un dossier qui ne serait pas le sien.
func NettoyerAncienWebview(exe string) {
	conf, err := os.UserConfigDir()
	if err != nil || exe == "" {
		return
	}
	ancien := filepath.Join(conf, filepath.Base(exe))
	entrees, err := os.ReadDir(ancien)
	if err != nil || len(entrees) != 1 || entrees[0].Name() != "EBWebView" || !entrees[0].IsDir() {
		return
	}
	_ = os.RemoveAll(ancien)
}
