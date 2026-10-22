// Sélection partagée entre l'arbre et la carte.
//   { type: "dossier" | "fichiers" | "fichier", id, nom, origine: "arbre" | "carte" }
// Pour « fichiers » et « fichier », id est celui du dossier qui les contient.

export function selectionDe(e, origine) {
  if (!e || e.type === 'reste') return null
  return { type: e.type, id: e.id, nom: e.nom ?? '', origine }
}

// Ligne de l'arbre correspondant exactement à la sélection.
export function estLigneChoisie(sel, e) {
  if (!sel || e.type !== sel.type || e.id !== sel.id) return false
  return sel.type !== 'fichier' || e.nom === sel.nom
}

// Bloc de la carte qui représente la sélection : un fichier seul n'a pas de
// bloc, on montre celui des fichiers de son dossier.
export function estBlocChoisi(sel, b) {
  if (!sel) return false
  if (sel.type === 'dossier') return b.type === 'dossier' && b.id === sel.id
  return b.type === 'fichiers' && b.id === sel.id
}

// ---------- Sélection multiple (marques) ----------
// Les éléments marqués sont ceux sur lesquels porteront les actions de ménage :
// des dossiers et des fichiers, jamais la racine de l'analyse.

export const marquable = (e) => !!e && ((e.type === 'dossier' && e.id !== 0) || e.type === 'fichier')

export const cleMarque = (c) => (c.type === 'fichier' ? `f:${c.id}/${c.nom}` : `d:${c.id}`)

export function cibleDe(e) {
  return e.type === 'fichier'
    ? { type: 'fichier', id: e.id, nom: e.nom, taille: e.taille ?? 0, disque: e.disque ?? 0 }
    : { type: 'dossier', id: e.id, nom: e.nom ?? '', taille: 0, disque: 0 }
}
