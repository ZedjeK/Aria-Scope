package export

import (
	"archive/zip"
	"bufio"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Un écrivain de classeur Excel (.xlsx) minimal, sans dépendance : le format
// n'est qu'un zip de fichiers XML (Office Open XML, ECMA-376). Chaque onglet
// s'écrit ligne par ligne, sans tout garder en mémoire ; les textes sont
// « en ligne » (jamais interprétés comme des formules).

// Styles de cellule (index dans cellXfs, voir styles.xml plus bas).
const (
	stTexte  = 0
	stEntete = 1
	stEntier = 2 // 12 345
	stDate   = 3 // 2026-11-06 14:05
	stPart   = 4 // 12,3 %
	stMo     = 5 // 1 234,5 (mégaoctets)
	stTitre  = 6 // titre de l'onglet Résumé
	stCle    = 7 // libellé en gras (Résumé)
)

// MaxLignesXLSX : Excel s'arrête à 1 048 576 lignes par onglet.
const MaxLignesXLSX = 1_048_576

type classeur struct {
	z       *zip.Writer
	onglets []string
	courant *onglet
}

type onglet struct {
	c        *classeur
	tableau  bool // première ligne d'en-tête : figée et filtrable
	w        *bufio.Writer
	ligne    int
	colonnes int
	largeurs []float64
	tronque  bool
	err      error
}

// cellule : une valeur et son style.
type cellule struct {
	texte  string
	nombre float64
	estNb  bool
	style  int
}

func txt(s string) cellule      { return cellule{texte: s, style: stTexte} }
func ent(n int64) cellule       { return cellule{nombre: float64(n), estNb: true, style: stEntier} }
func part100(p float64) cellule { return cellule{nombre: p, estNb: true, style: stPart} }
func mo(octets int64) cellule {
	return cellule{nombre: float64(octets) / (1 << 20), estNb: true, style: stMo}
}
func entete(s string) cellule             { return cellule{texte: s, style: stEntete} }
func avecStyle(c cellule, st int) cellule { c.style = st; return c }

// dateExcel : le numéro de série d'Excel (jours depuis le 30/12/1899), à
// l'heure locale ; vide si la date est inconnue.
func dateExcel(sec int64) cellule {
	if sec <= 0 {
		return txt("")
	}
	t := time.Unix(sec, 0)
	_, decalage := t.Zone()
	return cellule{nombre: float64(sec+int64(decalage))/86400 + 25569, estNb: true, style: stDate}
}

func nouveauClasseur(w io.Writer) *classeur {
	return &classeur{z: zip.NewWriter(w)}
}

// onglet commence un nouvel onglet en tableau (en-tête figé, filtre
// automatique) ; les largeurs sont en caractères.
func (c *classeur) onglet(nom string, largeurs ...float64) (*onglet, error) {
	return c.nouvelOnglet(nom, true, largeurs)
}

// feuille commence un onglet libre, sans en-tête de tableau (le résumé).
func (c *classeur) feuille(nom string, largeurs ...float64) (*onglet, error) {
	return c.nouvelOnglet(nom, false, largeurs)
}

func (c *classeur) nouvelOnglet(nom string, tableau bool, largeurs []float64) (*onglet, error) {
	if err := c.fermerOnglet(); err != nil {
		return nil, err
	}
	c.onglets = append(c.onglets, nomOnglet(nom))
	f, err := c.z.Create(fmt.Sprintf("xl/worksheets/sheet%d.xml", len(c.onglets)))
	if err != nil {
		return nil, err
	}
	o := &onglet{c: c, w: bufio.NewWriterSize(f, 1<<16), largeurs: largeurs, tableau: tableau}
	o.ecrire(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n" +
		`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">`)
	// La première ligne reste visible quand on fait défiler.
	if tableau {
		o.ecrire(`<sheetViews><sheetView workbookViewId="0"><pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews>`)
	}
	if len(largeurs) > 0 {
		o.ecrire("<cols>")
		for i, l := range largeurs {
			o.ecrire(fmt.Sprintf(`<col min="%d" max="%d" width="%.1f" customWidth="1"/>`, i+1, i+1, l))
		}
		o.ecrire("</cols>")
	}
	o.ecrire("<sheetData>")
	c.courant = o
	return o, nil
}

func (o *onglet) ecrire(s string) {
	if o.err == nil {
		_, o.err = o.w.WriteString(s)
	}
}

// ajouter écrit une ligne ; au-delà de la limite d'Excel, les lignes sont
// ignorées et l'onglet est marqué tronqué.
func (o *onglet) ajouter(cellules ...cellule) {
	if o.ligne >= MaxLignesXLSX {
		o.tronque = true
		return
	}
	o.ligne++
	o.colonnes = max(o.colonnes, len(cellules))
	var b strings.Builder
	fmt.Fprintf(&b, `<row r="%d">`, o.ligne)
	for i, c := range cellules {
		ref := colonne(i) + strconv.Itoa(o.ligne)
		switch {
		case c.estNb:
			fmt.Fprintf(&b, `<c r="%s" s="%d"><v>%s</v></c>`, ref, c.style, strconv.FormatFloat(c.nombre, 'g', -1, 64))
		case c.texte == "":
			if c.style != stTexte {
				fmt.Fprintf(&b, `<c r="%s" s="%d"/>`, ref, c.style)
			}
		default:
			fmt.Fprintf(&b, `<c r="%s" s="%d" t="inlineStr"><is><t xml:space="preserve">`, ref, c.style)
			xml.EscapeText(&b, []byte(texteXML(c.texte)))
			b.WriteString("</t></is></c>")
		}
	}
	b.WriteString("</row>")
	o.ecrire(b.String())
}

func (c *classeur) fermerOnglet() error {
	o := c.courant
	if o == nil {
		return nil
	}
	c.courant = nil
	o.ecrire("</sheetData>")
	// Filtre automatique sur l'en-tête, pour trier et filtrer d'un clic.
	if o.tableau && o.ligne > 1 && o.colonnes > 0 {
		o.ecrire(fmt.Sprintf(`<autoFilter ref="A1:%s%d"/>`, colonne(o.colonnes-1), o.ligne))
	}
	o.ecrire(`<pageMargins left="0.5" right="0.5" top="0.6" bottom="0.6" header="0.3" footer="0.3"/></worksheet>`)
	if o.err != nil {
		return o.err
	}
	return o.w.Flush()
}

// fermer écrit les fichiers communs et termine le zip.
func (c *classeur) fermer(titre, auteur string, quand time.Time) error {
	if err := c.fermerOnglet(); err != nil {
		return err
	}
	var types, onglets, rels strings.Builder
	for i, nom := range c.onglets {
		fmt.Fprintf(&types, `<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, i+1)
		fmt.Fprintf(&onglets, `<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, attrXML(nom), i+1, i+1)
		fmt.Fprintf(&rels, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, i+1, i+1)
	}
	n := len(c.onglets)
	date := quand.UTC().Format("2006-01-02T15:04:05Z")
	fichiers := []struct{ nom, contenu string }{
		{"[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/><Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/><Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>` + types.String() + `</Types>`},
		{"_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/></Relationships>`},
		{"docProps/core.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><dc:title>` + attrXML(titre) + `</dc:title><dc:creator>` + attrXML(auteur) + `</dc:creator><dcterms:created xsi:type="dcterms:W3CDTF">` + date + `</dcterms:created><dcterms:modified xsi:type="dcterms:W3CDTF">` + date + `</dcterms:modified></cp:coreProperties>`},
		{"docProps/app.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties"><Application>` + attrXML(auteur) + `</Application></Properties>`},
		{"xl/workbook.xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><bookViews><workbookView activeTab="0"/></bookViews><sheets>` + onglets.String() + `</sheets></workbook>`},
		{"xl/_rels/workbook.xml.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` + rels.String() + fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`, n+1) + `</Relationships>`},
		{"xl/styles.xml", stylesXML},
	}
	for _, f := range fichiers {
		w, err := c.z.Create(f.nom)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(w, f.contenu); err != nil {
			return err
		}
	}
	return c.z.Close()
}

// Formats : entier avec séparateur de milliers, date, pourcentage à une
// décimale, mégaoctets à une décimale. Excel les affiche à la française
// (espaces, virgule) selon les réglages régionaux de Windows.
const stylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
<numFmts count="3"><numFmt numFmtId="164" formatCode="yyyy-mm-dd hh:mm"/><numFmt numFmtId="165" formatCode="0.0%"/><numFmt numFmtId="166" formatCode="#,##0.0"/></numFmts>
<fonts count="3"><font><sz val="10"/><name val="Calibri"/><family val="2"/></font><font><b/><sz val="10"/><color rgb="FFFFFFFF"/><name val="Calibri"/><family val="2"/></font><font><b/><sz val="14"/><color rgb="FF06101F"/><name val="Calibri"/><family val="2"/></font></fonts>
<fills count="3"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill><fill><patternFill patternType="solid"><fgColor rgb="FF06101F"/><bgColor indexed="64"/></patternFill></fill></fills>
<borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders>
<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>
<cellXfs count="8">
<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>
<xf numFmtId="0" fontId="1" fillId="2" borderId="0" xfId="0" applyFont="1" applyFill="1"/>
<xf numFmtId="3" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="1"/>
<xf numFmtId="164" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="1"/>
<xf numFmtId="165" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="1"/>
<xf numFmtId="166" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="1"/>
<xf numFmtId="0" fontId="2" fillId="0" borderId="0" xfId="0" applyFont="1"/>
<xf numFmtId="0" fontId="1" fillId="2" borderId="0" xfId="0" applyFont="1" applyFill="1"/>
</cellXfs>
<cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles>
</styleSheet>`

// colonne : 0 → A, 25 → Z, 26 → AA.
func colonne(i int) string {
	s := ""
	for i++; i > 0; i = (i - 1) / 26 {
		s = string(rune('A'+(i-1)%26)) + s
	}
	return s
}

// nomOnglet : 31 caractères au plus, sans les caractères interdits par Excel.
func nomOnglet(nom string) string {
	nom = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`[]:*?/\`, r) {
			return '-'
		}
		return r
	}, nom)
	for utf8.RuneCountInString(nom) > 31 {
		_, n := utf8.DecodeLastRuneInString(nom)
		nom = nom[:len(nom)-n]
	}
	return nom
}

// texteXML : XML 1.0 refuse la plupart des caractères de contrôle (qu'un nom
// de fichier Linux peut contenir) ; on les remplace par « � ».
func texteXML(s string) string {
	s = strings.ToValidUTF8(s, "\uFFFD")
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\t' || r == '\n' || r == '\r':
			return r
		case r < 0x20, r == 0xFFFE, r == 0xFFFF:
			return '\uFFFD'
		}
		return r
	}, s)
}

func attrXML(s string) string {
	var b strings.Builder
	xml.EscapeText(&b, []byte(texteXML(s)))
	return strings.ReplaceAll(b.String(), `"`, "&#34;")
}
