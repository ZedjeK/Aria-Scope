//go:build windows

package volumes

import (
	"golang.org/x/sys/windows"
)

// Lister renvoie les lecteurs prêts (un lecteur de carte vide est ignoré).
func Lister() []Volume {
	buf := make([]uint16, 512)
	n, err := windows.GetLogicalDriveStrings(uint32(len(buf)), &buf[0])
	if err != nil || n == 0 {
		return nil
	}
	var out []Volume
	for _, racine := range decouper(buf[:n]) {
		p, err := windows.UTF16PtrFromString(racine)
		if err != nil {
			continue
		}
		var libre, total, libreTotal uint64
		if err := windows.GetDiskFreeSpaceEx(p, &libre, &total, &libreTotal); err != nil {
			continue // pas de média, lecteur réseau déconnecté…
		}
		v := Volume{Chemin: racine, Total: int64(total), Libre: int64(libreTotal)}
		switch windows.GetDriveType(p) {
		case windows.DRIVE_FIXED:
			v.Type = "fixe"
		case windows.DRIVE_REMOVABLE:
			v.Type = "amovible"
		case windows.DRIVE_REMOTE:
			v.Type = "reseau"
		case windows.DRIVE_CDROM:
			v.Type = "optique"
		case windows.DRIVE_RAMDISK:
			v.Type = "ram"
		default:
			v.Type = "autre"
		}
		nom := make([]uint16, windows.MAX_PATH+1)
		fs := make([]uint16, windows.MAX_PATH+1)
		if windows.GetVolumeInformation(p, &nom[0], uint32(len(nom)), nil, nil, nil, &fs[0], uint32(len(fs))) == nil {
			v.Nom = windows.UTF16ToString(nom)
			v.Systeme = windows.UTF16ToString(fs)
		}
		out = append(out, v)
	}
	return out
}

// decouper lit une liste de chaînes UTF-16 séparées par des zéros.
func decouper(b []uint16) []string {
	var out []string
	debut := 0
	for i, c := range b {
		if c == 0 {
			if i > debut {
				out = append(out, windows.UTF16ToString(b[debut:i]))
			}
			debut = i + 1
		}
	}
	if debut < len(b) {
		out = append(out, windows.UTF16ToString(b[debut:]))
	}
	return out
}
