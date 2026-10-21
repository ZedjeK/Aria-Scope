<script>
  import Icon from '../lib/Icon.svelte'
  import Arbre from '../lib/Arbre.svelte'
  import Carte from '../lib/Carte.svelte'
  import Anneau from '../lib/Anneau.svelte'
  import GrosFichiers from '../lib/GrosFichiers.svelte'
  import Types from '../lib/Types.svelte'
  import Ages from '../lib/Ages.svelte'
  import PanneauFiltres from '../lib/PanneauFiltres.svelte'
  import { resume, actifs } from '../lib/filtres.js'
  import { app, relancer, arreter, choisirEtLancer, scanEnCours } from '../lib/state.svelte.js'
  import { octets, nombre, pourcent, pourcentTexte, duree, parSeconde } from '../lib/format.js'

  const scan = $derived(app.etat?.scan)
  const gen = $derived(app.etat?.gen)
  const enCours = $derived(scanEnCours())
  const attendu = $derived(app.etat?.attendu ?? 0)
  const avance = $derived(enCours && attendu > 0 ? Math.min(pourcent(scan.disque, attendu), 99) : null)
  const filtresScan = $derived(app.etat?.filtres) // ceux de l'analyse affichée
</script>

<section class="page">
  {#if !scan && app.demarrageAuto}
    <div class="page-head">
      <h1>analyse</h1>
    </div>
    <p class="hint">Démarrage de l'analyse de tout le PC…</p>
  {:else if !scan}
    <div class="page-head">
      <h1>analyse</h1>
    </div>
    <div class="card vide">
      <p class="card-text">Aucune analyse pour l'instant. Choisis un disque, ou va droit au but avec un dossier.</p>
      <div class="actions">
        <button class="btn-primary" onclick={() => (app.page = 'accueil')}><Icon name="disk" size={16} /> Voir les disques</button>
        <button class="btn-ghost" onclick={choisirEtLancer}><Icon name="folder" size={16} /> Choisir un dossier…</button>
      </div>
    </div>
  {:else}
    <div class="page-head">
      <h1>analyse</h1>
      <div class="actions">
        <div class="mode" role="radiogroup" aria-label="Mesure des tailles">
          <button role="radio" aria-checked={app.mode === 'taille'} class:on={app.mode === 'taille'} onclick={() => (app.mode = 'taille')} title="Taille des données des fichiers">taille réelle</button>
          <button role="radio" aria-checked={app.mode === 'disque'} class:on={app.mode === 'disque'} onclick={() => (app.mode = 'disque')} title="Place réellement occupée sur le disque (clusters, compression, fichiers OneDrive en ligne…)">sur disque</button>
        </div>
        {#if enCours}
          <button class="btn-primary" onclick={arreter}><Icon name="stop" size={16} /> Arrêter</button>
        {:else}
          <button class="btn-ghost" class:on={app.carte} aria-pressed={app.carte} onclick={() => (app.carte = !app.carte)} title={app.carte ? 'Masquer les visuels' : 'Afficher les visuels'}><Icon name="map" size={16} /> Visuels</button>
          <button class="btn-ghost" class:on={app.panneauFiltres || filtresScan} aria-expanded={app.panneauFiltres} onclick={() => (app.panneauFiltres = !app.panneauFiltres)}>
            <Icon name="filter" size={16} /> Filtres{#if filtresScan}<span class="compte">{resume(filtresScan).length}</span>{/if}
          </button>
          <button class="btn-ghost" onclick={relancer}><Icon name="refresh" size={16} /> Relancer</button>
          <button class="btn-ghost" onclick={() => (app.page = 'accueil')}><Icon name="scope" size={16} /> Nouvelle analyse</button>
        {/if}
      </div>
    </div>
    <p class="path racine-chemin">{scan.racine}</p>

    {#if app.panneauFiltres && !enCours}
      <PanneauFiltres racine={scan.racine} onfermer={() => (app.panneauFiltres = false)} />
    {/if}
    {#if filtresScan}
      <div class="filtres-actifs" role="status">
        <span class="fa-titre"><Icon name="filter" size={14} /> Analyse filtrée</span>
        {#each resume(filtresScan) as r (r)}<span class="chip warn">{r}</span>{/each}
        {#if scan.ecartes > 0}<span class="hint">{nombre(scan.ecartes)} fichier{scan.ecartes > 1 ? 's' : ''} écarté{scan.ecartes > 1 ? 's' : ''}</span>{/if}
      </div>
    {/if}

    {#if enCours}
      <article class="card progression" aria-live="polite">
        <div class="card-head">
          <h2>fouille en cours</h2>
          <span class="chip ok">{app.etat.parallelisme} lectures en parallèle</span>
        </div>
        <div class="compteurs">
          <div><span class="eyebrow">trouvé</span><span class="grand">{octets(app.mode === 'disque' ? scan.disque : scan.taille)}</span></div>
          <div><span class="eyebrow">fichiers</span><span class="grand">{nombre(scan.fichiers)}</span><span class="hint">{parSeconde(scan.fichiers, scan.dureeMs)}</span></div>
          <div><span class="eyebrow">dossiers</span><span class="grand">{nombre(scan.dossiers)}</span></div>
          <div><span class="eyebrow">durée</span><span class="grand">{duree(scan.dureeMs)}</span></div>
        </div>
        <div class="barre-scan" class:indefini={avance === null}>
          <span style:width={avance === null ? '30%' : `${avance}%`}></span>
        </div>
        <div class="sous-barre">
          <span class="courant" title={scan.courant}><bdi>{scan.courant}</bdi></span>
          {#if avance !== null}<span class="mono">≈ {pourcentTexte(avance)} de l'espace utilisé</span>{/if}
        </div>
      </article>
    {:else}
      {#if scan.etat === 'arrete'}
        <p class="banner">Analyse arrêtée avant la fin : les chiffres ne couvrent que ce qui a été lu.</p>
      {/if}

      <div class="resume">
        <div class="stat" class:fort={app.mode === 'taille'}><span class="eyebrow">taille réelle</span><span class="valeur-stat">{octets(scan.taille)}</span></div>
        <div class="stat" class:fort={app.mode === 'disque'}><span class="eyebrow">sur disque</span><span class="valeur-stat">{octets(scan.disque)}</span></div>
        <div class="stat"><span class="eyebrow">fichiers</span><span class="valeur-stat">{nombre(scan.fichiers)}</span></div>
        <div class="stat"><span class="eyebrow">dossiers</span><span class="valeur-stat">{nombre(scan.dossiers)}</span></div>
        <div class="stat"><span class="eyebrow">durée</span><span class="valeur-stat">{duree(scan.dureeMs)}</span><span class="hint">{parSeconde(scan.fichiers, scan.dureeMs)}</span></div>
        <div class="stat" class:alerte={scan.erreurs > 0}><span class="eyebrow">illisibles</span><span class="valeur-stat">{nombre(scan.erreurs)}</span></div>
        <div class="stat"><span class="eyebrow">liens ignorés</span><span class="valeur-stat">{nombre(scan.liens)}</span></div>
      </div>

      {#if scan.erreurs > 0}
        <p class="hint">
          {nombre(scan.erreurs)} dossier{scan.erreurs > 1 ? 's' : ''} n'{scan.erreurs > 1 ? 'ont' : 'a'} pas pu être lu{scan.erreurs > 1 ? 's' : ''}, souvent des dossiers système protégés.
          Lancé en administrateur, Aria Scope en lira la plupart.
        </p>
      {/if}

      {#key gen}
        <div class="vue" class:avec-carte={app.carte}>
          <div class="col-arbre">
            <div class="onglets" role="tablist" aria-label="Vue">
              <button role="tab" aria-selected={app.vue === 'arbre'} class:on={app.vue === 'arbre'} onclick={() => (app.vue = 'arbre')}>arbre</button>
              <button role="tab" aria-selected={app.vue === 'gros'} class:on={app.vue === 'gros'} onclick={() => (app.vue = 'gros')}>gros fichiers</button>
              <button role="tab" aria-selected={app.vue === 'types'} class:on={app.vue === 'types'} onclick={() => (app.vue = 'types')}>types</button>
              <button role="tab" aria-selected={app.vue === 'ages'} class:on={app.vue === 'ages'} onclick={() => (app.vue = 'ages')}>âge</button>
            </div>
            {#if app.vue === 'arbre'}
              <Arbre {gen} mode={app.mode} pc={scan.pc} bind:tri={app.tri} bind:selection={app.selection} />
            {:else if app.vue === 'gros'}
              <GrosFichiers {gen} mode={app.mode} bind:selection={app.selection} />
            {:else if app.vue === 'types'}
              <Types {gen} mode={app.mode} selection={app.selection} bind:portee={app.portee} />
            {:else}
              <Ages {gen} mode={app.mode} selection={app.selection} bind:portee={app.portee} />
            {/if}
          </div>
          {#if app.carte}
            <div class="col-carte">
              <div class="onglets" role="tablist" aria-label="Visuel">
                <button role="tab" aria-selected={app.visuel === 'carte'} class:on={app.visuel === 'carte'} onclick={() => (app.visuel = 'carte')}>carte</button>
                <button role="tab" aria-selected={app.visuel === 'anneau'} class:on={app.visuel === 'anneau'} onclick={() => (app.visuel = 'anneau')}>anneau</button>
              </div>
              <div class="visuel">
                {#if app.visuel === 'carte'}
                  <Carte {gen} mode={app.mode} bind:selection={app.selection} />
                {:else}
                  <Anneau {gen} mode={app.mode} bind:selection={app.selection} />
                {/if}
              </div>
            </div>
          {/if}
        </div>
      {/key}
    {/if}
  {/if}
</section>

<style>
  .vide {
    align-items: flex-start;
  }
  .btn-ghost.on {
    border-color: color-mix(in srgb, var(--accent) 55%, transparent);
    color: var(--accent);
  }
  .compte {
    min-width: 18px;
    padding: 1px 5px;
    border-radius: 9px;
    background: var(--warn);
    color: var(--on-action);
    font-size: 11px;
    text-align: center;
  }
  .filtres-actifs {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px;
    margin-top: -6px;
    padding: 10px 12px;
    border: 1px solid color-mix(in srgb, var(--warn) 40%, transparent);
    border-radius: 10px 0;
    corner-shape: bevel;
    background: color-mix(in srgb, var(--warn) 7%, transparent);
  }
  .fa-titre {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    letter-spacing: 1px;
    text-transform: uppercase;
    color: var(--warn);
  }
  .filtres-actifs .chip {
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  /* Arbre et carte : côte à côte quand il y a la place, la carte au-dessus sinon. */
  .vue {
    container-type: inline-size;
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 14px;
    align-items: start;
  }
  .col-carte {
    order: -1;
    height: 420px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .visuel {
    flex: 1;
    min-height: 0;
  }
  .col-arbre {
    display: flex;
    flex-direction: column;
    gap: 8px;
    min-width: 0;
  }
  .onglets {
    flex: none;
    align-self: flex-start;
    display: inline-flex;
    padding: 3px;
    border: 1px solid var(--line-strong);
    border-radius: 10px 0;
    corner-shape: bevel;
    background: var(--surface-sunken);
  }
  .onglets button {
    min-height: 30px;
    padding: 0 12px;
    border: none;
    border-radius: 8px 0;
    corner-shape: bevel;
    background: none;
    color: var(--muted);
    font-family: var(--font-mono);
    font-size: 12px;
  }
  .onglets button.on {
    background: color-mix(in srgb, var(--accent) 14%, transparent);
    color: var(--accent);
  }
  @container (min-width: 900px) {
    .vue.avec-carte {
      grid-template-columns: minmax(0, 1.15fr) minmax(0, 1fr);
    }
    .col-carte {
      order: 0;
      position: sticky;
      top: 0;
      height: min(calc(100vh - 60px), 820px);
    }
  }
  .actions button {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }
  .racine-chemin {
    margin-top: -12px;
    font-size: 13px;
    color: var(--accent);
  }

  .mode {
    display: inline-flex;
    padding: 3px;
    border: 1px solid var(--line-strong);
    border-radius: 10px 0;
    corner-shape: bevel;
    background: var(--surface-sunken);
  }
  .mode button {
    min-height: 36px;
    padding: 0 12px;
    border: none;
    border-radius: 8px 0;
    corner-shape: bevel;
    background: none;
    color: var(--muted);
    font-family: var(--font-mono);
    font-size: 12px;
    font-weight: 500;
  }
  .mode button.on {
    background: color-mix(in srgb, var(--accent) 14%, transparent);
    color: var(--accent);
  }

  .compteurs {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 16px;
  }
  .compteurs > div,
  .stat {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }
  .grand {
    font-family: var(--font-display);
    font-size: 28px;
    font-weight: 700;
    color: #ffffff;
    font-variant-numeric: tabular-nums;
  }
  .barre-scan {
    height: 8px;
    border-radius: 4px;
    background: var(--surface-sunken);
    border: 1px solid var(--line);
    overflow: hidden;
  }
  .barre-scan span {
    display: block;
    height: 100%;
    background: linear-gradient(90deg, var(--action), #ffb15c);
    box-shadow: 0 0 10px var(--action);
    transition: width 0.3s;
  }
  .barre-scan.indefini span {
    animation: balayage 1.4s ease-in-out infinite;
  }
  @keyframes balayage {
    from {
      transform: translateX(-100%);
    }
    to {
      transform: translateX(340%);
    }
  }
  .sous-barre {
    display: flex;
    justify-content: space-between;
    gap: 16px;
    font-size: 12px;
    color: var(--faint);
  }
  .courant {
    min-width: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    direction: rtl;
    text-align: left;
  }
  .sous-barre .mono {
    flex: none;
  }

  .resume {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(112px, 1fr));
    gap: 10px;
  }
  .stat {
    padding: 12px 14px;
    background: var(--surface);
    border: 1px solid var(--line);
    border-radius: 10px 0;
    corner-shape: bevel;
  }
  .valeur-stat {
    font-family: var(--font-display);
    font-size: 19px;
    font-weight: 600;
    color: var(--text);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .stat.fort {
    border-color: color-mix(in srgb, var(--accent) 50%, transparent);
  }
  .stat.fort .valeur-stat {
    color: #ffffff;
  }
  .stat.alerte {
    border-color: color-mix(in srgb, var(--warn) 45%, transparent);
  }
  .stat.alerte .valeur-stat {
    color: var(--warn);
  }
</style>
