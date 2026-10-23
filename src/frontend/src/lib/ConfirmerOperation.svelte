<script>
  // Confirmation avant une opération de ménage (corbeille, déplacement,
  // copie, archive) : ce qui est concerné, ce qui est écarté et pourquoi, la
  // place à l'arrivée, et les cas où Windows n'a pas de corbeille. L'archive
  // se suit ensuite ici même, jusqu'à la vérification.
  import { onMount, onDestroy } from 'svelte'
  import Icon from './Icon.svelte'
  import { app, call, message, apresMenage } from './state.svelte.js'
  import { octets, nombre } from './format.js'

  let { gen, mode, operation = 'corbeille', destination = '', onfermer } = $props()

  const TEXTES = {
    corbeille: { titre: 'Envoyer à la corbeille ?', icone: 'trash', verbe: 'Envoyer', suite: 'à la corbeille', encours: 'Envoi en cours…' },
    deplacer: { titre: 'Déplacer ?', icone: 'move', verbe: 'Déplacer', suite: '', encours: 'Déplacement en cours…' },
    copier: { titre: 'Copier ?', icone: 'copy', verbe: 'Copier', suite: '', encours: 'Copie en cours…' },
    archiver: { titre: 'Archiver en .zip ?', icone: 'archive', verbe: 'Archiver', suite: '', encours: 'Archivage en cours…' },
  }
  const PHASES = {
    ecriture: 'Écriture de l\'archive',
    verification: 'Vérification : relecture complète',
    corbeille: 'Envoi des originaux à la corbeille',
  }
  const t = $derived(TEXTES[operation] ?? TEXTES.corbeille)
  const archive = $derived(operation === 'archiver')

  let apercu = $state(null)
  let erreur = $state('')
  let compris = $state(false) // disque amovible : seconde confirmation
  let enCours = $state(false)
  let annuler

  // Archive
  let nom = $state('') // tel que tapé
  let erreurNom = $state('')
  let recycler = $state(false) // originaux à la corbeille après vérification
  let suivi = $state(null)
  let chargement = 0
  let minuterie
  let sondage

  const danger = $derived(operation === 'corbeille' || (archive && recycler))
  const cibles = $state.snapshot(app.marques)
  const val = (x) => (mode === 'disque' ? x.disque : x.taille)
  const pluriel = (n, mot) => `${nombre(n)} ${mot}${n > 1 ? 's' : ''}`
  const avecZip = (n) => (n.trim().toLowerCase().endsWith('.zip') ? n.trim() : n.trim() + '.zip')

  function recevoir(a) {
    apercu = {
      ...a,
      elements: a.elements ?? [],
      ecartes: a.ecartes ?? [],
      conflits: a.conflits ?? [],
      amovibles: a.amovibles ?? [],
      gardes: a.gardes ?? [],
    }
  }

  onMount(() => {
    annuler?.focus() // le bouton sûr a le focus
    const demande = archive ? call('ApercuArchive', gen, cibles, destination, '') : call('Apercu', gen, operation, cibles, destination)
    demande
      .then((a) => {
        recevoir(a)
        if (archive) nom = a.archive
      })
      .catch((err) => (erreur = message(err)))
  })

  onDestroy(() => {
    clearTimeout(minuterie)
    clearInterval(sondage)
  })

  // Le nom change : on revérifie (nom permis, déjà pris ?) après une courte pause.
  function nomChange() {
    clearTimeout(minuterie)
    const n = ++chargement
    minuterie = setTimeout(() => {
      call('ApercuArchive', gen, cibles, destination, nom)
        .then((a) => {
          if (n !== chargement) return
          recevoir(a)
          erreurNom = ''
        })
        .catch((err) => n === chargement && (erreurNom = message(err)))
    }, 250)
  }

  const nomPris = $derived(archive && apercu && !erreurNom && nom.trim() !== '' && apercu.archive.toLowerCase() !== avecZip(nom).toLowerCase())
  const amovibles = $derived(apercu && (!archive || recycler) ? apercu.amovibles : [])

  const possible = $derived(
    apercu &&
      apercu.nombre > 0 &&
      apercu.manque <= 0 &&
      !(archive && (erreurNom || nom.trim() === '')) &&
      (amovibles.length === 0 || compris) &&
      !enCours,
  )

  const libelleBouton = $derived.by(() => {
    if (enCours) return t.encours
    const n = apercu?.nombre ?? 0
    const quoi = n > 1 ? `les ${nombre(n)} éléments` : "l'élément"
    if (archive && recycler) return `Archiver puis recycler ${quoi}`
    return `${t.verbe} ${quoi}${t.suite ? ' ' + t.suite : ''}`
  })

  async function executer() {
    enCours = true
    erreur = ''
    if (archive) return archiver()
    try {
      const bilan = await call('Executer', gen, operation, cibles, destination)
      await apresMenage(bilan, operation)
      onfermer?.()
    } catch (err) {
      erreur = message(err)
    } finally {
      enCours = false
    }
  }

  async function archiver() {
    try {
      await call('Archiver', gen, cibles, destination, { nom, corbeille: recycler })
    } catch (err) {
      erreur = message(err)
      enCours = false
      return
    }
    sondage = setInterval(async () => {
      try {
        const s = await call('SuiviArchive')
        suivi = s
        if (s.phase === 'fini' && s.bilan) {
          clearInterval(sondage)
          await apresMenage(s.bilan, 'archiver')
          enCours = false
          onfermer?.()
        }
      } catch (err) {
        erreur = message(err)
      }
    }, 200)
  }

  const arreter = () => call('AnnulerArchive').catch((err) => (erreur = message(err)))
  const arretable = $derived(suivi && (suivi.phase === 'ecriture' || suivi.phase === 'verification'))
  const avancement = $derived(suivi && suivi.octets > 0 ? Math.min(1, suivi.octetsFaits / suivi.octets) : 0)

  function clavier(evt) {
    if (evt.key === 'Escape' && !enCours) {
      evt.preventDefault()
      onfermer?.()
    }
  }
</script>

<svelte:window onkeydown={clavier} />

<div class="voile" role="presentation" onclick={(e) => e.target === e.currentTarget && !enCours && onfermer?.()}>
  <div class="boite" class:danger-cadre={danger} role="alertdialog" aria-modal="true" aria-labelledby="titre-operation">
    <div class="tete">
      <span class="icone" class:icone-danger={danger}><Icon name={t.icone} size={22} /></span>
      <h2 id="titre-operation">{t.titre}</h2>
    </div>

    {#if destination}
      <p class="vers">
        <span class="eyebrow">{archive ? 'dans' : 'vers'}</span>
        <span class="chemin-dest" title={destination}><Icon name="folder" size={14} /> <bdi>{destination}</bdi></span>
      </p>
    {/if}

    {#if archive && apercu}
      <label class="champ">
        <span class="eyebrow">nom de l'archive</span>
        <input type="text" bind:value={nom} oninput={nomChange} disabled={enCours} spellcheck="false" aria-invalid={!!erreurNom} />
        {#if erreurNom}
          <span class="error">{erreurNom}</span>
        {:else if nomPris}
          <span class="hint">Ce nom est déjà pris : l'archive s'appellera <strong>{apercu.archive}</strong>. Rien n'est jamais écrasé.</span>
        {/if}
      </label>
    {/if}

    {#if erreur}
      <p class="error">{erreur}</p>
    {/if}

    {#if suivi && enCours}
      <div class="suivi" aria-live="polite">
        <div class="suivi-tete">
          <strong>{PHASES[suivi.phase] ?? 'Finalisation…'}</strong>
          {#if suivi.phase !== 'corbeille'}<span>{Math.floor(avancement * 100)} %</span>{/if}
        </div>
        {#if suivi.phase !== 'corbeille'}
          <div class="jauge" role="progressbar" aria-valuemin="0" aria-valuemax="100" aria-valuenow={Math.floor(avancement * 100)}>
            <span style="width: {avancement * 100}%"></span>
          </div>
          <p class="hint">
            {octets(suivi.octetsFaits)} sur {octets(suivi.octets)} · {nombre(suivi.fichiersFaits)} / {pluriel(suivi.fichiers, 'fichier')}
          </p>
        {:else}
          <p class="hint">Windows peut afficher sa propre fenêtre de progression.</p>
        {/if}
        {#if suivi.courant}<p class="courant" title={suivi.courant}><bdi>{suivi.courant}</bdi></p>{/if}
      </div>
    {:else if !apercu && !erreur}
      <p class="hint">Préparation…</p>
    {:else if apercu}
      {#if apercu.nombre > 0}
        <p class="resume">
          <strong>{pluriel(apercu.nombre, 'élément')}</strong> · {pluriel(apercu.fichiers, 'fichier')}
          · <strong class="libere">{octets(val(apercu))}</strong>{operation === 'corbeille' ? ' libérés' : ''}
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
        {#if operation === 'corbeille' && apercu.amovibles.length === 0}
          <p class="hint">Tout reste restaurable depuis la corbeille de Windows.</p>
        {:else if archive}
          <p class="hint">Format .zip standard, que l'Explorateur ouvre sans rien installer. L'archive est relue en entier avant d'être validée ; tu peux l'arrêter en route sans rien laisser derrière.</p>
        {:else if operation !== 'corbeille'}
          <p class="hint">Windows affiche sa progression et peut annuler l'opération depuis l'Explorateur (Ctrl+Z).</p>
        {/if}
      {:else}
        <p class="resume">Rien dans cette sélection ne peut être {archive ? 'archivé' : operation === 'copier' ? 'copié' : operation === 'deplacer' ? 'déplacé' : 'envoyé à la corbeille'}.</p>
      {/if}

      {#if apercu.besoin > 0 && apercu.libre >= 0}
        {#if apercu.manque > 0}
          <div class="encart danger">
            <strong><Icon name="warn" size={14} /> Pas assez de place à l'arrivée</strong>
            <p>Il faut {archive ? "jusqu'à " : ''}{octets(apercu.besoin)}, il reste {octets(apercu.libre)} : il manque <strong>{octets(apercu.manque)}</strong>.</p>
          </div>
        {:else}
          <p class="place"><Icon name="check" size={14} /> {archive ? "jusqu'à " : ''}{octets(apercu.besoin)} nécessaires · {octets(apercu.libre)} libres à l'arrivée</p>
        {/if}
      {/if}

      {#if archive && apercu.nombre > 0}
        <label class="option" class:option-active={recycler}>
          <input type="checkbox" bind:checked={recycler} disabled={enCours} />
          <span>
            <strong>Envoyer les originaux à la corbeille</strong> une fois l'archive vérifiée.
            <span class="hint">Seulement ce qui est entièrement dans l'archive et n'a pas changé entre-temps.</span>
          </span>
        </label>
        {#if recycler && apercu.gardes.length > 0}
          <div class="encart protege">
            <strong><Icon name="shield" size={14} /> {apercu.gardes.length > 1 ? `${nombre(apercu.gardes.length)} éléments archivés mais laissés en place` : '1 élément archivé mais laissé en place'}</strong>
            <ul>
              {#each apercu.gardes.slice(0, 5) as g (g.chemin)}<li><bdi>{g.chemin}</bdi> : {g.raison}</li>{/each}
              {#if apercu.gardes.length > 5}<li>… et {apercu.gardes.length - 5} autres</li>{/if}
            </ul>
          </div>
        {/if}
      {/if}

      {#if apercu.conflits.length > 0}
        <div class="encart filtre">
          <Icon name="warn" size={14} />
          <span>
            {pluriel(apercu.conflits.length, 'nom')} déjà pris à l'arrivée ({apercu.conflits.slice(0, 3).join(', ')}{apercu.conflits.length > 3 ? '…' : ''}) :
            Windows te demandera s'il faut remplacer ou garder les deux.
          </span>
        </div>
      {/if}

      {#if apercu.ecartes.length > 0}
        <div class="encart protege">
          <strong><Icon name="shield" size={14} /> {apercu.ecartes.length > 1 ? `${nombre(apercu.ecartes.length)} éléments écartés, qui ne seront pas touchés` : '1 élément écarté, qui ne sera pas touché'}</strong>
          <ul>
            {#each apercu.ecartes.slice(0, 5) as p (p.chemin)}<li><bdi>{p.chemin}</bdi> : {p.raison}</li>{/each}
            {#if apercu.ecartes.length > 5}<li>… et {apercu.ecartes.length - 5} autres</li>{/if}
          </ul>
        </div>
      {/if}

      {#if apercu.dansAnalyse && apercu.nombre > 0}
        <div class="encart info">
          <Icon name="scope" size={14} /> L'arrivée fait partie de l'analyse : ce qui y arrive n'apparaîtra qu'après une nouvelle analyse.
        </div>
      {/if}

      {#if apercu.filtre}
        <div class="encart filtre">
          <Icon name="warn" size={14} /> Analyse filtrée : un dossier est traité en entier, y compris les fichiers que les filtres cachaient. Il peut donc peser plus lourd que ce qui est indiqué.
        </div>
      {/if}

      {#if amovibles.length > 0}
        <div class="encart danger">
          <strong><Icon name="warn" size={14} /> Disque amovible : {amovibles.join(', ')}</strong>
          <p>Sur une clé ou un disque USB, Windows n'a souvent pas de corbeille : les {archive ? 'originaux' : 'éléments'} seraient alors <strong>supprimés définitivement</strong>. Windows te préviendra avant de le faire.</p>
          <label><input type="checkbox" bind:checked={compris} /> J'ai compris, je veux continuer</label>
        </div>
      {/if}
    {/if}

    <div class="actions">
      {#if archive && enCours}
        <button class="btn-ghost" bind:this={annuler} disabled={!arretable} onclick={arreter}>
          <Icon name="stop" size={16} /> Arrêter
        </button>
      {:else}
        <button class="btn-ghost" bind:this={annuler} disabled={enCours} onclick={() => onfermer?.()}>Annuler</button>
      {/if}
      <button class={danger ? 'btn-danger' : 'btn-primary'} disabled={!possible} onclick={executer}>
        <Icon name={t.icone} size={16} />
        {libelleBouton}
      </button>
    </div>
  </div>
</div>

<style>
  .champ {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-top: -4px;
  }
  .champ input {
    font-family: var(--font-mono);
    font-size: 13px;
  }
  .champ strong {
    color: var(--text);
  }
  .option {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 10px 12px;
    border: 1px solid var(--line-strong);
    border-radius: 10px 0;
    corner-shape: bevel;
    font-size: 13px;
    color: var(--muted);
    cursor: pointer;
  }
  .option strong {
    color: var(--text);
  }
  .option .hint {
    display: block;
    margin-top: 2px;
    font-size: 12px;
  }
  .option input {
    width: auto;
    min-height: 0;
    margin-top: 3px;
  }
  .option.option-active {
    border-color: color-mix(in srgb, var(--danger) 60%, transparent);
    background: color-mix(in srgb, var(--danger) 8%, transparent);
  }
  .suivi {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 14px;
    border: 1px solid color-mix(in srgb, var(--action) 45%, transparent);
    border-radius: 10px 0;
    corner-shape: bevel;
    background: var(--surface-sunken);
  }
  .suivi-tete {
    display: flex;
    justify-content: space-between;
    font-size: 13px;
    color: var(--text);
    font-variant-numeric: tabular-nums;
  }
  .jauge {
    height: 8px;
    border-radius: 4px;
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
  .suivi .hint {
    margin: 0;
  }
  .courant {
    margin: 0;
    font-size: 11px;
    color: var(--dim);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    direction: rtl;
    text-align: left;
  }
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
    border: 1px solid color-mix(in srgb, var(--action) 55%, transparent);
    border-radius: var(--cut) 0;
    corner-shape: bevel;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.6), 0 0 24px color-mix(in srgb, var(--action) 14%, transparent);
  }
  .boite.danger-cadre {
    border-color: color-mix(in srgb, var(--danger) 55%, transparent);
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
    background: color-mix(in srgb, var(--action) 14%, transparent);
    color: var(--action);
  }
  .icone.icone-danger {
    background: color-mix(in srgb, var(--danger) 14%, transparent);
    color: var(--danger);
  }
  .vers {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin: -4px 0 0;
  }
  .chemin-dest {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
    padding: 8px 10px;
    border: 1px solid var(--line-strong);
    border-radius: 8px 0;
    corner-shape: bevel;
    background: var(--surface-sunken);
    color: var(--accent);
    font-size: 13px;
    overflow-wrap: anywhere;
  }
  .place {
    display: flex;
    align-items: center;
    gap: 6px;
    margin: 0;
    font-size: 12px;
    color: var(--accent);
  }
  .info {
    display: flex;
    gap: 8px;
    border: 1px solid color-mix(in srgb, var(--accent-2) 45%, transparent);
    background: color-mix(in srgb, var(--accent-2) 8%, transparent);
    color: var(--ghost);
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
    align-items: flex-start;
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
