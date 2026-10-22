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
	foDelete           = 0x0003
	fofNoConfirmation  = 0x0010 // Aria a déjà demandé confirmation
	fofAllowUndo       = 0x0040 // corbeille plutôt que suppression
	fofNoConfirmMkdir  = 0x0200
	fofWantNukeWarning = 0x4000 // Windows prévient s'il ne peut pas utiliser la corbeille
	erreurAnnulee      = 1223   // ERROR_CANCELLED
)

var procSHFileOperationW = windows.NewLazySystemDLL("shell32.dll").NewProc("SHFileOperationW")

// Corbeille envoie les chemins à la corbeille de Windows, en une seule
// opération (une seule entrée « Annuler la suppression » dans l'Explorateur).
// Windows affiche sa progression, propose de réessayer un fichier ouvert, et
// prévient avant toute suppression définitive (disque sans corbeille, élément
// trop gros). hwnd : la fenêtre d'Aria, pour que ses boîtes s'y rattachent.
// annule : l'utilisateur a interrompu tout ou partie de l'opération.
func Corbeille(chemins []string, hwnd uintptr) (annule bool, err error) {
	if len(chemins) == 0 {
		return false, nil
	}
	// Liste de chemins séparés par des zéros, terminée par un double zéro.
	var liste []uint16
	for _, c := range chemins {
		u, err := syscall.UTF16FromString(c)
		if err != nil {
			return false, fmt.Errorf("chemin invalide : %s", c)
		}
		liste = append(liste, u...) // u se termine déjà par un zéro
	}
	liste = append(liste, 0)

	op := shFileOpStruct{
		Hwnd:  hwnd,
		Func:  foDelete,
		From:  &liste[0],
		Flags: fofAllowUndo | fofNoConfirmation | fofNoConfirmMkdir | fofWantNukeWarning,
	}
	runtime.LockOSThread() // la progression de Windows vit sur ce fil
	defer runtime.UnlockOSThread()
	r, _, _ := procSHFileOperationW.Call(uintptr(unsafe.Pointer(&op)))
	runtime.KeepAlive(liste)
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
	}
	return fmt.Sprintf("Windows a refusé l'opération (code 0x%X)", code)
}
