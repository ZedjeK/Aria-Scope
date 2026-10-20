package scan

// entree est un élément lu dans un dossier, avec ce qu'il faut pour le compter.
type entree struct {
	Nom     string
	Dossier bool
	Taille  int64 // taille réelle (fin de fichier)
	Disque  int64 // place occupée sur le disque (clusters alloués)
	Lien    bool  // lien symbolique ou jonction : on ne le suit pas
}

// listerDossier appelle fn pour chaque élément du dossier, sans « . » ni « .. ».
// L'implémentation dépend du système (listdir_windows.go, listdir_other.go).
func listerDossier(chemin string, fn func(e entree)) error {
	return lister(chemin, fn)
}
