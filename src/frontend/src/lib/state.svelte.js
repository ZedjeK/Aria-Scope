// État partagé de l'interface et appels au backend Go (ScopeService).
import { Call } from '@wailsio/runtime'
import { filtresVides } from './filtres.js'
import { selectionDe, marquable, cibleDe, cleMarque } from './selection.js'

export const call = (method, ...args) => Call.ByName(`main.ScopeService.${method}`, ...args)
export const message = (err) => String(err?.message ?? err)

export const app = $state({
  page: 'accueil',
  etat: null, // { version, gen, scan, attendu, parallelisme }
  volumes: [],
  mode: 'taille', // "taille" (réelle) ou "disque" (place occupée)
  tri: { colonne: '', inverse: false }, // colonne vide : la taille du mode
  carte: true, // panneau des visuels affiché à côté de l'arbre
  visuel: 'carte', // dans ce panneau : "carte" (treemap) ou "anneau"
  vue: 'arbre', // colonne principale : "arbre", "gros" (plus gros fichiers), "types" ou "ages"
  portee: 'tout', // statistiques : "tout" (l'analyse) ou "selection" (le dossier choisi)
  filtres: filtresVides(), // appliqués aux prochaines analyses (voir filtres.js)
  panneauFiltres: false,
  reglages: { analyseAuto: true },
  demarrageAuto: false, // l'analyse de tout le PC va démarrer d'elle-même
  selection: null, // partagée entre l'arbre et la carte (voir selection.js)
  marques: [], // sélection multiple : cibles des actions de ménage
  ancre: null, // point de départ d'une sélection Maj+clic
  revision: 0, // augmente à chaque ménage : les vues se rechargent
  bilan: null, // résultat de la dernière opération de ménage
  erreur: '',
  loadError: '',
})

export const scanEnCours = () => app.etat?.scan?.etat === 'scan'

export async function refresh() {
  try {
    app.etat = await call('Etat')
    if (app.etat?.scan) app.demarrageAuto = false
    app.loadError = ''
  } catch (err) {
    app.loadError = message(err)
  }
}

export async function chargerVolumes() {
  try {
    app.volumes = (await call('Volumes')) ?? []
  } catch (err) {
    app.erreur = message(err)
  }
}

// Rafraîchissement rapide pendant l'analyse, lent sinon.
let minuteur = null
export function suivre() {
  if (minuteur) return
  let dernier = 0
  minuteur = setInterval(() => {
    const vite = scanEnCours()
    const maintenant = Date.now()
    if (vite || maintenant - dernier > 3000) {
      dernier = maintenant
      const etaitEnCours = vite
      refresh().then(() => {
        // Fin d'analyse : l'espace libre des disques a pu bouger.
        if (etaitEnCours && !scanEnCours()) chargerVolumes()
      })
    }
  }, 300)
}

// Lance l'analyse de chemin avec les filtres en cours ; renvoie false si le
// moteur l'a refusée (dossier introuvable, filtre invalide…).
export async function lancer(chemin) {
  app.erreur = ''
  let ok = true
  try {
    await call('Lancer', chemin, $state.snapshot(app.filtres))
    app.bilan = null
    app.selection = null
    app.marques = []
    app.page = 'analyse'
  } catch (err) {
    app.erreur = message(err)
    ok = false
  }
  await refresh()
  return ok
}

// Analyse de tous les disques internes et externes, réunis sous « Ce PC ».
export async function lancerPC() {
  app.erreur = ''
  let ok = true
  try {
    await call('LancerPC', $state.snapshot(app.filtres))
    app.bilan = null
    app.selection = null
    app.marques = []
    app.page = 'analyse'
  } catch (err) {
    app.erreur = message(err)
    ok = false
  }
  await refresh()
  return ok
}

// Au démarrage : avec l'analyse automatique, on va droit à la page d'analyse,
// le moteur lance « Ce PC » de son côté.
export async function demarrer() {
  try {
    app.reglages = await call('Reglages')
  } catch {}
  await refresh()
  if (app.etat?.scan || app.reglages.analyseAuto) app.page = 'analyse'
  if (!app.etat?.scan && app.reglages.analyseAuto) {
    app.demarrageAuto = true
    setTimeout(() => (app.demarrageAuto = false), 6000) // au cas où aucun disque ne serait lisible
  }
}

export async function setAnalyseAuto(actif) {
  try {
    app.reglages = await call('SetAnalyseAuto', actif)
  } catch (err) {
    app.erreur = message(err)
  }
}

// Clic sur un élément d'une liste (arbre, gros fichiers) : comme dans
// l'Explorateur, Ctrl ajoute ou retire, Maj étend depuis le dernier clic,
// un clic simple ne garde que lui. liste : les éléments visibles, dans l'ordre.
export function cliquerElement(e, origine, evt, liste = []) {
  app.selection = selectionDe(e, origine)
  if (!marquable(e)) {
    app.marques = []
    return
  }
  const c = cibleDe(e)
  const cle = cleMarque(c)
  if (evt?.ctrlKey || evt?.metaKey) {
    const i = app.marques.findIndex((m) => cleMarque(m) === cle)
    if (i >= 0) app.marques.splice(i, 1)
    else app.marques.push(c)
    app.ancre = c
    return
  }
  if (evt?.shiftKey && app.ancre) {
    const cibles = liste.filter(marquable).map(cibleDe)
    const a = cibles.findIndex((m) => cleMarque(m) === cleMarque(app.ancre))
    const b = cibles.findIndex((m) => cleMarque(m) === cle)
    if (a >= 0 && b >= 0) {
      app.marques = cibles.slice(Math.min(a, b), Math.max(a, b) + 1)
      return
    }
  }
  app.marques = [c]
  app.ancre = c
}

// Choix sur la carte ou l'anneau : la sélection multiple se réduit à cet élément.
export function choisirSeul(e) {
  app.marques = marquable(e) ? [cibleDe(e)] : []
  app.ancre = app.marques[0] ?? null
}

export const estMarque = (e) => marquable(e) && app.marques.some((m) => cleMarque(m) === cleMarque(cibleDe(e)))

export function toutDeselectionner() {
  app.marques = []
  app.ancre = null
}

// Après une opération de ménage : l'analyse a été mise à jour par le moteur,
// les vues se rechargent et le bilan s'affiche.
export async function apresMenage(bilan, operation) {
  app.bilan = { ...bilan, restants: bilan.restants ?? [], operation, quand: Date.now() }
  app.marques = []
  app.ancre = null
  app.selection = null
  app.revision++
  await refresh()
}

// Relance l'analyse affichée, qu'il s'agisse d'un dossier ou de tout le PC.
export const relancer = () => (app.etat?.scan?.pc ? lancerPC() : lancer(app.etat?.scan?.racine))

export async function choisirEtLancer() {
  app.erreur = ''
  try {
    const chemin = await call('ChoisirDossier')
    if (chemin) await lancer(chemin)
  } catch (err) {
    app.erreur = message(err)
  }
}

export async function arreter() {
  try {
    await call('Arreter')
  } catch (err) {
    app.erreur = message(err)
  }
  await refresh()
}

export async function montrer(gen, id, fichier = '') {
  try {
    await call('Montrer', gen, id, fichier)
  } catch (err) {
    app.erreur = message(err)
  }
}
