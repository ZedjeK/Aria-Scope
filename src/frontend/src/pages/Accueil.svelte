<script>
  import Icon from '../lib/Icon.svelte'
  import { app, lancer, choisirEtLancer, scanEnCours } from '../lib/state.svelte.js'
  import { octets, pourcent, pourcentTexte } from '../lib/format.js'

  const icones = { fixe: 'disk', amovible: 'usb', reseau: 'network', optique: 'disc', ram: 'disk' }
  const types = { fixe: 'disque interne', amovible: 'amovible', reseau: 'réseau', optique: 'optique', ram: 'RAM disque' }

  const enCours = $derived(scanEnCours())
</script>

<section class="page">
  <div class="page-head">
    <h1>disques</h1>
  </div>
  <p class="page-intro">
    Choisis un disque ou un dossier : Aria le fouille en parallèle et te montre qui prend toute la place.
  </p>

  {#if enCours}
    <p class="banner">
      Une analyse est en cours sur <span class="mono">{app.etat.scan.racine}</span>. En lancer une autre l'arrêtera.
    </p>
  {/if}

  <div class="volumes">
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
</style>
