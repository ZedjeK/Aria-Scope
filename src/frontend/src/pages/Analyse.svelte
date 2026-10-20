<script>
  import Icon from '../lib/Icon.svelte'
  import Arbre from '../lib/Arbre.svelte'
  import { app, lancer, arreter, choisirEtLancer, scanEnCours } from '../lib/state.svelte.js'
  import { octets, nombre, pourcent, pourcentTexte, duree, parSeconde } from '../lib/format.js'

  const scan = $derived(app.etat?.scan)
  const gen = $derived(app.etat?.gen)
  const enCours = $derived(scanEnCours())
  const attendu = $derived(app.etat?.attendu ?? 0)
  const avance = $derived(enCours && attendu > 0 ? Math.min(pourcent(scan.disque, attendu), 99) : null)
</script>

<section class="page">
  {#if !scan}
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
          <button class="btn-ghost" onclick={() => lancer(scan.racine)}><Icon name="refresh" size={16} /> Relancer</button>
          <button class="btn-ghost" onclick={() => (app.page = 'accueil')}><Icon name="scope" size={16} /> Nouvelle analyse</button>
        {/if}
      </div>
    </div>
    <p class="path racine-chemin">{scan.racine}</p>

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
        <Arbre {gen} mode={app.mode} bind:tri={app.tri} />
      {/key}
    {/if}
  {/if}
</section>

<style>
  .vide {
    align-items: flex-start;
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
