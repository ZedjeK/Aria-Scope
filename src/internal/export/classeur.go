package export

import (
	"fmt"
	"io"
	"time"

	"aria-scope/internal/scan"
)

// ClasseurXLSX écrit un classeur Excel sur le dossier o.Dans de l'analyse sc :
// un onglet de résumé, puis un par vue (dossiers jusqu'à o.Profondeur niveaux,
// plus gros fichiers, types, âge), et la recherche et les doublons s'ils ont
// été faits. Les colonnes sont typées (nombres, dates, pourcentages),
// filtrables, et l'en-tête reste visible.
func ClasseurXLSX(w io.Writer, sc *scan.Scan, r *scan.Recherche, d *scan.Doublons, o Options, version string, maintenant time.Time) error {
	mode := o.Mode
	c := nouveauClasseur(w)
	tailleLib := "Taille (Mo)"
	if mode == scan.ModeDisque {
		tailleLib = "Sur disque (Mo)"
	}

	// Les dossiers d'abord : on en tire aussi le résumé.
	var racine scan.LigneArbre
	var lignes int
	dossiers := func(ong *onglet) error {
		return sc.ParcourirArbre(o.Dans, o.Profondeur, mode, func(l scan.LigneArbre) error {
			if lignes == 0 {
				racine = l
			}
			lignes++
			ong.ajouter(ent(int64(l.Niveau)), txt(l.Nom), mo(poids(l.Taille, l.Disque, mode)), part100(l.Part), ent(l.Fichiers), ent(l.Dossiers),
				dateExcel(l.Modif), ent(l.Taille), ent(l.Disque), ent(l.Erreurs), txt(l.Chemin))
			return ong.err
		})
	}

	// Résumé (écrit en premier pour être le premier onglet, rempli avec la
	// racine lue juste avant)
	if err := sc.ParcourirArbre(o.Dans, 0, mode, func(l scan.LigneArbre) error {
		racine = l
		return errFin
	}); err != nil && err != errFin {
		return err
	}
	res, err := c.feuille("Résumé", 26, 22, 30)
	if err != nil {
		return err
	}
	res.ajouter(avecStyle(txt("Aria Scope · "+racine.Chemin), stTitre))
	p := sc.Progression()
	cle := func(s string) cellule { return avecStyle(txt(s), stCle) }
	res.ajouter(cle("Dossier"), txt(racine.Chemin))
	if o.Dans != 0 {
		res.ajouter(cle("Dans l'analyse de"), txt(sc.Racine()))
	}
	res.ajouter(cle("Classeur créé le"), dateExcel(maintenant.Unix()))
	res.ajouter(cle("Tailles"), txt(map[bool]string{true: "sur disque", false: "réelles"}[mode == scan.ModeDisque]))
	res.ajouter(cle("Taille réelle"), ent(racine.Taille), txt(Lisible(racine.Taille)))
	res.ajouter(cle("Sur disque"), ent(racine.Disque), txt(Lisible(racine.Disque)))
	res.ajouter(cle("Fichiers"), ent(racine.Fichiers))
	res.ajouter(cle("Dossiers"), ent(racine.Dossiers))
	res.ajouter(cle("Illisibles"), ent(racine.Erreurs))
	res.ajouter(cle("Durée de l'analyse (s)"), cellule{nombre: float64(p.DureeMs) / 1000, estNb: true, style: stMo})
	if f := sc.Filtres(); f.Actifs() {
		res.ajouter(cle("Filtres"), txt(decrireFiltres(f)))
		res.ajouter(cle("Fichiers écartés"), ent(p.Ecartes))
	}
	res.ajouter(cle("Généré par"), txt("Aria Scope "+version))

	// Dossiers
	niveaux := "tous les niveaux"
	if o.Profondeur > 0 {
		niveaux = fmt.Sprintf("%d niveau%s", o.Profondeur, map[bool]string{true: "x", false: ""}[o.Profondeur > 1])
	}
	ong, err := c.onglet("Dossiers", 7, 30, 12, 14, 10, 13, 16, 15, 17, 9, 70)
	if err != nil {
		return err
	}
	ong.ajouter(entete("Niveau"), entete("Nom"), entete(tailleLib), entete("Part du parent"), entete("Fichiers"), entete("Sous-dossiers"),
		entete("Modifié"), entete("Taille (octets)"), entete("Sur disque (octets)"), entete("Illisibles"), entete("Chemin"))
	if err := dossiers(ong); err != nil {
		return err
	}
	res.ajouter(cle("Onglet Dossiers"), txt(niveaux))
	if ong.tronque {
		res.ajouter(cle("Attention"), txt("trop de dossiers pour Excel : l'onglet Dossiers s'arrête à 1 048 575 lignes"))
	}

	// Plus gros fichiers
	if gros, err := sc.GrosFichiers(mode, scan.Tri{}); err == nil {
		ong, err := c.onglet("Gros fichiers", 7, 36, 12, 16, 15, 17, 70)
		if err != nil {
			return err
		}
		ong.ajouter(entete("Rang"), entete("Nom"), entete(tailleLib), entete("Modifié"), entete("Taille (octets)"), entete("Sur disque (octets)"), entete("Dossier"))
		rang := 0
		for _, g := range gros {
			if o.Dans != 0 && !dedans(g.Dossier, racine.Chemin) {
				continue
			}
			rang++
			ong.ajouter(ent(int64(rang)), txt(g.Nom), mo(poids(g.Taille, g.Disque, mode)), dateExcel(g.Modif), ent(g.Taille), ent(g.Disque), txt(g.Dossier))
		}
	}

	// Types
	if fams, err := sc.Types(o.Dans, mode); err == nil {
		ong, err := c.onglet("Types", 26, 18, 11, 12, 15, 15, 9)
		if err != nil {
			return err
		}
		ong.ajouter(entete("Famille"), entete("Extension"), entete("Fichiers"), entete(tailleLib), entete("Taille (octets)"), entete("Sur disque (octets)"), entete("Part"))
		var total int64
		for _, f := range fams {
			total += poids(f.Taille, f.Disque, mode)
		}
		for _, f := range fams {
			for _, e := range f.Extensions {
				ext := "." + e.Ext
				if e.Ext == "" {
					ext = "(sans extension)"
				}
				pe := poids(e.Taille, e.Disque, mode)
				ong.ajouter(txt(f.Nom), txt(ext), ent(e.Fichiers), mo(pe), ent(e.Taille), ent(e.Disque), part100(part(pe, total)))
			}
		}
	}

	// Âge
	if ages, err := sc.Ages(o.Dans); err == nil {
		ong, err := c.onglet("Âge", 26, 11, 12, 15, 15, 9)
		if err != nil {
			return err
		}
		ong.ajouter(entete("Dernière modification"), entete("Fichiers"), entete(tailleLib), entete("Taille (octets)"), entete("Sur disque (octets)"), entete("Part"))
		var total int64
		for _, a := range ages {
			total += poids(a.Taille, a.Disque, mode)
		}
		for _, a := range ages {
			pa := poids(a.Taille, a.Disque, mode)
			ong.ajouter(txt(a.Nom), ent(a.Fichiers), mo(pa), ent(a.Taille), ent(a.Disque), part100(part(pa, total)))
		}
	}

	// Recherche, si elle a été faite
	if r != nil && r.Suivi().Etat != scan.EnCours {
		if l, err := r.Resultats(mode, scan.Tri{}); err == nil {
			ong, err := c.onglet("Recherche", 9, 36, 12, 16, 15, 10, 70)
			if err != nil {
				return err
			}
			ong.ajouter(entete("Type"), entete("Nom"), entete(tailleLib), entete("Modifié"), entete("Taille (octets)"), entete("Longueur du chemin"), entete("Dossier"))
			for _, x := range l {
				ong.ajouter(txt(x.Type), txt(x.Nom), mo(poids(x.Taille, x.Disque, mode)), dateExcel(x.Modif), ent(x.Taille), ent(int64(x.Longueur)), txt(x.Dossier))
			}
		}
	}

	// Doublons, s'ils ont été cherchés
	if d != nil && d.Suivi().Etat == scan.Fini {
		if groupes, err := d.Groupes(); err == nil {
			ong, err := c.onglet("Doublons", 8, 36, 8, 14, 15, 16, 70, 66)
			if err != nil {
				return err
			}
			ong.ajouter(entete("Groupe"), entete("Nom"), entete("Copies"), entete("Exemplaire (Mo)"), entete("Récupérable (Mo)"), entete("Modifié"), entete("Dossier"), entete("SHA-256"))
			for i, g := range groupes {
				for _, f := range g.Fichiers {
					ong.ajouter(ent(int64(i+1)), txt(f.Nom), ent(int64(len(g.Fichiers))), mo(g.Taille), mo(g.Recuperable), dateExcel(f.Modif), txt(f.Dossier), txt(g.Empreinte))
				}
			}
		}
	}

	return c.fermer("Aria Scope · "+racine.Chemin, "Aria Scope "+version, maintenant)
}

var errFin = fmt.Errorf("fin")
