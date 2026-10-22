<script>
  // Graphique en anneau : la répartition du dossier sélectionné entre ses 8 plus
  // gros éléments, le reste regroupé. Cliquer sur une part la sélectionne (et
  // l'anneau plonge dedans si c'est un dossier).
  import Icon from './Icon.svelte'
  import { app, call, message, choisirSeul } from './state.svelte.js'
  import { octets, nombre, pourcent, pourcentTexte } from './format.js'
  import { selectionDe } from './selection.js'
  import { TEINTES, GRIS_FICHIERS, GRIS_RESTE } from './treemap.js'

  let { gen, mode, selection = $bindable(null) } = $props()

  const PARTS = 8

  // Le dossier montré : celui qui est sélectionné, ou celui qui contient les
  // fichiers sélectionnés (leur id est celui de leur dossier), sinon la racine.
  const dossierId = $derived(selection ? selection.id : 0)

  let lignes = $state([])
  let fil = $state([])
  let erreur = $state('')
  let survol = $state(-1)

  $effect(() => {
    const g = gen
    const id = dossierId
    const m = mode
    app.revision
    erreur = ''
    Promise.all([call('Enfants', g, id, m, { colonne: '', inverse: false }), call('Fil', g, id)])
      .then(([e, f]) => {
        if (g !== gen || id !== dossierId || m !== mode) return
        lignes = e ?? []
        fil = f ?? []
        survol = -1
      })
      .catch((err) => (erreur = message(err)))
  })

  const valeur = (e) => (mode === 'disque' ? e.disque : e.taille)

  // Les 8 plus gros (dossiers ou fichiers en vrac), puis « autres ».
  const parts = $derived.by(() => {
    const out = []
    let autres = { valeur: 0, nombre: 0 }
    for (const e of lignes) {
      const v = valeur(e)
      if (v <= 0) continue
      if (e.type === 'reste') {
        autres.valeur += v
        autres.nombre += e.nombre
      } else if (out.length < PARTS) {
        const couleur = e.type === 'fichiers' ? GRIS_FICHIERS : TEINTES[out.length % TEINTES.length]
        out.push({ e, valeur: v, couleur })
      } else {
        autres.valeur += v
        autres.nombre += 1
      }
    }
    if (autres.valeur > 0) out.push({ e: null, valeur: autres.valeur, nombre: autres.nombre, couleur: GRIS_RESTE })
    return out
  })
  const total = $derived(parts.reduce((s, p) => s + p.valeur, 0))

  // Géométrie de l'anneau (viewBox 240 × 240)
  const C = 120
  const R = 104
  const r = 66
  function arc(debut, fin, rExt, rInt) {
    const pt = (a, rr) => [C + rr * Math.sin(a), C - rr * Math.cos(a)]
    const grand = fin - debut > Math.PI ? 1 : 0
    const [x1, y1] = pt(debut, rExt)
    const [x2, y2] = pt(fin, rExt)
    const [x3, y3] = pt(fin, rInt)
    const [x4, y4] = pt(debut, rInt)
    return `M${x1},${y1} A${rExt},${rExt} 0 ${grand} 1 ${x2},${y2} L${x3},${y3} A${rInt},${rInt} 0 ${grand} 0 ${x4},${y4} Z`
  }
  const segments = $derived.by(() => {
    let a = 0
    return parts.map((p) => {
      const angle = total > 0 ? (p.valeur / total) * 2 * Math.PI : 0
      const s = { debut: a, fin: a + angle, complet: angle >= 2 * Math.PI - 1e-6 }
      a += angle
      return s
    })
  })

  const nomPart = (p) =>
    !p.e
      ? `${nombre(p.nombre)} autre${p.nombre > 1 ? 's' : ''}`
      : p.e.type === 'fichiers'
        ? `${nombre(p.e.fichiers)} fichier${p.e.fichiers > 1 ? 's' : ''} en vrac`
        : p.e.nom

  function choisir(p) {
    if (!p.e) return
    selection = selectionDe(p.e, 'anneau')
    choisirSeul(p.e)
  }

  function remonter() {
    if (fil.length >= 2) {
      const parent = fil[fil.length - 2]
      selection = { type: 'dossier', id: parent.id, nom: parent.nom, origine: 'anneau' }
    }
  }

  const actif = $derived(survol >= 0 ? parts[survol] : null)
  const nomDossier = $derived(fil.at(-1)?.nom ?? '')
</script>

<section class="anneau-cadre" aria-label="Répartition du dossier sélectionné">
  <div class="tete">
    <button class="monter" onclick={remonter} disabled={fil.length < 2} title="Dossier parent" aria-label="Dossier parent">
      <Icon name="up" size={15} />
    </button>
    <span class="titre" title={nomDossier}>{nomDossier}</span>
  </div>

  {#if erreur}
    <p class="error">{erreur}</p>
  {:else if total === 0}
    <p class="hint vide">Rien à répartir : ce dossier est vide.</p>
  {:else}
    <div class="corps">
      <svg viewBox="0 0 240 240" class="graphe" role="img" aria-label="Anneau de répartition">
        {#each parts as p, i (i)}
          {@const s = segments[i]}
          {#if s.complet}
            <circle cx={C} cy={C} r={(R + r) / 2} fill="none" stroke={p.couleur} stroke-width={R - r} class="part" class:eteinte={survol >= 0 && survol !== i} />
          {:else if s.fin - s.debut > 0.002}
            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <path
              d={arc(s.debut, s.fin, survol === i ? R + 5 : R, r)}
              fill={p.couleur}
              class="part"
              class:eteinte={survol >= 0 && survol !== i}
              class:cliquable={!!p.e}
              onmouseenter={() => (survol = i)}
              onmouseleave={() => (survol = -1)}
              onclick={() => choisir(p)}
            />
          {/if}
        {/each}
        <text x={C} y={C - 6} class="centre-valeur">{octets(actif ? actif.valeur : total)}</text>
        <text x={C} y={C + 14} class="centre-texte">{actif ? pourcentTexte(pourcent(actif.valeur, total)) : 'au total'}</text>
      </svg>

      <ul class="legende">
        {#each parts as p, i (i)}
          <li>
            <button class="ligne" class:active={survol === i} disabled={!p.e} onmouseenter={() => (survol = i)} onmouseleave={() => (survol = -1)} onfocus={() => (survol = i)} onblur={() => (survol = -1)} onclick={() => choisir(p)}>
              <span class="pastille" style:background={p.couleur}></span>
              <span class="nom">{nomPart(p)}</span>
              <span class="taille">{octets(p.valeur)}</span>
              <span class="pct">{pourcentTexte(pourcent(p.valeur, total))}</span>
            </button>
          </li>
        {/each}
      </ul>
    </div>
  {/if}

  <div class="pied">
    <span class="hint">Clic sur une part : la sélectionner et plonger dedans</span>
  </div>
</section>

<style>
  .anneau-cadre {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    background: var(--surface);
    border: 1px solid var(--line);
    border-radius: var(--cut) 0;
    corner-shape: bevel;
    overflow: clip;
  }
  .tete,
  .pied {
    flex: none;
    display: flex;
    align-items: center;
    gap: 8px;
    min-height: 38px;
    padding: 0 10px;
    background: var(--surface-sunken);
    font-size: 12px;
  }
  .tete {
    border-bottom: 1px solid var(--line-soft);
  }
  .pied {
    border-top: 1px solid var(--line-soft);
  }
  .monter {
    flex: none;
    display: flex;
    padding: 5px;
    border: 1px solid var(--line-strong);
    border-radius: 6px 0;
    corner-shape: bevel;
    background: none;
    color: var(--muted);
  }
  .monter:hover:not(:disabled) {
    color: var(--action);
    border-color: var(--action);
  }
  .monter:disabled {
    opacity: 0.35;
  }
  .titre {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--accent);
  }
  .vide {
    padding: 20px;
  }

  .corps {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 14px;
    padding: 16px;
  }
  .graphe {
    width: min(100%, 300px);
    flex: none;
    overflow: visible;
  }
  .part {
    transition: opacity 0.15s;
    stroke: var(--surface);
    stroke-width: 1.5;
  }
  circle.part {
    stroke-width: 38;
  }
  .part.cliquable {
    cursor: pointer;
  }
  .part.eteinte {
    opacity: 0.35;
  }
  .centre-valeur {
    text-anchor: middle;
    font-family: var(--font-display);
    font-size: 22px;
    font-weight: 700;
    fill: #ffffff;
  }
  .centre-texte {
    text-anchor: middle;
    font-family: var(--font-mono);
    font-size: 11px;
    fill: var(--muted);
  }

  .legende {
    width: 100%;
    margin: 0;
    padding: 0;
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .ligne {
    width: 100%;
    display: grid;
    grid-template-columns: 12px minmax(0, 1fr) auto 52px;
    align-items: center;
    gap: 10px;
    min-height: 32px;
    padding: 0 10px;
    border: none;
    border-radius: 6px 0;
    corner-shape: bevel;
    background: none;
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 12px;
    text-align: left;
  }
  .ligne:disabled {
    cursor: default;
    color: var(--faint);
  }
  .ligne.active,
  .ligne:hover:not(:disabled) {
    background: var(--hover);
  }
  .pastille {
    width: 12px;
    height: 12px;
    border-radius: 3px;
  }
  .nom {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .taille {
    color: var(--muted);
    font-variant-numeric: tabular-nums;
  }
  .pct {
    text-align: right;
    color: var(--faint);
    font-variant-numeric: tabular-nums;
  }
</style>
