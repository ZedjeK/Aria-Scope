// Filtres de scan : valeurs vides, choix proposés et résumé lisible.
import { octets } from './format.js'

export const filtresVides = () => ({ exclure: [], inclure: [], regex: '', tailleMin: 0, ageMinJours: 0 })

export const actifs = (f) =>
  !!f && (f.exclure?.length > 0 || f.inclure?.length > 0 || !!f.regex?.trim() || f.tailleMin > 0 || f.ageMinJours > 0)

const Mo = 1024 * 1024
export const TAILLES = [
  [0, 'toutes tailles'],
  [Mo, 'au moins 1 Mo'],
  [10 * Mo, 'au moins 10 Mo'],
  [100 * Mo, 'au moins 100 Mo'],
  [1024 * Mo, 'au moins 1 Go'],
]

export const AGES = [
  [0, 'toutes dates'],
  [30, "plus d'un mois"],
  [182, 'plus de 6 mois'],
  [365, "plus d'un an"],
  [730, 'plus de 2 ans'],
  [1826, 'plus de 5 ans'],
]

// « node_modules, *.iso ; *.tmp » -> ["node_modules", "*.iso", "*.tmp"]
export const decouper = (texte) =>
  (texte ?? '')
    .split(/[,;\n]/)
    .map((m) => m.trim())
    .filter(Boolean)

export const joindre = (motifs) => (motifs ?? []).join(', ')

// Résumé court de chaque filtre, pour les pastilles.
export function resume(f) {
  if (!actifs(f)) return []
  const out = []
  if (f.exclure?.length) out.push(`sans ${f.exclure.join(', ')}`)
  if (f.inclure?.length) out.push(`seulement ${f.inclure.join(', ')}`)
  if (f.regex?.trim()) out.push(`nom ~ /${f.regex.trim()}/`)
  if (f.tailleMin > 0) out.push(TAILLES.find(([v]) => v === f.tailleMin)?.[1].replace('au moins', '≥') ?? `≥ ${octets(f.tailleMin)}`)
  if (f.ageMinJours > 0) out.push(`modifiés il y a ${AGES.find(([j]) => j === f.ageMinJours)?.[1] ?? `plus de ${f.ageMinJours} jours`}`)
  return out
}
