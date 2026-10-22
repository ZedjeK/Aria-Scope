<script>
  // Les 1 000 plus gros fichiers de l'analyse, relevés pendant le scan.
  // Clic : sélectionner (la carte et l'arbre suivent) ; double-clic : l'Explorateur.
  import Icon from './Icon.svelte'
  import { call, message, montrer, cliquerElement, estMarque } from './state.svelte.js'
  import { octets, nombre, dateCourte, dateLongue } from './format.js'
  import { estLigneChoisie } from './selection.js'

  let { gen, mode, selection = $bindable(null) } = $props()

  let tri = $state({ colonne: '', inverse: false })
  let lignes = $state(null)
  let erreur = $state('')

  $effect(() => {
    const g = gen
    const m = mode
    const t = { colonne: tri.colonne, inverse: tri.inverse }
    erreur = ''
    call('GrosFichiers', g, m, t)
      .then((l) => {
        if (g === gen && m === mode) lignes = l ?? []
      })
      .catch((err) => (erreur = message(err)))
  })

  const effective = $derived(tri.colonne || (mode === 'disque' ? 'disque' : 'taille'))
  const decroissant = $derived((effective !== 'nom') !== tri.inverse)

  function trierPar(colonne) {
    const cible = colonne || (mode === 'disque' ? 'disque' : 'taille')
    if (cible === effective && colonne === tri.colonne) tri = { colonne, inverse: !tri.inverse }
    else tri = { colonne, inverse: false }
  }

  const total = $derived((lignes ?? []).reduce((s, l) => s + (mode === 'disque' ? l.disque : l.taille), 0))
</script>

{#if erreur}
  <p class="error">{erreur}</p>
{:else if !lignes}
  <p class="hint">Chargement…</p>
{:else if lignes.length === 0}
  <p class="hint">Aucun fichier dans cette analyse.</p>
{:else}
  <p class="hint resume">
    Les {nombre(lignes.length)} plus gros fichiers pèsent ensemble <strong>{octets(total)}</strong>.
    Double-clic pour les montrer dans l'Explorateur.
  </p>
  <div class="liste" role="grid" aria-label="Plus gros fichiers">
    <div class="ligne entete" role="row">
      {#snippet entete(colonne, texte, classe = '')}
        {@const actifTri = colonne === tri.colonne}
        <span role="columnheader" class={classe} aria-sort={actifTri ? (decroissant ? 'descending' : 'ascending') : undefined}>
          <button class="tri" class:trie={actifTri} onclick={() => trierPar(colonne)}>
            {texte}<span class="fleche" aria-hidden="true">{actifTri ? (decroissant ? '▼' : '▲') : ''}</span>
          </button>
        </span>
      {/snippet}
      <span role="columnheader" class="num c-rang">#</span>
      {@render entete('nom', 'Nom', 'c-nom')}
      {@render entete('taille', 'Taille', `num c-taille${mode === 'taille' ? ' actif' : ''}`)}
      {@render entete('disque', 'Sur disque', `num c-disque${mode === 'disque' ? ' actif' : ''}`)}
      {@render entete('modif', 'Modifié', 'num c-modif')}
      <span role="columnheader" class="c-dossier">Dossier</span>
      <span role="columnheader" class="c-act"><span class="sr">Actions</span></span>
    </div>

    {#each lignes as f, i (`${f.id}/${f.nom}`)}
      {@const choisie = estMarque(f) || estLigneChoisie(selection, f)}
      <div
        class="ligne"
        class:choisie
        role="row"
        tabindex="0"
        aria-selected={choisie}
        onclick={(evt) => cliquerElement(f, 'liste', evt, lignes)}
        ondblclick={() => montrer(gen, f.id, f.nom)}
        onkeydown={(evt) => evt.key === 'Enter' && cliquerElement(f, 'liste', evt, lignes)}
      >
        <span class="num c-rang" role="gridcell">{i + 1}</span>
        <span class="c-nom" role="gridcell" title={f.nom}><Icon name="file" size={15} /><span class="nom">{f.nom}</span></span>
        <span class="num c-taille" role="gridcell" class:actif={mode === 'taille'}>{octets(f.taille)}</span>
        <span class="num c-disque" role="gridcell" class:actif={mode === 'disque'}>{octets(f.disque)}</span>
        <span class="num c-modif" role="gridcell" title="Modifié le {dateLongue(f.modif)}">{dateCourte(f.modif)}</span>
        <span class="c-dossier" role="gridcell" title={f.dossier}><bdi>{f.dossier}</bdi></span>
        <span class="c-act" role="gridcell">
          <button class="ouvrir" title="Montrer dans l'Explorateur" aria-label="Montrer dans l'Explorateur" onclick={(e) => (e.stopPropagation(), montrer(gen, f.id, f.nom))}>
            <Icon name="external" size={15} />
          </button>
        </span>
      </div>
    {/each}
  </div>
{/if}

<style>
  .resume {
    margin: 0 0 10px;
  }
  .resume strong {
    color: var(--text);
  }
  .liste {
    container-type: inline-size;
    background: var(--surface);
    border: 1px solid var(--line);
    border-radius: var(--cut) 0;
    corner-shape: bevel;
    overflow: clip;
    font-size: 13px;
  }
  .ligne {
    display: grid;
    grid-template-columns: 44px minmax(160px, 1.2fr) 92px 92px 100px minmax(0, 1fr) 40px;
    align-items: center;
    min-height: 32px;
    padding-right: 6px;
    border-top: 1px solid var(--line-soft);
    cursor: default;
  }
  .ligne:hover:not(.entete) {
    background: var(--hover);
  }
  .ligne {
    user-select: none;
  }
  .ligne:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: -2px;
  }
  .ligne.choisie {
    background: color-mix(in srgb, var(--action) 10%, transparent);
    box-shadow: inset 3px 0 0 var(--action);
  }
  .entete {
    position: sticky;
    top: 0;
    z-index: 1;
    min-height: 36px;
    border-top: none;
    background: var(--surface-sunken);
    font-size: 11px;
    letter-spacing: 1.5px;
    text-transform: uppercase;
    color: var(--faint);
  }
  .entete .num {
    display: flex;
    justify-content: flex-end;
  }
  .entete .actif {
    color: var(--accent);
  }
  .tri {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 6px 0;
    border: none;
    background: none;
    color: inherit;
    font: inherit;
    letter-spacing: inherit;
    text-transform: inherit;
    white-space: nowrap;
  }
  .tri:hover {
    color: var(--text);
  }
  .tri.trie {
    color: var(--action);
  }
  .fleche {
    display: inline-block;
    min-width: 8px;
    font-size: 9px;
  }
  .num {
    text-align: right;
    padding-right: 12px;
    font-variant-numeric: tabular-nums;
    color: var(--muted);
    white-space: nowrap;
  }
  .num.actif {
    color: var(--text);
    font-weight: 600;
  }
  .c-rang {
    color: var(--dim);
    font-size: 11px;
  }
  .c-nom {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
    padding: 0 8px;
    color: var(--faint);
  }
  .c-nom :global(svg) {
    flex: none;
  }
  .nom {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--text);
  }
  .c-dossier {
    min-width: 0;
    padding: 0 8px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    direction: rtl;
    text-align: left;
    font-size: 12px;
    color: var(--faint);
  }
  .entete .c-dossier {
    direction: ltr;
  }
  .c-act {
    display: flex;
    justify-content: center;
  }
  .ouvrir {
    display: flex;
    padding: 5px;
    border: 1px solid transparent;
    border-radius: 6px 0;
    corner-shape: bevel;
    background: none;
    color: var(--dim);
    opacity: 0;
  }
  .ligne:hover .ouvrir,
  .ouvrir:focus-visible {
    opacity: 1;
  }
  .ouvrir:hover {
    color: var(--action);
    border-color: var(--line-strong);
  }
  .sr {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
  }

  /* Colonne étroite : nom, taille du mode et dossier */
  @container (max-width: 760px) {
    .ligne {
      grid-template-columns: 36px minmax(0, 1fr) 88px minmax(0, 0.8fr) 36px;
    }
    .ligne > .c-modif,
    .ligne > .c-taille:not(.actif),
    .ligne > .c-disque:not(.actif) {
      display: none;
    }
  }
</style>
