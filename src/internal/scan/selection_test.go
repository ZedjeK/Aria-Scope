package scan

import "testing"

func TestResumer(t *testing.T) {
	// racine : a.bin 100 ; gros/ (g1 5000, g2 3000, profond/x/y/z 2000) ; petit/ p 10
	s := scanner(t, arbre(t))
	enf, _ := s.Enfants(0, ModeTaille, Tri{}, 0)
	gros, petit := enf[0], enf[2]
	sousGros, _ := s.Enfants(gros.ID, ModeTaille, Tri{}, 0)
	var profond Entree
	for _, e := range sousGros {
		if e.Nom == "profond" {
			profond = e
		}
	}

	r, err := s.Resumer([]Cible{
		{Type: "dossier", ID: gros.ID},
		{Type: "dossier", ID: profond.ID},                       // dans gros : écarté
		{Type: "fichier", ID: gros.ID, Nom: "g1", Taille: 5000}, // dans gros : écarté
		{Type: "dossier", ID: petit.ID},                         // gardé
		{Type: "fichier", ID: 0, Nom: "a.bin", Taille: 100},     // gardé
		{Type: "fichier", ID: 0, Nom: "a.bin", Taille: 100},     // doublon
		{Type: "reste", ID: 0},                                  // ignoré
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.Elements != 3 || r.Dossiers != 2 || r.Fichiers != 5 || r.Taille != 10110 {
		t.Fatalf("résumé : %+v", r)
	}
	if len(r.Cibles) != 3 || r.Filtre {
		t.Fatalf("cibles : %+v", r.Cibles)
	}

	if _, err := s.Resumer([]Cible{{Type: "dossier", ID: 999}}); err == nil {
		t.Error("id inconnu accepté")
	}
}

func TestResumerFiltre(t *testing.T) {
	s := scannerAvec(t, medias(t), Filtres{Inclure: []string{"*.mkv"}})
	r, _ := s.Resumer(nil)
	if !r.Filtre {
		t.Error("une analyse filtrée doit être signalée dans le résumé")
	}
}
