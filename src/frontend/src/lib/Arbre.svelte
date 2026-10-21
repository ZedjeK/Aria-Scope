<script>
  // Arborescence de l'analyse : chaque dossier se déplie à la demande, ses
  // enfants sont demandés au moteur Go, déjà triés selon la colonne choisie.
  import { tick, untrack } from 'svelte'
  import Icon from './Icon.svelte'
  import { selectionDe, estLigneChoisie } from './selection.js'
  import { call, message, montrer } from './state.svelte.js'
  import { octets, nombre, pourcent, pourcentTexte, dateCourte, dateLongue } from './format.js'

  let { gen, mode, pc = false, tri = $bindable({ colonne: '', inverse: false }), selection = $bindable(null) } = $props()

  let racine = $state(null)
  let erreurRacine = $state('')
  let uid = 0

  const nouveau = (e) => ({ uid: ++uid, e, ouvert: false, enfants: null, chargement: false, erreur: '' })
  const cle = (e) => `${e.type}:${e.id}:${e.nom}`
  const valeur = (e) => (mode === 'disque' ? e.disque : e.taille)

  async function lire(n) {
    const methode = n.e.type === 'fichiers' ? 'Fichiers' : 'Enfants'
    return (await call(methode, gen, n.e.id, mode, { colonne: tri.colonne, inverse: tri.inverse })) ?? []
  }

  async function charger(n) {
    n.chargement = true
    n.erreur = ''
    try {
      n.enfants = (await lire(n)).map(nouveau)
    } catch (err) {
      n.erreur = message(err)
    } finally {
      n.chargement = false
    }
  }

  async function ouvrir(n) {
    if (!n.e.ouvrable) return
    n.ouvert = true
    if (!n.enfants && !n.chargement) await charger(n)
  }

  // ---------- Sélection, partagée avec la carte ----------

  const choisir = (n) => (selection = selectionDe(n.e, 'arbre'))

  // Choisi ailleurs (carte, anneau) : on déplie le chemin jusqu'à la ligne et on la montre.
  async function reveler(sel) {
    let fil
    try {
      fil = (await call('Fil', gen, sel.id)) ?? []
    } catch {
      return
    }
    let n = racine
    for (const etape of fil.slice(1)) {
      await ouvrir(n)
      const c = n.enfants?.find((c) => c.e.type === 'dossier' && c.e.id === etape.id)
      if (!c) return // au-delà des 300 premiers : regroupé dans « autres »
      n = c
    }
    if (sel.type !== 'dossier') {
      await ouvrir(n)
      if (sel.type === 'fichier') {
        const f = n.enfants?.find((c) => c.e.type === 'fichiers')
        if (f) await ouvrir(f)
      }
    }
    await tick()
    document.querySelector('.arbre .ligne.choisie')?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
  }

  $effect(() => {
    const sel = selection
    const r = racine
    if (sel && sel.origine !== 'arbre' && r) untrack(() => reveler(sel))
  })

  function basculer(n) {
    if (!n.e.ouvrable) return
    n.ouvert = !n.ouvert
    if (n.ouvert && !n.enfants && !n.chargement) charger(n)
  }

  // Nouveau mode de mesure ou nouveau tri : on retrie tout ce qui est déplié,
  // sans rien refermer.
  async function recharger(n) {
    if (!n.ouvert || !n.enfants) return
    const anciens = new Map(n.enfants.map((c) => [cle(c.e), c]))
    try {
      n.enfants = (await lire(n)).map((e) => {
        const a = anciens.get(cle(e))
        if (!a) return nouveau(e)
        a.e = e
        return a
      })
    } catch (err) {
      n.erreur = message(err)
      return
    }
    await Promise.all(n.enfants.map(recharger))
  }

  // Nouvelle analyse terminée : on repart de sa racine, dépliée.
  $effect(() => {
    const g = gen
    racine = null
    erreurRacine = ''
    call('Racine', g)
      .then((e) => {
        if (g !== gen) return
        racine = nouveau(e)
        basculer(racine) // le proxy réactif, pas l'objet brut
      })
      .catch((err) => (erreurRacine = message(err)))
  })

  let ordreAffiche = untrack(() => `${mode}|${tri.colonne}|${tri.inverse}`) // ordre de départ
  $effect(() => {
    const ordre = `${mode}|${tri.colonne}|${tri.inverse}`
    if (ordre !== ordreAffiche) {
      ordreAffiche = ordre
      if (racine) recharger(racine)
    }
  })

  // ---------- Tri par colonne ----------

  // La colonne réellement utilisée : « % du parent » trie sur la taille du mode.
  const effective = $derived(tri.colonne || (mode === 'disque' ? 'disque' : 'taille'))
  const decroissant = $derived((effective !== 'nom') !== tri.inverse) // la date : la plus récente d'abord

  // Premier clic : la colonne, dans son sens naturel ; clic suivant : on inverse.
  function trierPar(colonne) {
    const cible = colonne || (mode === 'disque' ? 'disque' : 'taille')
    if (cible === effective && colonne === tri.colonne) tri = { colonne, inverse: !tri.inverse }
    else tri = { colonne, inverse: false }
  }

  const triParTaille = $derived(decroissant && (effective === 'taille' || effective === 'disque'))

  const lignes = $derived.by(() => {
    const out = []
    const visiter = (n, prof, parent) => {
      out.push({ n, prof, parent })
      if (n.ouvert) {
        if (n.enfants) for (const c of n.enfants) visiter(c, prof + 1, n)
        if (n.chargement || n.erreur) out.push({ n, prof: prof + 1, parent: n, etat: true })
      }
    }
    if (racine) visiter(racine, 0, null)
    return out
  })

  function libelle(e) {
    switch (e.type) {
      case 'fichiers':
        return `${nombre(e.fichiers)} fichier${e.fichiers > 1 ? 's' : ''} dans ce dossier`
      case 'reste':
        return `${nombre(e.nombre)} autre${e.nombre > 1 ? 's' : ''} élément${e.nombre > 1 ? 's' : ''}${triParTaille ? ', plus légers' : ''}`
      default:
        return e.nom
    }
  }

  const icone = { dossier: 'folder', fichiers: 'files', fichier: 'file', reste: 'more' }

  function ouvrirExplorateur(n) {
    const e = n.e
    if (e.type === 'fichier') montrer(gen, e.id, e.nom)
    else montrer(gen, e.id)
  }
</script>

{#if erreurRacine}
  <p class="error">{erreurRacine}</p>
{:else if !racine}
  <p class="hint">Chargement du résultat…</p>
{:else}
  <div class="arbre" role="treegrid" aria-label="Contenu analysé">
    <div class="ligne entete" role="row">
      {#snippet entete(colonne, texte, classe = '')}
        {@const actifTri = colonne === tri.colonne}
        <span role="columnheader" class={classe} aria-sort={actifTri ? (decroissant ? 'descending' : 'ascending') : undefined}>
          <button class="tri" class:trie={actifTri} onclick={() => trierPar(colonne)} title="Trier par {texte.toLowerCase()}">
            {texte}<span class="fleche" aria-hidden="true">{actifTri ? (decroissant ? '▼' : '▲') : ''}</span>
          </button>
        </span>
      {/snippet}
      {@render entete('nom', 'Nom', 'c-entete-nom')}
      {@render entete('taille', 'Taille', `num c-taille${mode === 'taille' ? ' actif' : ''}`)}
      {@render entete('disque', 'Sur disque', `num c-disque${mode === 'disque' ? ' actif' : ''}`)}
      {@render entete('', '% du parent', 'c-pct')}
      {@render entete('fichiers', 'Fichiers', 'num c-fichiers')}
      {@render entete('dossiers', 'Dossiers', 'num c-dossiers')}
      {@render entete('modif', 'Modifié', 'num c-modif')}
      <span role="columnheader" class="c-act"><span class="sr">Actions</span></span>
    </div>

    {#each lignes as { n, prof, parent, etat } (etat ? `etat-${n.uid}` : n.uid)}
      {#if etat}
        <div class="ligne info" role="row" style:--prof={prof}>
          <span class="c-nom" role="gridcell">
            {#if n.erreur}<span class="error">{n.erreur}</span>{:else}<span class="hint">lecture…</span>{/if}
          </span>
        </div>
      {:else}
        {@const e = n.e}
        {@const p = parent ? pourcent(valeur(e), valeur(parent.e)) : 100}
        {@const choisie = estLigneChoisie(selection, e)}
        <div class="ligne" class:racine={prof === 0} class:attenue={e.type === 'reste'} class:choisie role="row" aria-level={prof + 1} aria-expanded={e.ouvrable ? n.ouvert : undefined} aria-selected={choisie} style:--prof={prof}>
          <div class="c-nom" role="gridcell">
            <button class="pli" class:ouvert={n.ouvert} class:invisible={!e.ouvrable} tabindex={e.ouvrable ? 0 : -1} onclick={() => basculer(n)} aria-label={n.ouvert ? 'Replier' : 'Déplier'}>
              <Icon name="chevron" size={14} />
            </button>
            <button class="nom-btn" onclick={() => choisir(n)} ondblclick={() => basculer(n)} title={prof === 0 ? e.nom : e.type === 'reste' ? undefined : 'Clic : sélectionner · double-clic : déplier'}>
              <span class="type t-{e.type}"><Icon name={pc && prof === 0 ? 'scope' : pc && prof === 1 ? 'disk' : icone[e.type]} size={16} /></span>
              <span class="nom" class:italique={e.type === 'fichiers' || e.type === 'reste'}>{libelle(e)}</span>
              {#if e.erreurs > 0 && e.type === 'dossier'}
                <span class="chip warn" title="{nombre(e.erreurs)} dossier{e.erreurs > 1 ? 's' : ''} illisible{e.erreurs > 1 ? 's' : ''} dans cette branche">
                  <Icon name="warn" size={11} />{nombre(e.erreurs)}
                </span>
              {/if}
            </button>
          </div>
          <span class="num c-taille" role="gridcell" class:actif={mode === 'taille'}>{octets(e.taille)}</span>
          <span class="num c-disque" role="gridcell" class:actif={mode === 'disque'}>{octets(e.disque)}</span>
          <span class="c-pct" role="gridcell">
            <span class="barre"><span style:width="{p}%"></span></span>
            <span class="pct">{pourcentTexte(p)}</span>
          </span>
          <span class="num c-fichiers" role="gridcell">{e.type === 'fichier' ? '' : nombre(e.fichiers)}</span>
          <span class="num c-dossiers" role="gridcell">{e.type === 'dossier' || e.type === 'reste' ? nombre(e.dossiers) : ''}</span>
          <span class="num c-modif" role="gridcell" title={e.type === 'fichier' ? `Modifié le ${dateLongue(e.modif)}` : e.modif ? `Fichier le plus récent : ${dateLongue(e.modif)}` : 'Aucun fichier'}>{dateCourte(e.modif)}</span>
          <span class="c-act" role="gridcell">
            {#if e.type !== 'reste' && !(pc && prof === 0)}
              <button class="ouvrir" title="Montrer dans l'Explorateur" aria-label="Montrer dans l'Explorateur" onclick={() => ouvrirExplorateur(n)}>
                <Icon name="external" size={15} />
              </button>
            {/if}
          </span>
        </div>
      {/if}
    {/each}
  </div>
{/if}

<style>
  .arbre {
    container-type: inline-size;
    background: var(--surface);
    border: 1px solid var(--line);
    border-radius: var(--cut) 0;
    corner-shape: bevel;
    overflow: clip; /* et non hidden : l'en-tête doit rester collé en haut au défilement */
    font-size: 13px;
  }
  .ligne {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 96px 96px 168px 88px 80px 100px 40px;
    align-items: center;
    min-height: 34px;
    padding-right: 6px;
    border-top: 1px solid var(--line-soft);
  }
  .ligne:hover:not(.entete) {
    background: var(--hover);
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
  .tri {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    max-width: 100%;
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
  .c-entete-nom {
    padding-left: 14px;
  }
  .entete .num {
    display: flex;
    justify-content: flex-end;
  }
  .entete .actif {
    color: var(--accent);
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
  .racine .num.actif {
    color: #ffffff;
  }

  .c-nom {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
    min-height: 34px;
    padding: 0 8px 0 calc(8px + var(--prof, 0) * 18px);
    background: none;
    border: none;
    color: var(--text);
    font: inherit;
    text-align: left;
  }
  .nom-btn {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 6px;
    align-self: stretch;
    padding: 0;
    border: none;
    background: none;
    color: inherit;
    font: inherit;
    text-align: left;
  }
  .pli {
    flex: none;
    display: flex;
    padding: 2px;
    border: none;
    background: none;
    color: var(--faint);
    transition: transform 0.15s;
  }
  .pli:hover {
    color: var(--text);
  }
  .ligne.choisie {
    background: color-mix(in srgb, var(--action) 10%, transparent);
    box-shadow: inset 3px 0 0 var(--action);
  }
  .ligne.choisie:hover {
    background: color-mix(in srgb, var(--action) 14%, transparent);
  }
  .pli.ouvert {
    transform: rotate(90deg);
    color: var(--accent);
  }
  .pli.invisible {
    visibility: hidden;
  }
  .type {
    flex: none;
    display: flex;
  }
  .t-dossier {
    color: var(--accent-2);
  }
  .racine .t-dossier {
    color: var(--action);
  }
  .t-fichiers,
  .t-fichier {
    color: var(--faint);
  }
  .t-reste {
    color: var(--dim);
  }
  .nom {
    min-width: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .racine .nom {
    font-weight: 600;
    color: #ffffff;
  }
  .italique {
    font-style: italic;
    color: var(--muted);
  }
  .attenue .nom {
    color: var(--faint);
  }
  .chip {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 2px 6px;
  }

  .c-pct {
    display: flex;
    align-items: center;
    gap: 8px;
    padding-right: 10px;
  }
  .barre {
    flex: 1;
    height: 8px;
    border-radius: 4px;
    background: var(--surface-sunken);
    border: 1px solid var(--line-soft);
    overflow: hidden;
  }
  .barre span {
    display: block;
    height: 100%;
    background: linear-gradient(90deg, var(--accent-2), var(--accent));
    box-shadow: 0 0 6px color-mix(in srgb, var(--accent) 50%, transparent);
  }
  .racine .barre span {
    background: linear-gradient(90deg, var(--action), #ffb15c);
    box-shadow: 0 0 6px color-mix(in srgb, var(--action) 50%, transparent);
  }
  .pct {
    flex: none;
    width: 52px;
    text-align: right;
    font-variant-numeric: tabular-nums;
    font-size: 12px;
    color: var(--muted);
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
  .info {
    min-height: 30px;
  }
  .info .c-nom {
    grid-column: 1 / -1;
    padding-left: calc(30px + var(--prof, 0) * 18px);
    font-size: 12px;
  }
  .sr {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
  }

  /* Fenêtre étroite : on garde l'essentiel (nom, taille du mode, %) */
  @container (max-width: 760px) {
    .ligne {
      grid-template-columns: minmax(0, 1fr) 92px 120px 36px;
    }
    /* « .ligne > » : plus spécifique que les cellules d'en-tête, qui sont en flex */
    .ligne > .c-fichiers,
    .ligne > .c-dossiers,
    .ligne > .c-modif,
    .ligne > .c-taille:not(.actif),
    .ligne > .c-disque:not(.actif) {
      display: none;
    }
  }
</style>
