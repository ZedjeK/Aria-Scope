<script>
  // Export de la vue affichée. Le fichier est écrit par le moteur, à l'endroit
  // choisi dans « Enregistrer sous ».
  import { onMount } from 'svelte'
  import Icon from './Icon.svelte'
  import Portee from './Portee.svelte'
  import { app, call, message } from './state.svelte.js'

  let { gen, onfermer } = $props()

  const VUES = {
    arbre: { titre: "l'arborescence", detail: 'Un dossier par ligne, du plus lourd au plus léger, avec sa taille, sa part, ses fichiers et sa date.' },
    gros: { titre: 'les plus gros fichiers', detail: 'Les 1 000 plus gros fichiers relevés pendant le scan, avec leur chemin complet.' },
    types: { titre: 'la répartition par type', detail: 'Chaque famille, puis ses extensions.' },
    ages: { titre: "l'âge des fichiers", detail: 'La répartition par date de dernière modification.' },
    recherche: { titre: 'les résultats de la recherche', detail: 'Tous les résultats gardés, pas seulement ceux affichés.' },
    doublons: { titre: 'les doublons', detail: 'Une ligne par copie, regroupées par empreinte SHA-256.' },
  }
  const PROFONDEURS = [
    [1, '1 niveau'],
    [2, '2 niveaux'],
    [3, '3 niveaux'],
    [5, '5 niveaux'],
    [0, 'tous les dossiers'],
  ]

  const vue = app.vue
  let format = $state('csv') // csv ou html
  const RAPPORT = {
    titre: 'un rapport',
    detail:
      "Un seul fichier .html, aux couleurs d'Aria : résumé, carte, plus gros dossiers, types, âge, plus gros fichiers, et les doublons s'ils ont été cherchés. Il s'ouvre dans n'importe quel navigateur, s'envoie tel quel et s'imprime en PDF.",
  }
  const v = $derived(format === 'html' ? RAPPORT : (VUES[vue] ?? VUES.arbre))
  let profondeur = $state(3)
  let depart = $state(app.portee === 'selection' ? 'selection' : 'tout')
  let enCours = $state(false)
  let erreur = $state('')
  let fait = $state('') // chemin du fichier écrit
  let premier

  onMount(() => premier?.focus())

  const dans = $derived((vue === 'arbre' || format === 'html' ? depart : app.portee) === 'selection' && app.selection ? app.selection.id : 0)

  async function exporter() {
    enCours = true
    erreur = ''
    try {
      const options = { vue, mode: app.mode, tri: { colonne: '', inverse: false }, dans, profondeur }
      const chemin = await call(format === 'html' ? 'ExporterHTML' : 'ExporterCSV', gen, options)
      if (chemin) fait = chemin
    } catch (err) {
      erreur = message(err)
    } finally {
      enCours = false
    }
  }

  const montrer = () => call('MontrerChemin', fait, true).catch((err) => (erreur = message(err)))
  const ouvrir = () => call('OuvrirFichier', fait).catch((err) => (erreur = message(err)))

  function clavier(evt) {
    if (evt.key === 'Escape' && !enCours) {
      evt.preventDefault()
      onfermer?.()
    }
  }
</script>

<svelte:window onkeydown={clavier} />

<div class="voile" role="presentation" onclick={(e) => e.target === e.currentTarget && !enCours && onfermer?.()}>
  <div class="boite" role="dialog" aria-modal="true" aria-labelledby="titre-export">
    <div class="tete">
      <span class="icone"><Icon name="download" size={22} /></span>
      <h2 id="titre-export">Exporter {v.titre}</h2>
    </div>
    <p class="hint">{v.detail}</p>

    {#if !fait}
      <div class="champs">
        <div class="field">
          <span class="field-label">Format</span>
          <div class="formats" role="radiogroup" aria-label="Format">
            <button role="radio" aria-checked={format === 'csv'} class="format" class:on={format === 'csv'} bind:this={premier} onclick={() => (format = 'csv')}>
              <strong>CSV</strong><span>cette vue · Excel, scripts</span>
            </button>
            <button role="radio" aria-checked={format === 'html'} class="format" class:on={format === 'html'} onclick={() => (format = 'html')}>
              <strong>Rapport HTML</strong><span>tout · navigateur, PDF</span>
            </button>
          </div>
        </div>

        {#if format === 'html'}
          <div class="field">
            <span class="field-label">Sur</span>
            <Portee {gen} selection={app.selection} bind:portee={depart} />
          </div>
          <p class="hint">Pour un PDF : ouvre le rapport, bouton « Imprimer · PDF » (ou Ctrl+P), imprimante « Microsoft Print to PDF ».</p>
        {:else if vue === 'arbre'}
          <label class="field">
            <span class="field-label">Profondeur</span>
            <select bind:value={profondeur}>
              {#each PROFONDEURS as [p, texte] (p)}<option value={p}>{texte}</option>{/each}
            </select>
          </label>
          <div class="field">
            <span class="field-label">À partir de</span>
            <Portee {gen} selection={app.selection} bind:portee={depart} />
          </div>
        {:else if (vue === 'types' || vue === 'ages') && app.portee === 'selection'}
          <p class="hint">Portée de l'onglet : le dossier sélectionné.</p>
        {/if}
        {#if format === 'csv'}
          <p class="hint">Tailles {app.mode === 'disque' ? 'sur disque' : 'réelles'}, en octets et lisibles ; séparateur « ; » et dates reconnues par Excel en français.</p>
        {/if}
      </div>
    {:else}
      <div class="ok">
        <Icon name="check" size={16} />
        <span>Exporté dans <bdi>{fait}</bdi></span>
      </div>
    {/if}

    {#if erreur}<p class="error">{erreur}</p>{/if}

    <div class="actions">
      {#if fait}
        <button class="btn-ghost" onclick={montrer}><Icon name="folder" size={16} /> Montrer</button>
        {#if format === 'html'}<button class="btn-ghost" onclick={ouvrir}><Icon name="external" size={16} /> Ouvrir</button>{/if}
        <button class="btn-primary" onclick={() => onfermer?.()}>Fermer</button>
      {:else}
        <button class="btn-ghost" disabled={enCours} onclick={() => onfermer?.()}>Annuler</button>
        <button class="btn-primary" disabled={enCours} onclick={exporter}>
          <Icon name="download" size={16} />
          {enCours ? 'Export…' : 'Enregistrer sous…'}
        </button>
      {/if}
    </div>
  </div>
</div>

<style>
  .voile {
    position: fixed;
    inset: 0;
    z-index: 20;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
    background: rgba(1, 4, 10, 0.72);
    backdrop-filter: blur(3px);
  }
  .boite {
    width: min(560px, 100%);
    display: flex;
    flex-direction: column;
    gap: 14px;
    padding: 22px;
    background: var(--surface);
    border: 1px solid color-mix(in srgb, var(--accent) 50%, transparent);
    border-radius: var(--cut) 0;
    corner-shape: bevel;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.6), 0 0 24px color-mix(in srgb, var(--accent) 12%, transparent);
  }
  .tete {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .icone {
    display: flex;
    padding: 8px;
    border-radius: 8px 0;
    corner-shape: bevel;
    background: color-mix(in srgb, var(--accent) 14%, transparent);
    color: var(--accent);
  }
  h2 {
    margin: 0;
    font-family: var(--font-display);
    font-size: 22px;
    color: #ffffff;
  }
  .hint {
    margin: 0;
  }
  .champs {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }
  .formats {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }
  .format {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
    min-width: 150px;
    padding: 10px 14px;
    border: 1px solid var(--line-strong);
    border-radius: 10px 0;
    corner-shape: bevel;
    background: none;
    color: var(--muted);
    font-family: var(--font-mono);
    font-size: 11px;
    text-align: left;
  }
  .format strong {
    font-size: 14px;
    color: var(--text);
  }
  .format.on {
    border-color: color-mix(in srgb, var(--accent) 60%, transparent);
    background: color-mix(in srgb, var(--accent) 10%, transparent);
  }
  .format.on strong {
    color: var(--accent);
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
  .ok {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    padding: 10px 12px;
    border: 1px solid color-mix(in srgb, var(--accent) 45%, transparent);
    border-radius: 10px 0;
    corner-shape: bevel;
    background: color-mix(in srgb, var(--accent) 8%, transparent);
    color: var(--text);
    font-size: 13px;
    overflow-wrap: anywhere;
  }
  .ok :global(svg) {
    flex: none;
    color: var(--accent);
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
  }
  .actions button {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }
</style>
