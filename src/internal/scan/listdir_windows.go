//go:build windows

package scan

import (
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// fileFullDirInfo reprend FILE_FULL_DIR_INFO (absente de x/sys/windows).
// Une seule lecture du dossier donne la taille réelle, la taille allouée et
// les attributs de chaque élément, sans ouvrir les fichiers un par un.
type fileFullDirInfo struct {
	NextEntryOffset uint32
	FileIndex       uint32
	CreationTime    int64
	LastAccessTime  int64
	LastWriteTime   int64
	ChangeTime      int64
	EndOfFile       int64
	AllocationSize  int64
	FileAttributes  uint32
	FileNameLength  uint32
	EaSize          uint32 // contient le tag de reparse si FILE_ATTRIBUTE_REPARSE_POINT
	FileName        [1]uint16
}

// Tampon de 64 Kio, aligné sur 8 octets comme l'exige la structure.
var tampons = sync.Pool{New: func() any {
	b := make([]uint64, 8192)
	return &b
}}

func lister(chemin string, fn func(e entree)) error {
	p, err := windows.UTF16PtrFromString(cheminLong(chemin))
	if err != nil {
		return err
	}
	h, err := windows.CreateFile(p,
		windows.FILE_LIST_DIRECTORY,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)

	tb := tampons.Get().(*[]uint64)
	defer tampons.Put(tb)
	base := unsafe.Pointer(&(*tb)[0])
	taille := uint32(len(*tb) * 8)

	classe := uint32(windows.FileFullDirectoryRestartInfo)
	for {
		err := windows.GetFileInformationByHandleEx(h, classe, (*byte)(base), taille)
		if err == windows.ERROR_NO_MORE_FILES {
			return nil
		}
		// Racine de volume vide : le pilote peut répondre « fichier introuvable ».
		if err == windows.ERROR_FILE_NOT_FOUND && classe == windows.FileFullDirectoryRestartInfo {
			return nil
		}
		if err != nil {
			return err
		}
		classe = windows.FileFullDirectoryInfo

		for off := uintptr(0); ; {
			info := (*fileFullDirInfo)(unsafe.Add(base, off))
			nom := unsafe.Slice(&info.FileName[0], info.FileNameLength/2)
			if !estPointOuPointPoint(nom) {
				attr := info.FileAttributes
				e := entree{
					Nom:     windows.UTF16ToString(nom),
					Dossier: attr&windows.FILE_ATTRIBUTE_DIRECTORY != 0,
					Taille:  info.EndOfFile,
					Disque:  info.AllocationSize,
					Modif:   filetimeUnix(info.LastWriteTime),
				}
				// Jonctions et liens symboliques : ne pas les suivre, sinon on
				// compte deux fois (ou on tourne en rond). Les autres points de
				// reparse (OneDrive, déduplication…) sont de vrais contenus.
				if attr&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
					switch info.EaSize {
					case windows.IO_REPARSE_TAG_MOUNT_POINT, windows.IO_REPARSE_TAG_SYMLINK:
						e.Lien = true
					}
				}
				fn(e)
			}
			if info.NextEntryOffset == 0 {
				break
			}
			off += uintptr(info.NextEntryOffset)
		}
	}
}

// filetimeUnix convertit un FILETIME (centaines de nanosecondes depuis 1601)
// en secondes Unix.
func filetimeUnix(ft int64) int64 {
	const ecart1601a1970 = 116444736000000000
	if ft <= ecart1601a1970 {
		return 0
	}
	return (ft - ecart1601a1970) / 10000000
}

func estPointOuPointPoint(n []uint16) bool {
	return (len(n) == 1 && n[0] == '.') || (len(n) == 2 && n[0] == '.' && n[1] == '.')
}

// cheminLong ajoute le préfixe \\?\ au-delà de la limite historique de
// 260 caractères, pour que les arborescences profondes soient lues aussi.
func cheminLong(p string) string {
	if len(p) < 240 || strings.HasPrefix(p, `\\?\`) {
		return p
	}
	if strings.HasPrefix(p, `\\`) {
		return `\\?\UNC\` + p[2:]
	}
	return `\\?\` + p
}
