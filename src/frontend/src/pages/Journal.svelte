<script>
  // Le journal des opérations de ménage, d'un lancement à l'autre, avec de
  // quoi les retrouver dans l'Explorateur et les annuler.
  import Icon from '../lib/Icon.svelte'
  import AnnulerOperation from '../lib/AnnulerOperation.svelte'
  import { app, call, message, relancer } from '../lib/state.svelte.js'
  import { octets, nombre } from '../lib/format.js'

  let entrees = $state(null)
  let erreur = $state('')
  let ouverts = $state({}) // id → détails dépliés
  let aAnnuler = $state(null)
  let resultat = $state(null) // dernière annulation

  const ICONES = { corbeille: 'trash', deplacer: 'move', copier: 'copy', archiver: 'archive' }
  const VERBES = { corbeille: 'envoyé{s} à la corbeille', deplacer: 'déplacé{s}', copier: 'copié{s}', archiver: 'archivé{s}' }

  async function charger() {
    try {
      entrees = (await call('Journal')) ?? []
      erreur = ''
    } catch (err) {
      erreur = message(err)
    }
  }

  $effect(() => {
    app.revision // une opération vient d'avoir lieu
    charger()
  })

  const titre = (e) => {
    const n = e.elements?.length ?? 0
    return `${nombre(n)} élément${n > 1 ? 's' : ''} ${(VERBES[e.operation] ?? 'traité{s}').replace('{s}', n > 1 ? 's' : '')}`
  }
  const quand = (iso) => {
    const d = new Date(iso)
    return `${d.toLocaleDateString('fr-FR', { day: 'numeric', month: 'short', year: 'numeric' })} à ${d.toLocaleTimeString('fr-FR', { hour: '2-digit', minute: '2-digit' })}`
  }
  const recycles = (e) => (e.elements ?? []).filter((x) => x.recycle).length

  async function montrer(chemin, selectionner) {
    try {
      await call('MontrerChemin', chemin, selectionner)
    } catch (err) {
      erreur = message(err)
    }
  }
  async function ouvrirCorbeille() {
    try {
      await call('OuvrirCorbeille')
    } catch (err) {
      erreur = message(err)
    }
  }

  async function fait(b) {
    resultat = b
    aAnnuler = null
    await charger()
  }
</script>

<section class="page">
  <div class="page-head">
    <h1>journal</h1>
  </div>
  <p class="page-intro">
    Tout ce que tu as fait depuis la barre de sélection. Un déplacement, une copie ou une archive peut être annulé, même des semaines plus tard.
  </p>

  {#if erreur}
    <p class="error">{erreur}</p>
  {/if}

  {#if resultat}
    <div class="bilan" class:mitige={resultat.restants.length > 0 || resultat.erreur || resultat.archiveGardee} role="status">
      <span class="b-icone"><Icon name={resultat.faits > 0 || resultat.archiveRecyclee ? 'check' : 'warn'} size={18} /></span>
      <div class="b-texte">
        <strong>Annulé : {resultat.resume || 'rien n’a pu être fait'}.</strong>
        {#if resultat.archiveGardee}<span>{resultat.archiveGardee[0].toUpperCase() + resultat.archiveGardee.slice(1)}.</span>{/if}
        {#if resultat.erreur}<span class="err">{resultat.erreur}</span>{/if}
        {#each resultat.restants.slice(0, 5) as r (r.chemin)}<span class="petit"><bdi>{r.chemin}</bdi> : {r.raison}</span>{/each}
        {#if app.etat?.scan}
          <span>L'analyse ne voit pas encore ce changement. <button class="lien" onclick={relancer}>Relancer l'analyse</button></span>
        {/if}
      </div>
      <button class="fermer" aria-label="Fermer" onclick={() => (resultat = null)}>×</button>
    </div>
  {/if}

  {#if entrees === null}
    <p class="hint">Lecture du journal…</p>
  {:else if entrees.length === 0}
    <div class="vide">
      <Icon name="history" size={28} />
      <p>Aucune opération pour l'instant.</p>
      <p class="hint">Sélectionne des éléments dans l'analyse : ce que tu déplaces, copies, archives ou envoies à la corbeille s'inscrira ici.</p>
    </div>
  {:else}
    <ul class="journal">
      {#each entrees as e (e.id)}
        <li class="entree" class:annulee={e.annulee}>
          <span class="icone op-{e.operation}"><Icon name={ICONES[e.operation] ?? 'file'} size={18} /></span>
          <div class="corps">
            <div class="ligne1">
              <span class="titre">{titre(e)}</span>
              {#if e.operation === 'archiver'}
                <span class="lieu">dans <bdi>{e.archive}</bdi></span>
              {:else if e.destination}
                <span class="lieu">vers <bdi>{e.destination}</bdi></span>
              {/if}
            </div>
            <div class="meta">
              <span>{quand(e.date)}</span>
              <span>{octets(e.taille)}</span>
              {#if e.operation === 'archiver' && recycles(e) > 0}<span>{nombre(recycles(e))} {recycles(e) > 1 ? 'originaux recyclés' : 'original recyclé'}</span>{/if}
              {#if e.annulee}<span class="badge">annulée le {quand(e.annulee)}{e.annulation ? ` · ${e.annulation}` : ''}</span>{/if}
            </div>
            {#if ouverts[e.id]}
              <ul class="details">
                {#each (e.elements ?? []).slice(0, 50) as x (x.origine)}
                  <li>
                    <Icon name={x.type === 'dossier' ? 'folder' : 'file'} size={13} />
                    <span class="chemin" title={x.origine}><bdi>{x.origine}</bdi></span>
                    {#if x.arrivee && e.operation !== 'archiver'}<span class="fleche">→</span><span class="chemin arrivee" title={x.arrivee}><bdi>{x.arrivee}</bdi></span>{/if}
                    {#if x.recycle}<span class="petit">recyclé</span>{/if}
                    <span class="taille">{octets(x.taille)}</span>
                  </li>
                {/each}
                {#if (e.elements?.length ?? 0) > 50}<li class="hint">… et {nombre(e.elements.length - 50)} autres</li>{/if}
              </ul>
            {/if}
          </div>
          <div class="actions">
            <button class="btn-ghost small" aria-expanded={!!ouverts[e.id]} onclick={() => (ouverts[e.id] = !ouverts[e.id])}>
              {ouverts[e.id] ? 'Masquer' : 'Détails'}
            </button>
            {#if e.operation === 'corbeille'}
              <button class="btn-ghost small" onclick={ouvrirCorbeille} title="Restaurer depuis la corbeille de Windows"><Icon name="trash" size={14} /> Ouvrir la corbeille</button>
            {:else}
              {#if !(e.operation === 'archiver' && e.annulee)}
                <button class="btn-ghost small" onclick={() => (e.operation === 'archiver' ? montrer(e.archive, true) : montrer(e.destination, false))}>
                  <Icon name="external" size={14} /> Montrer
                </button>
              {/if}
              {#if !e.annulee}
                <button class="btn-ghost small annuler" onclick={() => (aAnnuler = e)}><Icon name="history" size={14} /> Annuler…</button>
              {/if}
            {/if}
          </div>
        </li>
      {/each}
    </ul>
  {/if}
</section>

{#if aAnnuler}
  <AnnulerOperation entree={aAnnuler} onfermer={() => (aAnnuler = null)} onfait={fait} />
{/if}

<style>
  .journal {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .entree {
    display: flex;
    align-items: flex-start;
    gap: 14px;
    padding: 14px 16px;
    background: var(--surface);
    border: 1px solid var(--line);
    border-radius: 12px 0;
    corner-shape: bevel;
  }
  .entree.annulee {
    opacity: 0.62;
  }
  .icone {
    flex: none;
    display: flex;
    padding: 8px;
    border-radius: 8px 0;
    corner-shape: bevel;
    background: color-mix(in srgb, var(--accent) 12%, transparent);
    color: var(--accent);
  }
  .icone.op-corbeille {
    background: color-mix(in srgb, var(--danger) 12%, transparent);
    color: var(--danger);
  }
  .icone.op-archiver {
    background: color-mix(in srgb, var(--action) 14%, transparent);
    color: var(--action);
  }
  .icone.op-deplacer {
    background: color-mix(in srgb, var(--accent-2) 14%, transparent);
    color: var(--accent-2);
  }
  .corps {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .ligne1 {
    display: flex;
    flex-wrap: wrap;
    gap: 4px 8px;
    align-items: baseline;
  }
  .titre {
    font-size: 14px;
    font-weight: 600;
    color: #ffffff;
  }
  .lieu {
    min-width: 0;
    font-size: 12px;
    color: var(--muted);
    overflow-wrap: anywhere;
  }
  .lieu bdi {
    color: var(--accent);
  }
  .meta {
    display: flex;
    flex-wrap: wrap;
    gap: 4px 14px;
    font-size: 12px;
    color: var(--faint);
    font-variant-numeric: tabular-nums;
  }
  .badge {
    padding: 0 8px;
    border-radius: 6px 0;
    background: color-mix(in srgb, var(--accent-2) 16%, transparent);
    color: var(--ghost);
  }
  .details {
    list-style: none;
    margin: 6px 0 0;
    padding: 6px 0;
    background: var(--surface-sunken);
    border: 1px solid var(--line);
    border-radius: 8px 0;
    corner-shape: bevel;
  }
  .details li {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 3px 10px;
    font-size: 12px;
    color: var(--faint);
  }
  .chemin {
    flex: 0 1 auto;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    direction: rtl;
    text-align: left;
    color: var(--text);
  }
  .arrivee {
    color: var(--accent);
  }
  .fleche {
    flex: none;
  }
  .taille {
    flex: none;
    margin-left: auto;
    color: var(--muted);
  }
  .petit {
    font-size: 11px;
    color: var(--faint);
    overflow-wrap: anywhere;
  }
  .actions {
    flex: none;
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
    justify-content: flex-end;
  }
  .actions button {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
  .annuler {
    border-color: color-mix(in srgb, var(--accent-2) 60%, transparent);
    color: var(--ghost);
  }
  .vide {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    padding: 40px 20px;
    border: 1px dashed var(--line-strong);
    border-radius: 12px 0;
    corner-shape: bevel;
    color: var(--muted);
    text-align: center;
  }
  .vide p {
    margin: 0;
  }
  .bilan {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    padding: 12px 14px;
    border: 1px solid color-mix(in srgb, var(--accent) 50%, transparent);
    border-radius: 10px 0;
    corner-shape: bevel;
    background: color-mix(in srgb, var(--accent) 7%, transparent);
  }
  .bilan.mitige {
    border-color: color-mix(in srgb, var(--warn) 50%, transparent);
    background: color-mix(in srgb, var(--warn) 7%, transparent);
  }
  .b-icone {
    flex: none;
    display: flex;
    color: var(--accent);
  }
  .mitige .b-icone {
    color: var(--warn);
  }
  .b-texte {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: 13px;
    color: var(--muted);
  }
  .b-texte strong {
    color: var(--text);
  }
  .err {
    color: var(--danger);
  }
  .lien {
    padding: 0;
    border: none;
    background: none;
    color: var(--warn);
    font-family: var(--font-mono);
    font-size: 12px;
  }
  .fermer {
    flex: none;
    background: none;
    border: none;
    color: var(--muted);
    font-size: 18px;
    line-height: 1;
  }
  @media (max-width: 900px) {
    .entree {
      flex-wrap: wrap;
    }
    .actions {
      width: 100%;
      justify-content: flex-start;
    }
  }
</style>
