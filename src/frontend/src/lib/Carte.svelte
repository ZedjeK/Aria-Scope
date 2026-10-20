<script>
  // Treemap néon : chaque rectangle a une surface proportionnelle à sa taille.
  // Survol : le détail ; clic : sélection ; double-clic : on plonge dans le dossier.
  import { untrack } from 'svelte'
  import Icon from './Icon.svelte'
  import { selectionDe, estBlocChoisi } from './selection.js'
  import { call, message, montrer } from './state.svelte.js'
  import { octets, nombre, pourcentTexte, pourcent } from './format.js'
  import { disposer } from './treemap.js'

  let { gen, mode, selection = $bindable(null) } = $props()

  let racineId = $state(0) // dossier affiché par la carte
  let donnees = $state(null)
  let fil = $state([])
  let erreur = $state('')
  let chargement = $state(false)
  let largeur = $state(0)
  let hauteur = $state(0)
  let survol = $state(null) // { tuile, x, y }

  // Nouvelle analyse : on repart de la racine.
  $effect(() => {
    gen
    racineId = 0
  })

  // (Re)chargement des blocs quand le dossier affiché ou le mode change.
  $effect(() => {
    const g = gen
    const id = racineId
    const m = mode
    chargement = true
    erreur = ''
    Promise.all([call('Carte', g, id, m), call('Fil', g, id)])
      .then(([c, f]) => {
        if (g !== gen || id !== racineId || m !== mode) return
        donnees = c
        fil = f ?? []
      })
      .catch((err) => (erreur = message(err)))
      .finally(() => (chargement = false))
  })

  const tuiles = $derived(donnees && largeur > 0 && hauteur > 0 ? disposer(donnees, largeur, hauteur) : [])
  const tuileChoisie = $derived(selection ? (tuiles.find((t) => estBlocChoisi(selection, t.bloc)) ?? null) : null)

  // Choisi ailleurs (arbre, anneau) mais absent de la carte : on recadre sur son dossier
  // parent (ou, pour des fichiers, sur le dossier qui les contient).
  let dejaRecadre = ''
  $effect(() => {
    const sel = selection
    const ts = tuiles
    if (!sel || sel.origine === 'carte' || !donnees || chargement) return
    if (ts.some((t) => estBlocChoisi(sel, t.bloc))) return
    if (sel.type === 'dossier' && sel.id === racineId) return // c'est le dossier affiché
    const cle = `${sel.type}:${sel.id}`
    if (cle === dejaRecadre) return // trop petit pour être dessiné même là : tant pis
    dejaRecadre = cle
    untrack(() => recadrer(sel))
  })

  async function recadrer(sel) {
    if (sel.type !== 'dossier') {
      racineId = sel.id
      return
    }
    try {
      const f = (await call('Fil', gen, sel.id)) ?? []
      if (f.length >= 2) racineId = f[f.length - 2].id
    } catch {}
  }

  const nomBloc = (b) =>
    b.type === 'fichiers'
      ? `${nombre(b.fichiers)} fichier${b.fichiers > 1 ? 's' : ''}`
      : b.type === 'reste'
        ? `${nombre(b.nombre)} autre${b.nombre > 1 ? 's' : ''}`
        : b.nom

  // Le nom tient-il dans la tuile ? Plex Mono 10,5 px : environ 6,3 px par caractère.
  function etiquette(t) {
    const place = Math.floor((t.w - 8) / 6.3)
    if (t.h < 14 || place < 3) return ''
    const nom = nomBloc(t.bloc)
    return nom.length <= place ? nom : nom.slice(0, place - 1) + '…'
  }

  function tuileDe(evt) {
    const i = evt.target?.dataset?.i
    return i === undefined ? null : tuiles[Number(i)]
  }

  function bouger(evt) {
    const t = tuileDe(evt)
    if (!t) {
      survol = null
      return
    }
    const cadre = evt.currentTarget.getBoundingClientRect()
    survol = { tuile: t, x: evt.clientX - cadre.left, y: evt.clientY - cadre.top }
  }

  function cliquer(evt) {
    const t = tuileDe(evt)
    if (t && t.bloc.type !== 'reste') selection = selectionDe(t.bloc, 'carte')
  }

  function plonger(evt) {
    const t = tuileDe(evt)
    if (t?.bloc.type === 'dossier' && t.bloc.ouvrable) {
      survol = null
      selection = selectionDe(t.bloc, 'carte')
      racineId = t.bloc.id
    }
  }

  const remonter = () => fil.length > 1 && (racineId = fil[fil.length - 2].id)

  function clavier(evt) {
    if (evt.key === 'Backspace' || evt.key === 'Escape') {
      if (selection && evt.key === 'Escape') selection = null
      else remonter()
    }
    const b = tuileChoisie?.bloc
    if (evt.key === 'Enter' && b?.type === 'dossier' && b.ouvrable) racineId = b.id
  }

  // Bulle : on la garde dans le cadre.
  const bulleX = $derived(survol ? Math.min(survol.x + 14, Math.max(0, largeur - 250)) : 0)
  const bulleY = $derived(survol ? (survol.y + 110 > hauteur ? survol.y - 96 : survol.y + 16) : 0)
</script>

<section class="carte" aria-label="Carte de l'espace disque">
  <div class="tete">
    <button class="monter" onclick={remonter} disabled={fil.length < 2} title="Remonter d'un niveau" aria-label="Remonter d'un niveau">
      <Icon name="up" size={15} />
    </button>
    <nav class="fil" aria-label="Fil d'Ariane">
      {#each fil as etape, i (etape.id)}
        {#if i > 0}<span class="sep" aria-hidden="true">›</span>{/if}
        <button class="maillon" class:courant={i === fil.length - 1} onclick={() => (racineId = etape.id)} title={etape.nom}>
          {etape.nom}
        </button>
      {/each}
    </nav>
    {#if chargement}<span class="hint">…</span>{/if}
  </div>

  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
  <div class="zone" bind:clientWidth={largeur} bind:clientHeight={hauteur} tabindex="0" role="application" aria-label="Treemap : clic pour sélectionner, double-clic ou Entrée pour plonger, Retour arrière pour remonter" onkeydown={clavier} onmousemove={bouger} onmouseleave={() => (survol = null)} onclick={cliquer} ondblclick={plonger}>
    {#if erreur}
      <p class="error">{erreur}</p>
    {:else if largeur > 0}
      <svg width={largeur} height={hauteur}>
        {#each tuiles as t, i (i)}
          {@const txt = etiquette(t)}
          {@const opac = t.interieur ? 0.06 + 0.03 * t.prof : t.bloc.type === 'reste' ? 0.1 : 0.2 + 0.04 * t.prof}
          <rect
            data-i={i}
            x={t.x + 0.5}
            y={t.y + 0.5}
            width={Math.max(0, t.w - 1)}
            height={Math.max(0, t.h - 1)}
            rx="2"
            fill={t.couleur}
            fill-opacity={survol?.tuile === t ? opac + 0.14 : opac}
            stroke={t.couleur}
            stroke-opacity={t.interieur ? 0.55 : 0.4}
            class:plongeable={t.bloc.type === 'dossier' && t.bloc.ouvrable}
          />
          {#if txt}
            <text x={t.x + 5} y={t.y + 11.5} class="lbl" class:tete-dossier={t.interieur} fill={t.interieur ? t.couleur : '#ddebff'}>{txt}</text>
            {#if !t.interieur && t.h > 30 && t.w > 56}
              <text x={t.x + 5} y={t.y + 25} class="lbl taille">{octets(t.bloc.valeur)}</text>
            {/if}
          {/if}
        {/each}
        {#if tuileChoisie}
          {@const s = tuileChoisie}
          <rect class="selection" x={s.x + 1} y={s.y + 1} width={Math.max(0, s.w - 2)} height={Math.max(0, s.h - 2)} rx="2" />
        {/if}
      </svg>

      {#if survol}
        {@const b = survol.tuile.bloc}
        <div class="bulle" style:left="{bulleX}px" style:top="{bulleY}px">
          <strong>{nomBloc(b)}</strong>
          <span>{octets(b.valeur)} · {pourcentTexte(pourcent(b.valeur, donnees.valeur))} de {fil.at(-1)?.nom ?? ''}</span>
          {#if b.type === 'dossier'}<span>{nombre(b.fichiers)} fichier{b.fichiers > 1 ? 's' : ''}</span>{/if}
          {#if b.type === 'dossier' && b.ouvrable}<span class="astuce">double-clic pour plonger</span>{/if}
          {#if b.type === 'reste'}<span class="astuce">trop petits pour être dessinés un par un</span>{/if}
        </div>
      {/if}
    {/if}
  </div>

  <div class="pied">
    {#if tuileChoisie}
      {@const b = tuileChoisie.bloc}
      <span class="pastille" style:background={tuileChoisie.couleur}></span>
      <span class="sel-nom">{nomBloc(b)}</span>
      <span class="sel-taille">{octets(b.valeur)} · {pourcentTexte(pourcent(b.valeur, donnees.valeur))}</span>
      <button class="ouvrir" onclick={() => montrer(gen, b.id)} title="Montrer dans l'Explorateur" aria-label="Montrer dans l'Explorateur"><Icon name="external" size={15} /></button>
    {:else if selection?.type === 'dossier' && selection.id === racineId}
      <span class="sel-nom">{fil.at(-1)?.nom ?? ''}</span>
      <span class="hint sel-taille">dossier affiché par la carte</span>
    {:else if selection}
      <span class="sel-nom">{selection.nom || 'Sélection'}</span>
      <span class="hint sel-taille">trop petit pour la carte</span>
    {:else}
      <span class="hint">Clic : sélection · double-clic : plonger · ⌫ : remonter</span>
    {/if}
  </div>
</section>

<style>
  .carte {
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
  .monter,
  .ouvrir {
    flex: none;
    display: flex;
    padding: 5px;
    border: 1px solid var(--line-strong);
    border-radius: 6px 0;
    corner-shape: bevel;
    background: none;
    color: var(--muted);
  }
  .monter:hover:not(:disabled),
  .ouvrir:hover {
    color: var(--action);
    border-color: var(--action);
  }
  .monter:disabled {
    opacity: 0.35;
  }
  .fil {
    display: flex;
    align-items: center;
    gap: 4px;
    min-width: 0;
    overflow: hidden;
    white-space: nowrap;
  }
  .maillon {
    min-width: 0;
    max-width: 180px;
    padding: 3px 4px;
    border: none;
    background: none;
    color: var(--faint);
    font-family: var(--font-mono);
    font-size: 12px;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .maillon:hover {
    color: var(--text);
  }
  .maillon.courant {
    flex: none;
    max-width: 260px;
    color: var(--accent);
  }
  .sep {
    color: var(--dim);
  }

  .zone {
    position: relative;
    flex: 1;
    min-height: 0;
    outline: none;
  }
  .zone:focus-visible {
    box-shadow: inset 0 0 0 2px var(--accent);
  }
  svg {
    display: block;
  }
  rect {
    transition: fill-opacity 0.12s;
  }
  rect.plongeable {
    cursor: zoom-in;
  }
  .lbl {
    font-family: var(--font-mono);
    font-size: 10.5px;
    pointer-events: none;
  }
  .tete-dossier {
    font-weight: 600;
  }
  .taille {
    fill: var(--muted);
  }
  .selection {
    fill: none;
    stroke: var(--action);
    stroke-width: 2;
    filter: drop-shadow(0 0 4px var(--action));
    pointer-events: none;
  }

  .bulle {
    position: absolute;
    z-index: 2;
    display: flex;
    flex-direction: column;
    gap: 3px;
    max-width: 240px;
    padding: 8px 10px;
    background: color-mix(in srgb, var(--bg) 92%, transparent);
    border: 1px solid var(--line-strong);
    border-radius: 8px 0;
    corner-shape: bevel;
    box-shadow: 0 6px 20px rgba(0, 0, 0, 0.5);
    font-size: 12px;
    color: var(--muted);
    pointer-events: none;
  }
  .bulle strong {
    color: #ffffff;
    overflow-wrap: anywhere;
  }
  .astuce {
    color: var(--faint);
    font-size: 11px;
  }

  .pastille {
    flex: none;
    width: 10px;
    height: 10px;
    border-radius: 2px;
    box-shadow: 0 0 6px currentColor;
  }
  .sel-nom {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--text);
    font-weight: 600;
  }
  .sel-taille {
    flex: none;
    margin-left: auto;
    color: var(--muted);
  }
</style>
