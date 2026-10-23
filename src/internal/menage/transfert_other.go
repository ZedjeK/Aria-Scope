//go:build !windows

package menage

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Hors de Windows (tests, banc d'essai, future version Linux) : déplacement
// et copie faits par Aria elle-même. Un nom déjà pris à l'arrivée n'est
// jamais écrasé : l'élément est laissé à sa place.

// Deplacer déplace les chemins dans le dossier dest.
func Deplacer(chemins []string, dest string, _ uintptr) (annule bool, err error) {
	var premiere error
	for _, c := range chemins {
		cible := filepath.Join(dest, filepath.Base(c))
		if _, err := os.Lstat(cible); err == nil {
			premiere = garder(premiere, fmt.Errorf("%s existe déjà", cible))
			continue
		}
		if err := os.Rename(c, cible); err != nil {
			// Autre disque : copie, puis suppression de l'original.
			if err := copierArbre(c, cible); err != nil {
				premiere = garder(premiere, err)
				continue
			}
			premiere = garder(premiere, os.RemoveAll(c))
		}
	}
	return false, premiere
}

// Copier copie les chemins dans le dossier dest.
func Copier(chemins []string, dest string, _ uintptr) (annule bool, err error) {
	var premiere error
	for _, c := range chemins {
		cible := filepath.Join(dest, filepath.Base(c))
		if _, err := os.Lstat(cible); err == nil {
			premiere = garder(premiere, fmt.Errorf("%s existe déjà", cible))
			continue
		}
		premiere = garder(premiere, copierArbre(c, cible))
	}
	return false, premiere
}

func garder(premiere, err error) error {
	if premiere != nil {
		return premiere
	}
	return err
}

// copierArbre copie un fichier ou un dossier entier (liens recréés tels quels).
func copierArbre(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		cible := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		switch {
		case d.Type()&fs.ModeSymlink != 0:
			lien, err := os.Readlink(p)
			if err != nil {
				return err
			}
			return os.Symlink(lien, cible)
		case d.IsDir():
			return os.MkdirAll(cible, info.Mode().Perm()|0o700)
		case d.Type().IsRegular():
			return copierFichier(p, cible, info.Mode().Perm())
		}
		return errors.New("type de fichier non pris en charge : " + p)
	})
}

func copierFichier(src, dst string, mode fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
