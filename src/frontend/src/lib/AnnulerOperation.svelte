<script>
  // Confirmation avant d'annuler une opération du journal : ce qui revient
  // à sa place, ce qui part à la corbeille, ce qui est écarté et pourquoi.
  import { onMount } from 'svelte'
  import Icon from './Icon.svelte'
  import { call, message } from './state.svelte.js'
  import { octets, nombre } from './format.js'

  let { entree, onfermer, onfait } = $props()

  let apercu = $state(null)
  let erreur = $state('')
  let enCours = $state(false)
  let annuler

  const pluriel = (n, mot) => `${nombre(n)} ${mot}${n > 1 ? 's' : ''}`

  onMount(() => {
    annuler?.focus()
    call('ApercuAnnulation', entree.id)
      .then((a) => (apercu = { ...a, lignes: a.lignes ?? [], ecartes: a.ecartes ?? [] }))
      .catch((err) => (erreur = message(err)))
  })

  const textes = $derived.by(() => {
    const n = apercu?.lignes.length ?? 0
    switch (entree.operation) {
      case 'deplacer':
        return { titre: 'Remettre en place ?', icone: 'move', bouton: n > 1 ? `Remettre les ${nombre(n)} éléments en place` : "Remettre l'élément en place", encours: 'Remise en place…' }
      case 'copier':
        return { titre: n === 1 ? 'Envoyer la copie à la corbeille ?' : 'Envoyer les copies à la corbeille ?', icone: 'copy', bouton: n > 1 ? `Envoyer les ${nombre(n)} copies à la corbeille` : 'Envoyer la copie à la corbeille', encours: 'Envoi en cours…' }
      default:
        if (n === 0) return { titre: "Envoyer l'archive à la corbeille ?", icone: 'archive', bouton: "Envoyer l'archive à la corbeille", encours: 'Envoi en cours…' }
        return { titre: "Restaurer depuis l'archive ?", icone: 'archive', bouton: n > 1 ? `Restaurer les ${nombre(n)} originaux` : "Restaurer l'original", encours: 'Restauration…' }
    }
  })

  const possible = $derived(apercu && !apercu.impossible && !enCours)

  async function executer() {
    enCours = true
    erreur = ''
    try {
      const b = await call('AnnulerOperation', entree.id)
      onfait?.({ ...b, restants: b.restants ?? [] })
    } catch (err) {
      erreur = message(err)
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
  <div class="boite" role="alertdialog" aria-modal="true" aria-labelledby="titre-annulation">
    <div class="tete">
      <span class="icone"><Icon name="history" size={22} /></span>
      <h2 id="titre-annulation">{textes.titre}</h2>
    </div>

    {#if erreur}
      <p class="error">{erreur}</p>
    {/if}

    {#if !apercu && !erreur}
      <p class="hint">Vérification de ce qui peut être annulé…</p>
    {:else if apercu}
      {#if apercu.impossible}
        <div class="encart protege">
          <strong><Icon name="shield" size={14} /> Impossible d'annuler</strong>
          <p>{apercu.impossible[0].toUpperCase() + apercu.impossible.slice(1)}.</p>
        </div>
      {:else}
        {#if apercu.lignes.length > 0}
          <p class="resume">
            <strong>{pluriel(apercu.lignes.length, 'élément')}</strong> · <strong class="accent">{octets(apercu.taille)}</strong>
          </p>
          <ul class="liste">
            {#each apercu.lignes.slice(0, 100) as l (l.chemin)}
              <li>
                <Icon name={l.type === 'dossier' ? 'folder' : 'file'} size={14} />
                <span class="chemins">
                  <span class="de" title={l.chemin}><bdi>{l.chemin}</bdi></span>
                  <span class="vers" title={l.vers}><span class="fleche">→</span><span class="coupe">{#if l.vers}<bdi>{l.vers}</bdi>{:else}corbeille{/if}</span></span>
                </span>
                <span class="taille">{octets(l.taille)}</span>
              </li>
            {/each}
            {#if apercu.lignes.length > 100}<li class="hint">… et {nombre(apercu.lignes.length - 100)} autres</li>{/if}
          </ul>
        {/if}

        {#if entree.operation === 'deplacer'}
          <p class="hint">Chaque élément retourne dans son dossier d'origine, recréé s'il le faut. Rien n'est écrasé.</p>
        {:else if entree.operation === 'copier'}
          <p class="hint">Les originaux ne bougent pas. Les copies restent restaurables depuis la corbeille de Windows.</p>
        {:else if apercu.lignes.length > 0}
          <p class="hint">Les originaux sont extraits de l'archive, contrôlés, puis remis à leur place, avec leurs dates. Rien n'est écrasé.</p>
        {/if}

        {#if entree.operation === 'archiver'}
          {#if apercu.archiveRecyclee}
            <p class="place"><Icon name="archive" size={14} /> Ensuite, l'archive part à la corbeille : <bdi>{apercu.archive}</bdi></p>
          {:else if apercu.archiveGardee}
            <div class="encart filtre"><Icon name="warn" size={14} /> {apercu.archiveGardee[0].toUpperCase() + apercu.archiveGardee.slice(1)}.</div>
          {/if}
        {/if}
      {/if}

      {#if apercu.ecartes.length > 0}
        <div class="encart protege">
          <strong><Icon name="shield" size={14} /> {apercu.ecartes.length > 1 ? `${nombre(apercu.ecartes.length)} éléments laissés de côté` : '1 élément laissé de côté'}</strong>
          <ul>
            {#each apercu.ecartes.slice(0, 8) as p (p.chemin)}<li><bdi>{p.chemin}</bdi> : {p.raison}</li>{/each}
            {#if apercu.ecartes.length > 8}<li>… et {apercu.ecartes.length - 8} autres</li>{/if}
          </ul>
        </div>
      {/if}
    {/if}

    <div class="actions">
      <button class="btn-ghost" bind:this={annuler} disabled={enCours} onclick={() => onfermer?.()}>{apercu?.impossible ? 'Fermer' : 'Ne rien faire'}</button>
      {#if !apercu?.impossible}
        <button class="btn-primary" disabled={!possible} onclick={executer}>
          <Icon name={textes.icone} size={16} />
          {enCours ? textes.encours : textes.bouton}
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
    width: min(760px, 100%);
    max-height: calc(100vh - 48px);
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 14px;
    padding: 22px;
    background: var(--surface);
    border: 1px solid color-mix(in srgb, var(--accent-2) 55%, transparent);
    border-radius: var(--cut) 0;
    corner-shape: bevel;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.6), 0 0 24px color-mix(in srgb, var(--accent-2) 14%, transparent);
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
    background: color-mix(in srgb, var(--accent-2) 14%, transparent);
    color: var(--accent-2);
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
  .resume .accent {
    color: var(--accent);
  }
  .liste {
    margin: 0;
    padding: 6px 0;
    list-style: none;
    max-height: 280px;
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
    padding: 5px 12px;
    font-size: 12px;
    color: var(--faint);
  }
  .chemins {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .vers {
    display: flex;
    gap: 6px;
    min-width: 0;
    color: var(--accent);
  }
  .fleche {
    flex: none;
  }
  .coupe {
    min-width: 0;
  }
  .de,
  .coupe {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    direction: rtl;
    text-align: left;
  }
  .de {
    color: var(--text);
  }
  .taille {
    flex: none;
    font-variant-numeric: tabular-nums;
    color: var(--muted);
  }
  .place {
    margin: 0;
    font-size: 12px;
    color: var(--accent);
    overflow-wrap: anywhere;
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
    margin: 6px 0 0;
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
</style>
