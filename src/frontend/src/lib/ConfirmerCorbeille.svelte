<script>
  // Confirmation avant l'envoi à la corbeille : ce qui part, ce qui est
  // protégé, et les disques où Windows n'a souvent pas de corbeille.
  import { onMount } from 'svelte'
  import Icon from './Icon.svelte'
  import { app, call, message, apresMenage } from './state.svelte.js'
  import { octets, nombre } from './format.js'

  let { gen, mode, onfermer } = $props()

  let apercu = $state(null)
  let erreur = $state('')
  let compris = $state(false) // disque amovible : seconde confirmation
  let enCours = $state(false)
  let annuler

  const cibles = $state.snapshot(app.marques)
  const val = (x) => (mode === 'disque' ? x.disque : x.taille)

  onMount(() => {
    annuler?.focus() // le bouton sûr a le focus
    call('ApercuCorbeille', gen, cibles)
      .then((a) => (apercu = { ...a, elements: a.elements ?? [], proteges: a.proteges ?? [], amovibles: a.amovibles ?? [] }))
      .catch((err) => (erreur = message(err)))
  })

  const possible = $derived(apercu && apercu.nombre > 0 && (apercu.amovibles.length === 0 || compris) && !enCours)

  async function envoyer() {
    enCours = true
    erreur = ''
    try {
      const bilan = await call('Corbeille', gen, cibles)
      await apresMenage(bilan, 'corbeille')
      onfermer?.()
    } catch (err) {
      erreur = message(err)
    } finally {
      enCours = false
    }
  }

  function clavier(evt) {
    if (evt.key === 'Escape' && !enCours) {
      evt.preventDefault()
      onfermer?.()
    }
  }
</script>

<svelte:window onkeydown={clavier} />

<div class="voile" role="presentation" onclick={(e) => e.target === e.currentTarget && !enCours && onfermer?.()}>
  <div class="boite" role="alertdialog" aria-modal="true" aria-labelledby="titre-corbeille">
    <div class="tete">
      <span class="icone"><Icon name="trash" size={22} /></span>
      <h2 id="titre-corbeille">Envoyer à la corbeille ?</h2>
    </div>

    {#if erreur}
      <p class="error">{erreur}</p>
    {/if}

    {#if !apercu && !erreur}
      <p class="hint">Préparation…</p>
    {:else if apercu}
      {#if apercu.nombre > 0}
        <p class="resume">
          <strong>{nombre(apercu.nombre)} élément{apercu.nombre > 1 ? 's' : ''}</strong> · {nombre(apercu.fichiers)} fichier{apercu.fichiers > 1 ? 's' : ''}
          · <strong class="libere">{octets(val(apercu))}</strong> libérés
        </p>
        <ul class="liste">
          {#each apercu.elements as e (e.chemin)}
            <li>
              <Icon name={e.type === 'dossier' ? 'folder' : 'file'} size={14} />
              <span class="chemin" title={e.chemin}><bdi>{e.chemin}</bdi></span>
              <span class="taille">{octets(val(e))}</span>
            </li>
          {/each}
          {#if apercu.nombre > apercu.elements.length}
            <li class="hint">… et {nombre(apercu.nombre - apercu.elements.length)} autres, plus légers</li>
          {/if}
        </ul>
        {#if apercu.amovibles.length === 0}<p class="hint">Tout reste restaurable depuis la corbeille de Windows.</p>{/if}
      {:else}
        <p class="resume">Rien ne peut être envoyé à la corbeille dans cette sélection.</p>
      {/if}

      {#if apercu.proteges.length > 0}
        <div class="encart protege">
          <strong><Icon name="shield" size={14} /> {nombre(apercu.proteges.length)} élément{apercu.proteges.length > 1 ? 's protégés, qui ne seront pas touchés' : ' protégé, qui ne sera pas touché'}</strong>
          <ul>
            {#each apercu.proteges.slice(0, 5) as p (p.chemin)}<li><bdi>{p.chemin}</bdi> : {p.raison}</li>{/each}
            {#if apercu.proteges.length > 5}<li>… et {apercu.proteges.length - 5} autres</li>{/if}
          </ul>
        </div>
      {/if}

      {#if apercu.filtre}
        <div class="encart filtre">
          <Icon name="warn" size={14} /> Analyse filtrée : un dossier part en entier, y compris les fichiers que les filtres cachaient. Il peut donc peser plus lourd que ce qui est indiqué.
        </div>
      {/if}

      {#if apercu.amovibles.length > 0}
        <div class="encart danger">
          <strong><Icon name="warn" size={14} /> Disque amovible : {apercu.amovibles.join(', ')}</strong>
          <p>Sur une clé ou un disque USB, Windows n'a souvent pas de corbeille : les éléments seraient alors <strong>supprimés définitivement</strong>. Windows te préviendra avant de le faire.</p>
          <label><input type="checkbox" bind:checked={compris} /> J'ai compris, je veux continuer</label>
        </div>
      {/if}
    {/if}

    <div class="actions">
      <button class="btn-ghost" bind:this={annuler} disabled={enCours} onclick={() => onfermer?.()}>Annuler</button>
      <button class="btn-danger" disabled={!possible} onclick={envoyer}>
        <Icon name="trash" size={16} />
        {enCours ? 'Envoi en cours…' : `Envoyer ${apercu?.nombre > 1 ? `les ${nombre(apercu.nombre)} éléments` : "l'élément"} à la corbeille`}
      </button>
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
    width: min(720px, 100%);
    max-height: calc(100vh - 48px);
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 14px;
    padding: 22px;
    background: var(--surface);
    border: 1px solid color-mix(in srgb, var(--danger) 55%, transparent);
    border-radius: var(--cut) 0;
    corner-shape: bevel;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.6), 0 0 24px color-mix(in srgb, var(--danger) 18%, transparent);
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
    background: color-mix(in srgb, var(--danger) 14%, transparent);
    color: var(--danger);
  }
  h2 {
    margin: 0;
    font-family: var(--font-display);
    font-size: 22px;
    color: #ffffff;
  }
  .resume {
    margin: 0;
    font-size: 14px;
    color: var(--muted);
  }
  .resume strong {
    color: var(--text);
  }
  .resume .libere {
    color: var(--accent);
  }
  .liste {
    margin: 0;
    padding: 6px 0;
    list-style: none;
    max-height: 240px;
    overflow-y: auto;
    background: var(--surface-sunken);
    border: 1px solid var(--line);
    border-radius: 10px 0;
    corner-shape: bevel;
  }
  .liste li {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 4px 12px;
    font-size: 12px;
    color: var(--faint);
  }
  .chemin {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    direction: rtl;
    text-align: left;
    color: var(--text);
  }
  .taille {
    flex: none;
    font-variant-numeric: tabular-nums;
    color: var(--muted);
  }
  .encart {
    padding: 10px 12px;
    border-radius: 10px 0;
    corner-shape: bevel;
    font-size: 12px;
    line-height: 1.5;
  }
  .encart strong {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
  .encart ul {
    margin: 6px 0 0;
    padding-left: 18px;
    overflow-wrap: anywhere;
  }
  .encart p {
    margin: 6px 0;
  }
  .protege {
    border: 1px solid var(--line-strong);
    background: var(--surface-sunken);
    color: var(--muted);
  }
  .filtre {
    display: flex;
    gap: 8px;
    border: 1px solid color-mix(in srgb, var(--warn) 45%, transparent);
    background: color-mix(in srgb, var(--warn) 8%, transparent);
    color: var(--warn);
  }
  .danger {
    border: 1px solid color-mix(in srgb, var(--danger) 60%, transparent);
    background: color-mix(in srgb, var(--danger) 10%, transparent);
    color: var(--danger);
  }
  .danger label {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--text);
    cursor: pointer;
  }
  .danger input {
    width: auto;
    min-height: 0;
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
    flex-wrap: wrap;
  }
  .actions button {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }
  .btn-danger {
    min-height: 44px;
    padding: 0 16px;
    border-radius: 10px 0;
    corner-shape: bevel;
    border: 1px solid var(--danger);
    background: var(--danger);
    color: #1a0006;
    font-family: var(--font-mono);
    font-size: 13px;
    font-weight: 600;
  }
  .btn-danger:hover:not(:disabled) {
    background: #ff7a92;
  }
  .btn-danger:disabled {
    opacity: 0.4;
  }
</style>
