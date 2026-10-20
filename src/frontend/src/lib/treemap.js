// Treemap « squarifiée » (Bruls, Huizing et van Wijk, 2000) : on remplit le
// rectangle par rangées, en gardant les tuiles aussi proches du carré que
// possible, ce qui les rend lisibles et comparables.

// Pire rapport largeur/hauteur d'une rangée d'aires posée sur un côté donné.
function pire(rangee, cote) {
  let somme = 0
  let max = 0
  let min = Infinity
  for (const a of rangee) {
    somme += a
    if (a > max) max = a
    if (a < min) min = a
  }
  const c2 = cote * cote
  const s2 = somme * somme
  return Math.max((c2 * max) / s2, s2 / (c2 * min))
}

// squarifier répartit des valeurs (triées de la plus grande à la plus petite)
// dans le rectangle donné, et renvoie un rectangle par valeur, dans le même ordre.
export function squarifier(valeurs, x, y, w, h) {
  const total = valeurs.reduce((s, v) => s + v, 0)
  const out = []
  if (total <= 0 || w <= 0 || h <= 0) return valeurs.map(() => ({ x, y, w: 0, h: 0 }))
  const echelle = (w * h) / total
  const aires = valeurs.map((v) => v * echelle)

  let i = 0
  while (i < aires.length) {
    const cote = Math.min(w, h)
    const rangee = [aires[i]]
    let pireActuel = pire(rangee, cote)
    let j = i + 1
    while (j < aires.length && aires[j] > 0) {
      rangee.push(aires[j])
      const p = pire(rangee, cote)
      if (p > pireActuel) {
        rangee.pop()
        break
      }
      pireActuel = p
      j++
    }
    const somme = rangee.reduce((s, a) => s + a, 0)
    if (w >= h) {
      // colonne collée à gauche
      const largeur = somme / h
      let yy = y
      for (const a of rangee) {
        const hh = a / largeur
        out.push({ x, y: yy, w: largeur, h: hh })
        yy += hh
      }
      x += largeur
      w -= largeur
    } else {
      // rangée collée en haut
      const hauteur = somme / w
      let xx = x
      for (const a of rangee) {
        const ww = a / hauteur
        out.push({ x: xx, y, w: ww, h: hauteur })
        xx += ww
      }
      y += hauteur
      h -= hauteur
    }
    i = j
    // Les valeurs nulles restantes n'ont pas de surface.
    while (i < aires.length && aires[i] <= 0) {
      out.push({ x, y, w: 0, h: 0 })
      i++
    }
  }
  return out
}

// Teintes néon des grandes branches : chaque dossier du premier niveau garde
// sa couleur dans toute sa descendance, pour qu'on s'y retrouve d'un coup d'œil.
const TEINTES = ['#7df9ff', '#5c8bff', '#b18cff', '#3fe0c5', '#ff6fae', '#59c3ff', '#b8f35a', '#a9b8ff']
const GRIS_FICHIERS = '#93a8c8'
const GRIS_RESTE = '#3b5378'

const ENTETE = 15 // bandeau du nom d'un dossier dont on dessine l'intérieur
const MARGE = 2

// disposer calcule tous les rectangles à dessiner pour le bloc racine, qui
// occupe tout le cadre (lui-même n'est pas dessiné).
export function disposer(racine, largeur, hauteur) {
  const tuiles = []
  const poser = (bloc, x, y, w, h, prof, teinte) => {
    const enfants = (bloc.enfants ?? []).filter((e) => e.valeur > 0)
    const rects = squarifier(enfants.map((e) => e.valeur), x, y, w, h)
    enfants.forEach((e, i) => {
      const r = rects[i]
      if (r.w < 1 || r.h < 1) return
      const t = prof === 1 ? TEINTES[i % TEINTES.length] : teinte
      const couleur = e.type === 'fichiers' ? GRIS_FICHIERS : e.type === 'reste' ? GRIS_RESTE : t
      const interieur = (e.enfants?.length ?? 0) > 0 && r.w > 34 && r.h > 34
      tuiles.push({ bloc: e, ...r, prof, couleur, interieur })
      if (interieur) {
        poser(e, r.x + MARGE, r.y + ENTETE, r.w - 2 * MARGE, r.h - ENTETE - MARGE, prof + 1, t)
      }
    })
  }
  poser(racine, 0, 0, largeur, hauteur, 1, TEINTES[0])
  return tuiles
}
