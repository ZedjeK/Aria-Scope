//go:build windows

package menage

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// shFileOpStruct reprend SHFILEOPSTRUCTW (absente de x/sys/windows).
type shFileOpStruct struct {
	Hwnd                 uintptr
	Func                 uint32
	From                 *uint16
	To                   *uint16
	Flags                uint16
	AnyOperationsAborted int32
	NameMappings         uintptr
	ProgressTitle        *uint16
}

const (
	foMove             = 0x0001
	foCopy             = 0x0002
	foDelete           = 0x0003
	fofNoConfirmation  = 0x0010 // Aria a déjà demandé confirmation
	fofAllowUndo       = 0x0040 // corbeille, et « Annuler » possible dans l'Explorateur
	fofNoConfirmMkdir  = 0x0200
	fofWantNukeWarning = 0x4000 // Windows prévient s'il ne peut pas utiliser la corbeille
	erreurAnnulee      = 1223   // ERROR_CANCELLED
)

var procSHFileOperationW = windows.NewLazySystemDLL("shell32.dll").NewProc("SHFileOperationW")

// listeDoubleZero : des chemins séparés par des zéros, terminés par un double zéro.
func listeDoubleZero(chemins []string) ([]uint16, error) {
	var liste []uint16
	for _, c := range chemins {
		u, err := syscall.UTF16FromString(c)
		if err != nil {
			return nil, fmt.Errorf("chemin invalide : %s", c)
		}
		liste = append(liste, u...) // u se termine déjà par un zéro
	}
	return append(liste, 0), nil
}

// operationShell confie l'opération à l'Explorateur de Windows, en une seule
// fois : sa fenêtre de progression, ses questions (fichier ouvert, nom déjà
// pris) et « Annuler » dans l'Explorateur. hwnd : la fenêtre d'Aria.
func operationShell(fonction uint32, sources []string, dest string, flags uint16, hwnd uintptr) (annule bool, err error) {
	if len(sources) == 0 {
		return false, nil
	}
	from, err := listeDoubleZero(sources)
	if err != nil {
		return false, err
	}
	op := shFileOpStruct{Hwnd: hwnd, Func: fonction, From: &from[0], Flags: flags}
	var to []uint16
	if dest != "" {
		if to, err = listeDoubleZero([]string{dest}); err != nil {
			return false, err
		}
		op.To = &to[0]
	}
	runtime.LockOSThread() // la progression de Windows vit sur ce fil
	defer runtime.UnlockOSThread()
	r, _, _ := procSHFileOperationW.Call(uintptr(unsafe.Pointer(&op)))
	runtime.KeepAlive(from)
	runtime.KeepAlive(to)
	annule = op.AnyOperationsAborted != 0
	switch {
	case r == 0:
		return annule, nil
	case r == erreurAnnulee:
		return true, nil
	default:
		return annule, errors.New(messageShell(r))
	}
}

// Corbeille envoie les chemins à la corbeille. Windows prévient avant toute
// suppression définitive (disque sans corbeille, élément trop gros).
func Corbeille(chemins []string, hwnd uintptr) (annule bool, err error) {
	return operationShell(foDelete, chemins, "", fofAllowUndo|fofNoConfirmation|fofNoConfirmMkdir|fofWantNukeWarning, hwnd)
}

// Deplacer déplace les chemins dans le dossier dest. Si un nom y est déjà
// pris, Windows demande quoi faire (remplacer, ignorer, comparer).
func Deplacer(chemins []string, dest string, hwnd uintptr) (annule bool, err error) {
	return operationShell(foMove, chemins, dest, fofAllowUndo|fofNoConfirmMkdir, hwnd)
}

// Copier copie les chemins dans le dossier dest, avec les mêmes questions.
func Copier(chemins []string, dest string, hwnd uintptr) (annule bool, err error) {
	return operationShell(foCopy, chemins, dest, fofAllowUndo|fofNoConfirmMkdir, hwnd)
}

// messageShell traduit les codes d'erreur les plus courants de SHFileOperation.
func messageShell(code uintptr) string {
	switch code {
	case 0x7C, 0x7B:
		return "chemin refusé par Windows (trop long ou caractères invalides)"
	case 0x78:
		return "accès refusé"
	case 0x20:
		return "un fichier est utilisé par un autre programme"
	case 0x2:
		return "élément introuvable"
	case 0x71, 0x72:
		return "la source et la destination sont identiques"
	case 0x73:
		return "plusieurs sources vers une destination impossible"
	case 0x75:
		return "opération annulée"
	case 0x76:
		return "destination inaccessible"
	}
	return fmt.Sprintf("Windows a refusé l'opération (code 0x%X)", code)
}
