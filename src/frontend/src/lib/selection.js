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
