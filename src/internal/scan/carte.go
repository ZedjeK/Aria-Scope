package scan

// Bloc est un rectangle de la treemap : un dossier, les fichiers posés dans
// un dossier, ou le regroupement des éléments trop petits pour être dessinés.
type Bloc struct {
	ID       int32  `json:"id"`   // dossier : son id ; « fichiers » et « reste » : l'id du parent
	Type     string `json:"type"` // "dossier", "fichiers", "reste"
	Nom      string `json:"nom"`
	Valeur   int64  `json:"valeur"` // taille selon le mode (réelle ou sur disque)
	Fichiers int64  `json:"fichiers"`
	Nombre   int    `json:"nombre,omitempty"` // « reste » : éléments regroupés
	Ouvrable bool   `json:"ouvrable"`         // dossier dans lequel on peut plonger
	Enfants  []Bloc `json:"enfants,omitempty"`
}

// Réglages de la carte : assez de détail pour s'orienter, assez peu de
// rectangles (un millier au plus) pour rester fluide.
const (
	carteProfondeur = 4     // niveaux de dossiers dessinés sous le dossier affiché
	carteSeuil      = 0.001 // en dessous de 0,1 % du dossier affiché, on regroupe
	carteParNiveau  = 60    // au plus 60 rectangles par dossier
)

// Carte renvoie les blocs de la treemap du dossier id.
func (s *Scan) Carte(id int32, mode string) (Bloc, error) {
	d, err := s.noeud(id)
	if err != nil {
		return Bloc{}, err
	}
	total := valeurDossier(d, mode)
	min := int64(float64(total) * carteSeuil)
	return bloc(d, mode, carteProfondeur, min), nil
}

func valeurDossier(d *Dossier, mode string) int64 {
	if mode == ModeDisque {
		return d.disque
	}
	return d.taille
}

func bloc(d *Dossier, mode string, profondeur int, min int64) Bloc {
	b := Bloc{
		ID: d.id, Type: "dossier", Nom: d.nom,
		Valeur:   valeurDossier(d, mode),
		Fichiers: d.fichiers,
		Ouvrable: len(d.enfants) > 0 || d.nFichiers > 0,
	}
	if profondeur == 0 || b.Valeur == 0 {
		return b
	}

	// Les candidats, triés du plus lourd au plus léger
	lignes := make([]Entree, 0, len(d.enfants)+1)
	for _, c := range d.enfants {
		lignes = append(lignes, Entree{ID: c.id, Type: "dossier", Taille: c.taille, Disque: c.disque})
	}
	if d.nFichiers > 0 {
		lignes = append(lignes, Entree{ID: d.id, Type: "fichiers", Taille: d.fTaille, Disque: d.fDisque, Fichiers: d.nFichiers})
	}
	Tri{}.trier(lignes, mode)

	parID := make(map[int32]*Dossier, len(d.enfants))
	for _, c := range d.enfants {
		parID[c.id] = c
	}
	var reste Bloc
	for i, l := range lignes {
		v := valeurEntree(l, mode)
		if v == 0 {
			break // trié : tout le reste est vide, rien à dessiner
		}
		if i >= carteParNiveau || v < min {
			reste.Valeur += v
			reste.Nombre++
			continue
		}
		if l.Type == "fichiers" {
			b.Enfants = append(b.Enfants, Bloc{ID: d.id, Type: "fichiers", Valeur: v, Fichiers: l.Fichiers})
			continue
		}
		b.Enfants = append(b.Enfants, bloc(parID[l.ID], mode, profondeur-1, min))
	}
	if reste.Nombre > 0 {
		reste.ID, reste.Type = d.id, "reste"
		b.Enfants = append(b.Enfants, reste)
	}
	return b
}

func valeurEntree(e Entree, mode string) int64 {
	if mode == ModeDisque {
		return e.Disque
	}
	return e.Taille
}

// Etape est un maillon du fil d'Ariane, de la racine analysée au dossier affiché.
type Etape struct {
	ID  int32  `json:"id"`
	Nom string `json:"nom"`
}

// Fil renvoie les dossiers de la racine jusqu'à id inclus.
func (s *Scan) Fil(id int32) ([]Etape, error) {
	d, err := s.noeud(id)
	if err != nil {
		return nil, err
	}
	var fil []Etape
	for ; d != nil; d = d.parent {
		fil = append(fil, Etape{ID: d.id, Nom: d.nom})
	}
	for i, j := 0, len(fil)-1; i < j; i, j = i+1, j-1 {
		fil[i], fil[j] = fil[j], fil[i]
	}
	return fil, nil
}
