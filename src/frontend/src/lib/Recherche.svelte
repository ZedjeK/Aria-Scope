<script>
  // Recherche multicritère dans l'analyse : les dossiers connus sont relus
  // en parallèle. Les résultats se sélectionnent comme partout (Ctrl+clic,
  // Maj+clic) et passent par la barre de ménage.
  import { onMount, onDestroy } from 'svelte'
  import Icon from './Icon.svelte'
  import Portee from './Portee.svelte'
  import { app, call, message, montrer, cliquerElement } from './state.svelte.js'
  import { octets, nombre, dateCourte, dateLongue } from './format.js'
  import { estLigneChoisie, marquable, cibleDe, cleMarque } from './selection.js'

  let { gen, mode, selection = $bindable(null) } = $props()

  const UNITES = { Ko: 1024, Mo: 1024 ** 2, Go: 1024 ** 3 }
  const PAS = 500 // lignes affichées à la fois

  const f = app.recherche // brouillon partagé, gardé d'un onglet à l'autre
  let familles = $state([])
  let suivi = $state(null) // dernière recherche sur cette analyse
  let lignes = $state(null)
  let erreur = $state('')
  let tri = $state({ colonne: '', inverse: false })
  let visibles = $state(PAS)
  let typesOuverts = $state(false)
  let sondage

  onMount(async () => {
    try {
      familles = (await call('Familles')) ?? []
      await suivre()
    } catch (err) {
      erreur = message(err)
    }
  })
  onDestroy(() => clearInterval(sondage))

  const octetsDe = (v, u) => (v === '' || v == null ? 0 : Math.round(Number(v) * UNITES[u]))
  const jour = (texte, fin) => {
    if (!texte) return 0
    const [a, m, j] = texte.split('-').map(Number)
    const d = fin ? new Date(a, m - 1, j, 23, 59, 59) : new Date(a, m - 1, j)
    return Math.floor(d.getTime() / 1000)
  }

  const criteres = $derived({
    nom: f.nom,
    regex: f.regex,
    quoi: f.quoi,
    tailleMin: octetsDe(f.min, f.minUnite),
    tailleMax: octetsDe(f.max, f.maxUnite),
    modifApres: jour(f.apres, false),
    modifAvant: jour(f.avant, true),
    familles: f.familles,
    cheminMin: Number(f.cheminMin) || 0,
  })

  const enCours = $derived(suivi?.etat === 'scan')

  async function chercher() {
    erreur = ''
    const dans = f.portee === 'selection' && selection ? selection.id : 0
    try {
      await call('Chercher', gen, dans, $state.snapshot(criteres))
      lignes = null
      visibles = PAS
      await suivre()
    } catch (err) {
      erreur = message(err)
    }
  }

  // Suit la recherche jusqu'au bout, puis charge les résultats.
  async function suivre() {
    clearInterval(sondage)
    const lire = async () => {
      const s = await call('SuiviRecherche')
      if (!s?.gen || s.gen !== gen) {
        suivi = null
        clearInterval(sondage)
        return
      }
      suivi = s
      if (s.etat !== 'scan') {
        clearInterval(sondage)
        await charger()
      }
    }
    await lire()
    if (suivi?.etat === 'scan') sondage = setInterval(() => lire().catch((err) => (erreur = message(err))), 250)
  }

  async function charger() {
    try {
      lignes = (await call('ResultatsRecherche', gen, mode, $state.snapshot(tri))) ?? []
    } catch (err) {
      erreur = message(err)
    }
  }

  // Tri, mode de taille, ménage : on recharge.
  let premier = true
  $effect(() => {
    tri.colonne
    tri.inverse
    mode
    app.revision
    if (premier) {
      premier = false
      return
    }
    if (suivi && suivi.etat !== 'scan') charger()
  })

  const arreter = () => call('ArreterRecherche').catch((err) => (erreur = message(err)))

  // Limite historique de Windows (MAX_PATH) : 260 caractères, caractère de
  // fin compris ; un chemin de 260 caractères ou plus pose problème.
  const LIMITE = 260

  function cheminsLongs() {
    Object.assign(f, { nom: '', regex: false, quoi: 'tout', min: '', max: '', apres: '', avant: '', familles: [], cheminMin: LIMITE })
    chercher()
  }

  function effacer() {
    Object.assign(f, { nom: '', regex: false, quoi: 'fichiers', min: '', max: '', apres: '', avant: '', familles: [], cheminMin: '' })
  }

  function basculerFamille(nom) {
    const i = f.familles.indexOf(nom)
    if (i >= 0) f.familles.splice(i, 1)
    else f.familles.push(nom)
  }

  const effective = $derived(tri.colonne || (mode === 'disque' ? 'disque' : 'taille'))
  const decroissant = $derived((effective !== 'nom') !== tri.inverse)
  function trierPar(colonne) {
    const cible = colonne || (mode === 'disque' ? 'disque' : 'taille')
    if (cible === effective && colonne === tri.colonne) tri = { colonne, inverse: !tri.inverse }
    else tri = { colonne, inverse: false }
  }

  const affichees = $derived((lignes ?? []).slice(0, visibles))
  // Ensemble des éléments sélectionnés : rapide même avec des milliers de lignes.
  const marquees = $derived(new Set(app.marques.map(cleMarque)))
  const selectionnables = $derived((lignes ?? []).filter(marquable))
  const toutChoisi = $derived(selectionnables.length > 0 && selectionnables.every((x) => marquees.has(cleMarque(cibleDe(x)))))
  function toutSelectionner() {
    app.marques = toutChoisi ? [] : selectionnables.map(cibleDe)
    app.ancre = null
  }
  // Total sans double compte : ce qui se trouve dans un dossier déjà listé
  // est compris dans la taille de ce dossier.
  const totalListe = $derived.by(() => {
    const l = lignes ?? []
    const sep = (d) => (d.includes('\\') ? '\\' : '/')
    const complet = (x) => (x.dossier ? x.dossier.replace(/[\\/]+$/, '') + sep(x.dossier) + x.nom : x.nom)
    const dossiers = new Set(l.filter((x) => x.type === 'dossier').map(complet))
    let t = 0
    for (const x of l) {
      let d = x.dossier ?? ''
      let dedans = false
      while (d && !dedans) {
        dedans = dossiers.has(d.replace(/[\\/]+$/, ''))
        const i = Math.max(d.lastIndexOf('/'), d.lastIndexOf('\\'))
        d = i > 0 ? d.slice(0, i) : ''
      }
      if (!dedans) t += mode === 'disque' ? x.disque : x.taille
    }
    return t
  })
  const avecLongueur = $derived((suivi?.criteres?.cheminMin ?? 0) > 0)
  const avance = $derived(suivi && suivi.dossiers > 0 ? Math.min(1, suivi.lus / suivi.dossiers) : 0)
  const montrerLigne = (x) => (x.type === 'fichier' ? montrer(gen, x.id, x.nom) : montrer(gen, x.id))
</script>

<form class="card criteres" onsubmit={(e) => (e.preventDefault(), chercher())}>
  <div class="rangee">
    <label class="field nom">
      <span class="field-label">Nom</span>
      <input bind:value={f.nom} placeholder={f.regex ? '^IMG_\\d{4}' : '*.mkv ; IMG_* ; facture'} spellcheck="false" />
    </label>
    <label class="coche" title="Le nom est une expression régulière">
      <input type="checkbox" bind:checked={f.regex} /> regex
    </label>
    <label class="field quoi">
      <span class="field-label">Chercher</span>
      <select bind:value={f.quoi}>
        <option value="fichiers">des fichiers</option>
        <option value="dossiers">des dossiers</option>
        <option value="tout">les deux</option>
      </select>
    </label>
  </div>
  {#if !f.regex}<span class="hint aide">Sans joker, il suffit que le nom contienne le texte. Plusieurs motifs : sépare-les par « ; ».</span>{/if}

  <div class="rangee grille">
    <div class="field">
      <span class="field-label">Taille</span>
      <div class="duo">
        <input type="number" min="0" step="any" bind:value={f.min} placeholder="min" aria-label="Taille minimale" />
        <select bind:value={f.minUnite} aria-label="Unité de la taille minimale">{#each Object.keys(UNITES) as u (u)}<option>{u}</option>{/each}</select>
        <span class="a">à</span>
        <input type="number" min="0" step="any" bind:value={f.max} placeholder="max" aria-label="Taille maximale" />
        <select bind:value={f.maxUnite} aria-label="Unité de la taille maximale">{#each Object.keys(UNITES) as u (u)}<option>{u}</option>{/each}</select>
      </div>
    </div>
    <div class="field">
      <span class="field-label">Modifié entre</span>
      <div class="duo">
        <input type="date" bind:value={f.apres} aria-label="Modifié après le" />
        <span class="a">et</span>
        <input type="date" bind:value={f.avant} aria-label="Modifié avant le" />
      </div>
    </div>
    <label class="field">
      <span class="field-label">Chemin d'au moins</span>
      <div class="duo">
        <input type="number" min="0" step="1" bind:value={f.cheminMin} placeholder="—" aria-label="Longueur minimale du chemin" />
        <span class="a">caractères</span>
      </div>
    </label>
  </div>

  {#if f.quoi !== 'dossiers' && familles.length > 0}
    <div class="field">
      <span class="field-label">
        Types
        <button type="button" class="deplier" aria-expanded={typesOuverts} onclick={() => (typesOuverts = !typesOuverts)}>
          {f.familles.length === 0 ? 'tous' : f.familles.join(', ')} <span aria-hidden="true">{typesOuverts ? '▲' : '▼'}</span>
        </button>
        {#if f.familles.length > 0}<button type="button" class="lien" onclick={() => (f.familles = [])}>effacer</button>{/if}
      </span>
      {#if typesOuverts}
        <div class="familles">
          {#each familles as nom (nom)}
            <button type="button" class="famille" class:on={f.familles.includes(nom)} aria-pressed={f.familles.includes(nom)} onclick={() => basculerFamille(nom)}>{nom}</button>
          {/each}
        </div>
      {/if}
    </div>
  {/if}

  <div class="pied">
    <Portee {gen} {selection} bind:portee={f.portee} />
    <div class="actions">
      <button type="button" class="btn-ghost" onclick={cheminsLongs} disabled={enCours} title="Fichiers et dossiers dont le chemin atteint {LIMITE} caractères">
        <Icon name="warn" size={15} /> Chemins trop longs
      </button>
      <button type="button" class="btn-ghost" onclick={effacer} disabled={enCours}>Effacer</button>
      {#if enCours}
        <button type="button" class="btn-primary" onclick={arreter}><Icon name="stop" size={16} /> Arrêter</button>
      {:else}
        <button type="submit" class="btn-primary"><Icon name="folder-search" size={16} /> Chercher</button>
      {/if}
    </div>
  </div>
</form>

{#if erreur}
  <p class="error">{erreur}</p>
{/if}

{#if suivi}
  <div class="etat" aria-live="polite">
    {#if enCours}
      <div class="jauge"><span style="width: {avance * 100}%"></span></div>
      <span class="hint">{nombre(suivi.lus)} / {nombre(suivi.dossiers)} dossiers relus · {nombre(suivi.trouves)} trouvé{suivi.trouves > 1 ? 's' : ''}</span>
    {:else}
      <p class="hint resume">
        {#if suivi.gardes < suivi.trouves}
          <strong>{nombre(suivi.trouves)}</strong> trouvés · les <strong>{nombre(lignes?.length ?? 0)}</strong> plus lourds sont listés · <strong>{octets(totalListe)}</strong>
        {:else}
          <strong>{nombre(lignes?.length ?? 0)}</strong> résultat{(lignes?.length ?? 0) > 1 ? 's' : ''} · <strong>{octets(totalListe)}</strong>
        {/if}
        {#if suivi.etat === 'arrete'}· recherche arrêtée avant la fin{/if}
        {#if selectionnables.length > 1}
          <button class="lien tout" onclick={toutSelectionner}>{toutChoisi ? 'tout désélectionner' : `tout sélectionner (${nombre(selectionnables.length)})`}</button>
        {/if}
        {#if suivi.illisibles > 0}· {nombre(suivi.illisibles)} dossier{suivi.illisibles > 1 ? 's' : ''} illisible{suivi.illisibles > 1 ? 's' : ''}{/if}
      </p>
    {/if}
  </div>
{/if}

{#if suivi && !enCours && lignes && (suivi.criteres?.cheminMin ?? 0) >= LIMITE}
  {#if lignes.length > 0}
    <div class="encart" role="note">
      <strong><Icon name="warn" size={14} /> Chemins de {LIMITE} caractères ou plus</strong>
      <p>
        Windows limite historiquement un chemin à {LIMITE} caractères, caractère de fin compris. Depuis Windows 10, les logiciels prévus pour peuvent
        dépasser cette limite, mais beaucoup d'autres non : l'Explorateur, certains outils de sauvegarde ou de synchronisation, de vieux programmes
        n'arrivent plus à copier, ouvrir ou supprimer ces éléments.
      </p>
      <p>
        Le ménage d'Aria passe par l'Explorateur, qui peut lui aussi les refuser. Le plus simple : <strong>raccourcir le chemin par le haut</strong>,
        en renommant un dossier parent ou en déplaçant la branche plus près de la racine du disque. La colonne « Chemin » donne la longueur de chacun.
      </p>
    </div>
  {:else}
    <p class="hint ok"><Icon name="check" size={14} /> Aucun chemin de {LIMITE} caractères ou plus : rien à craindre de ce côté.</p>
  {/if}
{/if}

{#if lignes && lignes.length > 0}
  <div class="liste" class:avec-longueur={avecLongueur} role="grid" aria-label="Résultats de la recherche">
    <div class="ligne entete" role="row">
      {#snippet entete(colonne, texte, classe = '')}
        {@const actifTri = colonne === tri.colonne}
        <span role="columnheader" class={classe} aria-sort={actifTri ? (decroissant ? 'descending' : 'ascending') : undefined}>
          <button class="tri" class:trie={actifTri} onclick={() => trierPar(colonne)}>
            {texte}<span class="fleche" aria-hidden="true">{actifTri ? (decroissant ? '▼' : '▲') : ''}</span>
          </button>
        </span>
      {/snippet}
      {@render entete('nom', 'Nom', 'c-nom')}
      {@render entete('taille', 'Taille', `num c-taille${mode === 'taille' ? ' actif' : ''}`)}
      {@render entete('disque', 'Sur disque', `num c-disque${mode === 'disque' ? ' actif' : ''}`)}
      {@render entete('modif', 'Modifié', 'num c-modif')}
      {#if avecLongueur}<span role="columnheader" class="num c-long">Chemin</span>{/if}
      <span role="columnheader" class="c-dossier">Dossier</span>
      <span role="columnheader" class="c-act"><span class="sr">Actions</span></span>
    </div>

    {#each affichees as x (`${x.type}/${x.id}/${x.nom}`)}
      {@const choisie = (marquable(x) && marquees.has(cleMarque(cibleDe(x)))) || estLigneChoisie(selection, x)}
      <div
        class="ligne"
        class:choisie
        role="row"
        tabindex="0"
        aria-selected={choisie}
        onclick={(evt) => cliquerElement(x, 'liste', evt, affichees)}
        ondblclick={() => montrerLigne(x)}
        onkeydown={(evt) => evt.key === 'Enter' && cliquerElement(x, 'liste', evt, affichees)}
      >
        <span class="c-nom" role="gridcell" title={x.nom}><Icon name={x.type === 'dossier' ? 'folder' : 'file'} size={15} /><span class="nom">{x.nom}</span></span>
        <span class="num c-taille" role="gridcell" class:actif={mode === 'taille'}>{octets(x.taille)}</span>
        <span class="num c-disque" role="gridcell" class:actif={mode === 'disque'}>{octets(x.disque)}</span>
        <span class="num c-modif" role="gridcell" title="Modifié le {dateLongue(x.modif)}">{dateCourte(x.modif)}</span>
        {#if avecLongueur}<span class="num c-long" role="gridcell" class:trop={x.longueur >= 260}>{nombre(x.longueur)}</span>{/if}
        <span class="c-dossier" role="gridcell" title={x.dossier}><bdi>{x.dossier}</bdi></span>
        <span class="c-act" role="gridcell">
          <button class="ouvrir" title="Montrer dans l'Explorateur" aria-label="Montrer dans l'Explorateur" onclick={(e) => (e.stopPropagation(), montrerLigne(x))}>
            <Icon name="external" size={15} />
          </button>
        </span>
      </div>
    {/each}
  </div>
  {#if lignes.length > visibles}
    <button class="btn-ghost plus" onclick={() => (visibles += PAS)}>Afficher {nombre(Math.min(PAS, lignes.length - visibles))} de plus ({nombre(lignes.length - visibles)} restants)</button>
  {/if}
{:else if lignes && suivi && !enCours && (suivi.criteres?.cheminMin ?? 0) < LIMITE}
  <p class="hint">Rien ne correspond à ces critères.</p>
{/if}

<style>
  .criteres {
    gap: 12px;
  }
  .rangee {
    display: flex;
    align-items: flex-end;
    gap: 12px;
    flex-wrap: wrap;
  }
  .nom {
    flex: 1 1 260px;
  }
  .quoi {
    flex: 0 0 160px;
  }
  .grille {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
    align-items: start;
  }
  .duo {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .duo input {
    flex: 1 1 0;
    min-width: 0;
  }
  .duo select {
    flex: none;
    width: auto;
  }
  .a {
    flex: none;
    font-size: 12px;
    color: var(--faint);
  }
  .aide {
    margin-top: -6px;
  }
  .coche {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    min-height: 40px;
    font-size: 13px;
    color: var(--muted);
    cursor: pointer;
  }
  .coche input {
    width: auto;
    min-height: 0;
  }
  select {
    width: 100%;
    min-height: 40px;
    padding: 0 10px;
    border-radius: 8px 0;
    corner-shape: bevel;
    border: 1px solid var(--line-strong);
    background: var(--surface-sunken);
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 13px;
  }
  input[type='date'] {
    color-scheme: dark;
  }
  .familles {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
  .famille {
    min-height: 28px;
    padding: 0 9px;
    border: 1px solid var(--line-strong);
    border-radius: 8px 0;
    corner-shape: bevel;
    background: none;
    color: var(--muted);
    font-family: var(--font-mono);
    font-size: 12px;
  }
  .famille:hover {
    color: var(--text);
  }
  .famille.on {
    border-color: color-mix(in srgb, var(--accent) 60%, transparent);
    background: color-mix(in srgb, var(--accent) 12%, transparent);
    color: var(--accent);
  }
  .tout {
    margin-left: 6px;
    color: var(--action);
  }
  .deplier {
    margin-left: 8px;
    padding: 0;
    border: none;
    background: none;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 12px;
    font-weight: 400;
    text-align: left;
  }
  .lien {
    margin-left: 8px;
    padding: 0;
    border: none;
    background: none;
    color: var(--accent);
    font-family: var(--font-mono);
    font-size: 12px;
  }
  .pied {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    flex-wrap: wrap;
  }
  .actions {
    display: flex;
    gap: 8px;
  }
  .actions button {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }
  .etat {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .jauge {
    height: 6px;
    border-radius: 3px;
    background: var(--line);
    overflow: hidden;
  }
  .jauge span {
    display: block;
    height: 100%;
    background: var(--action);
    box-shadow: 0 0 10px var(--action);
    transition: width 0.2s linear;
  }
  .resume {
    margin: 0;
  }
  .resume strong {
    color: var(--text);
  }
  .plus {
    align-self: center;
  }
  .encart {
    padding: 10px 14px;
    border: 1px solid color-mix(in srgb, var(--warn) 45%, transparent);
    border-radius: 10px 0;
    corner-shape: bevel;
    background: color-mix(in srgb, var(--warn) 7%, transparent);
    color: var(--muted);
    font-size: 12px;
    line-height: 1.55;
  }
  .encart > strong {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    color: var(--warn);
  }
  .encart p {
    margin: 6px 0 0;
  }
  .encart p strong {
    color: var(--text);
  }
  .ok {
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--accent);
  }

  /* Liste des résultats : comme celle des plus gros fichiers */
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
    grid-template-columns: minmax(160px, 1.2fr) 92px 92px 100px minmax(0, 1fr) 40px;
    align-items: center;
    min-height: 32px;
    padding-left: 6px;
    padding-right: 6px;
    border-top: 1px solid var(--line-soft);
    cursor: default;
    user-select: none;
  }
  .avec-longueur .ligne {
    grid-template-columns: minmax(160px, 1.2fr) 92px 92px 100px 76px minmax(0, 1fr) 40px;
  }
  .ligne:hover:not(.entete) {
    background: var(--hover);
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
  .c-long.trop {
    color: var(--warn);
    font-weight: 600;
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
  @container (max-width: 760px) {
    .ligne,
    .avec-longueur .ligne {
      grid-template-columns: minmax(0, 1fr) 88px minmax(0, 0.8fr) 36px;
    }
    .ligne > .c-modif,
    .ligne > .c-long,
    .ligne > .c-taille:not(.actif),
    .ligne > .c-disque:not(.actif) {
      display: none;
    }
  }
</style>
