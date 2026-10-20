// Mise en forme des tailles, nombres et durées, à la française.

const UNITES = ['o', 'Ko', 'Mo', 'Go', 'To', 'Po']

// Trois chiffres significatifs, comme l'Explorateur : 9,87 Go · 98,7 Go · 987 Go.
export function octets(n) {
  if (!Number.isFinite(n) || n <= 0) return '0 o'
  let i = 0
  while (n >= 1024 && i < UNITES.length - 1) {
    n /= 1024
    i++
  }
  const decimales = i === 0 ? 0 : n < 10 ? 2 : n < 100 ? 1 : 0
  return `${n.toLocaleString('fr-FR', { minimumFractionDigits: decimales, maximumFractionDigits: decimales })} ${UNITES[i]}`
}

export function nombre(n) {
  return (n ?? 0).toLocaleString('fr-FR')
}

export function pourcent(part, total) {
  if (!total || total <= 0) return 0
  return Math.max(0, Math.min(100, (part / total) * 100))
}

export function pourcentTexte(p) {
  if (p > 0 && p < 0.1) return '< 0,1 %'
  const d = p > 0 && p < 10 ? 1 : 0
  return `${p.toLocaleString('fr-FR', { minimumFractionDigits: d, maximumFractionDigits: d })} %`
}

export function duree(ms) {
  const s = Math.round((ms ?? 0) / 1000)
  if (s < 1) return `${(ms / 1000).toLocaleString('fr-FR', { maximumFractionDigits: 1 })} s`
  if (s < 60) return `${s} s`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m} min ${String(s % 60).padStart(2, '0')} s`
  return `${Math.floor(m / 60)} h ${String(m % 60).padStart(2, '0')} min`
}

export function parSeconde(n, ms) {
  if (!ms || ms < 500) return ''
  return `${Math.round((n * 1000) / ms).toLocaleString('fr-FR')}/s`
}
