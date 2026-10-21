// État partagé de l'interface et appels au backend Go (ScopeService).
import { Call } from '@wailsio/runtime'

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
  selection: null, // partagée entre l'arbre et la carte (voir selection.js)
  erreur: '',
  loadError: '',
})

export const scanEnCours = () => app.etat?.scan?.etat === 'scan'

export async function refresh() {
  try {
    app.etat = await call('Etat')
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

export async function lancer(chemin) {
  app.erreur = ''
  app.selection = null
  try {
    await call('Lancer', chemin)
    app.page = 'analyse'
  } catch (err) {
    app.erreur = message(err)
  }
  await refresh()
}

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
