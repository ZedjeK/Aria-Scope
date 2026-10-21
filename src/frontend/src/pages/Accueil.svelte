<script>
  import Icon from '../lib/Icon.svelte'
  import PanneauFiltres from '../lib/PanneauFiltres.svelte'
  import Switch from '../lib/Switch.svelte'
  import { resume, actifs, filtresVides } from '../lib/filtres.js'
  import { app, lancer, lancerPC, choisirEtLancer, scanEnCours, setAnalyseAuto } from '../lib/state.svelte.js'
  import { octets, pourcent, pourcentTexte } from '../lib/format.js'

  const icones = { fixe: 'disk', amovible: 'usb', reseau: 'network', optique: 'disc', ram: 'disk' }
  const types = { fixe: 'disque interne', amovible: 'amovible', reseau: 'réseau', optique: 'optique', ram: 'RAM disque' }

  const enCours = $derived(scanEnCours())

  // Ce que « Tout le PC » analyse : disques internes et externes.
  const locaux = $derived(app.volumes.filter((v) => v.type === 'fixe' || v.type === 'amovible'))
  const pcUtilise = $derived(locaux.reduce((s, v) => s + v.total - v.libre, 0))
  const pcTotal = $derived(locaux.reduce((s, v) => s + v.total, 0))
</script>

<section class="page">
  <div class="page-head">
    <h1>disques</h1>
    <button class="btn-ghost filtres-btn" class:on={actifs(app.filtres)} aria-expanded={app.panneauFiltres} onclick={() => (app.panneauFiltres = !app.panneauFiltres)}>
      <Icon name="filter" size={16} /> Filtres
    </button>
  </div>
  <p class="page-intro">
    Choisis un disque ou un dossier : Aria le fouille en parallèle et te montre qui prend toute la place.
  </p>

  {#if app.panneauFiltres}
    <PanneauFiltres onfermer={() => (app.panneauFiltres = false)} />
  {:else if actifs(app.filtres)}
    <div class="filtres-actifs" role="status">
      <span class="fa-titre"><Icon name="filter" size={14} /> Prochaine analyse filtrée</span>
      {#each resume(app.filtres) as r (r)}<span class="chip warn">{r}</span>{/each}
      <button class="lien" onclick={() => (app.filtres = filtresVides())}>effacer</button>
    </div>
  {/if}

  {#if enCours}
    <p class="banner">
      Une analyse est en cours sur <span class="mono">{app.etat.scan.racine}</span>. En lancer une autre l'arrêtera.
    </p>
  {/if}

  <div class="volumes">
    {#if locaux.length > 0}
      {@const p = pourcent(pcUtilise, pcTotal)}
      <article class="card volume pc">
        <div class="vol-head">
          <span class="vol-icon"><Icon name="scope" size={22} /></span>
          <div class="vol-titre">
            <span class="vol-lettre">Tout le PC</span>
            <span class="vol-nom">{locaux.map((v) => v.chemin.replace(/\\$/, '')).join(' · ')}</span>
          </div>
          <span class="chip">{locaux.length} disque{locaux.length > 1 ? 's' : ''}</span>
        </div>
        <div class="jauge" class:plein={p >= 90} role="meter" aria-valuemin="0" aria-valuemax="100" aria-valuenow={Math.round(p)} aria-label="Espace utilisé sur l'ensemble des disques">
          <span style:width="{p}%"></span>
        </div>
        <div class="vol-chiffres">
          <span><strong>{octets(pcUtilise)}</strong> utilisés · {pourcentTexte(p)}</span>
          <span>sur {octets(pcTotal)}</span>
        </div>
        <label class="auto">
          <Switch checked={app.reglages.analyseAuto} label="Analyser tout le PC à l'ouverture" onchange={setAnalyseAuto} />
          <span>Analyser tout le PC à l'ouverture d'Aria Scope</span>
        </label>
        <button class="btn-primary" onclick={lancerPC}>
          <Icon name="scope" size={16} /> Analyser tout le PC
        </button>
      </article>
    {/if}

    {#each app.volumes as v (v.chemin)}
      {@const utilise = v.total - v.libre}
      {@const p = pourcent(utilise, v.total)}
      <article class="card volume">
        <div class="vol-head">
          <span class="vol-icon"><Icon name={icones[v.type] ?? 'disk'} size={22} /></span>
          <div class="vol-titre">
            <span class="vol-lettre">{v.chemin}</span>
            <span class="vol-nom">{v.nom || 'Sans nom'}</span>
          </div>
          <span class="chip">{[types[v.type], v.systeme].filter(Boolean).join(' · ')}</span>
        </div>

        <div class="jauge" class:plein={p >= 90} role="meter" aria-valuemin="0" aria-valuemax="100" aria-valuenow={Math.round(p)} aria-label="Espace utilisé">
          <span style:width="{p}%"></span>
        </div>
        <div class="vol-chiffres">
          <span><strong>{octets(utilise)}</strong> utilisés · {pourcentTexte(p)}</span>
          <span>{octets(v.libre)} libres sur {octets(v.total)}</span>
        </div>

        <button class="btn-primary" onclick={() => lancer(v.chemin)}>
          <Icon name="scope" size={16} /> Analyser {v.chemin}
        </button>
      </article>
    {:else}
      <p class="hint">Aucun disque trouvé. Tu peux quand même choisir un dossier.</p>
    {/each}

    <article class="card volume dossier">
      <div class="vol-head">
        <span class="vol-icon"><Icon name="folder-search" size={22} /></span>
        <div class="vol-titre">
          <span class="vol-lettre">Un dossier précis</span>
          <span class="vol-nom">Téléchargements, jeux, projets…</span>
        </div>
      </div>
      <p class="card-text">Plus rapide qu'un disque entier quand tu sais déjà où chercher.</p>
      <button class="btn-ghost" onclick={choisirEtLancer}>
        <Icon name="folder" size={16} /> Choisir un dossier…
      </button>
    </article>
  </div>
</section>

<style>
  .filtres-btn {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }
  .filtres-btn.on {
    border-color: color-mix(in srgb, var(--warn) 60%, transparent);
    color: var(--warn);
  }
  .filtres-actifs {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px;
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
  .lien {
    margin-left: auto;
    background: none;
    border: none;
    color: var(--muted);
    font-family: var(--font-mono);
    font-size: 12px;
    text-decoration: underline;
  }
  .lien:hover {
    color: var(--text);
  }
  .volumes {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 16px;
  }
  .volume {
    gap: 14px;
  }
  .vol-head {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .vol-icon {
    flex: none;
    display: flex;
    padding: 9px;
    border-radius: 8px 0;
    corner-shape: bevel;
    background: color-mix(in srgb, var(--accent) 10%, transparent);
    color: var(--accent);
  }
  .dossier .vol-icon {
    background: color-mix(in srgb, var(--action) 12%, transparent);
    color: var(--action);
  }
  .vol-titre {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
    min-width: 0;
  }
  .vol-lettre {
    font-family: var(--font-display);
    font-size: 20px;
    font-weight: 700;
    color: #ffffff;
  }
  .vol-nom {
    font-size: 12px;
    color: var(--faint);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .jauge {
    height: 10px;
    border-radius: 5px;
    background: var(--surface-sunken);
    border: 1px solid var(--line);
    overflow: hidden;
  }
  .jauge span {
    display: block;
    height: 100%;
    background: linear-gradient(90deg, var(--accent), var(--accent-2));
    box-shadow: 0 0 10px color-mix(in srgb, var(--accent) 60%, transparent);
  }
  .jauge.plein span {
    background: linear-gradient(90deg, var(--action), var(--danger));
    box-shadow: 0 0 10px color-mix(in srgb, var(--danger) 60%, transparent);
  }
  .vol-chiffres {
    display: flex;
    justify-content: space-between;
    gap: 8px;
    flex-wrap: wrap;
    font-size: 12px;
    color: var(--muted);
  }
  .vol-chiffres strong {
    color: var(--text);
  }
  .volume button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    margin-top: auto;
  }
  .dossier {
    border-style: dashed;
  }
  .pc {
    border-color: color-mix(in srgb, var(--action) 45%, transparent);
    box-shadow: 0 0 18px color-mix(in srgb, var(--action) 10%, transparent);
  }
  .auto {
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 12px;
    color: var(--muted);
    cursor: pointer;
  }
  .pc .vol-icon {
    background: color-mix(in srgb, var(--action) 14%, transparent);
    color: var(--action);
  }
</style>
