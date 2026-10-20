<script>
  import { onMount } from 'svelte'
  import Icon from './lib/Icon.svelte'
  import Logo from './lib/Logo.svelte'
  import { app, refresh, chargerVolumes, suivre, scanEnCours } from './lib/state.svelte.js'
  import { octets, pourcent, pourcentTexte } from './lib/format.js'
  import Accueil from './pages/Accueil.svelte'
  import Analyse from './pages/Analyse.svelte'

  const nav = [
    { id: 'accueil', label: 'Disques', cmd: 'disques', icon: 'home' },
    { id: 'analyse', label: 'Analyse', cmd: 'analyse', icon: 'scope' },
  ]

  const scan = $derived(app.etat?.scan)
  const enCours = $derived(scanEnCours())
  const version = $derived(app.etat?.version ?? '')
  const avance = $derived(enCours && app.etat?.attendu > 0 ? pourcent(scan.disque, app.etat.attendu) : null)

  onMount(() => {
    refresh()
    chargerVolumes()
    suivre()
  })
</script>

<div class="shell">
  <nav class="sidebar" aria-label="Sections">
    <div class="brand">
      <Logo size={56} actif={enCours} />
      <span class="brand-name label"><span class="brand-top">ARIA</span><span class="brand-sub">SCOPE</span></span>
    </div>

    <ul class="nav">
      {#each nav as item (item.id)}
        {@const lit = item.id === 'analyse' && enCours}
        <li>
          <button
            class="nav-item"
            class:active={app.page === item.id}
            aria-current={app.page === item.id ? 'page' : undefined}
            aria-label={item.label}
            title={item.label}
            onclick={() => (app.page = item.id)}
          >
            <span class="nav-icon">
              <Icon name={item.icon} />
              {#if lit}<span class="dot icon-dot" aria-hidden="true"></span>{/if}
            </span>
            <span class="prompt label" aria-hidden="true">&gt;</span>
            <span class="label">{item.cmd}</span>
            {#if lit}<span class="dot label" aria-hidden="true"></span>{/if}
          </button>
        </li>
      {/each}
    </ul>

    <div class="sidebar-foot">
      {#if scan}
        <button class="statut" class:live={enCours} onclick={() => (app.page = 'analyse')} title={scan.racine}>
          <span class="statut-titre label">
            {#if enCours}Analyse en cours{avance !== null ? ` · ${pourcentTexte(Math.min(avance, 99))}` : '…'}
            {:else if scan.etat === 'arrete'}Analyse arrêtée
            {:else}Dernière analyse{/if}
          </span>
          <span class="statut-chemin label"><bdi>{scan.racine}</bdi></span>
          <span class="statut-taille label">{octets(app.mode === 'disque' ? scan.disque : scan.taille)}</span>
          {#if enCours}
            <span class="statut-barre" class:indefini={avance === null} aria-hidden="true">
              <span style:width={avance === null ? '35%' : `${Math.min(avance, 99)}%`}></span>
            </span>
          {/if}
        </button>
      {/if}
      <span class="version label">aria-scope {version ? `v${version}` : ''}</span>
    </div>
  </nav>

  <main class="content">
    {#if app.loadError}
      <p class="banner" role="alert">Impossible de joindre le moteur d'Aria : {app.loadError}</p>
    {/if}
    {#if app.erreur}
      <p class="banner" role="alert">
        {app.erreur}
        <button class="fermer" aria-label="Fermer le message" onclick={() => (app.erreur = '')}>×</button>
      </p>
    {/if}

    {#if app.page === 'accueil'}
      <Accueil />
    {:else}
      <Analyse />
    {/if}
  </main>
</div>

<style>
  .shell {
    display: grid;
    grid-template-columns: 240px minmax(0, 1fr);
    height: 100vh;
  }

  .sidebar {
    background: var(--sidebar);
    border-right: 1px solid var(--line-soft);
    padding: 18px 12px;
    display: flex;
    flex-direction: column;
    gap: 20px;
    min-width: 0;
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 2px 4px 6px;
    min-height: 58px;
  }
  .brand-name {
    display: flex;
    flex-direction: column;
    line-height: 1.15;
  }
  .brand-top {
    font-family: var(--font-display);
    font-weight: 700;
    font-size: 26px;
    color: #ffffff;
    letter-spacing: 1px;
    text-shadow: -2px 0 0 color-mix(in srgb, var(--accent) 55%, transparent), 2px 0 0 color-mix(in srgb, var(--accent-2) 55%, transparent);
  }
  .brand-sub {
    font-size: 14px;
    letter-spacing: 5px;
    color: var(--accent);
  }

  .nav {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .nav-item {
    width: 100%;
    min-height: 42px;
    padding: 0 10px;
    display: flex;
    align-items: center;
    gap: 8px;
    border: none;
    border-radius: 8px 0;
    corner-shape: bevel;
    background: transparent;
    color: var(--muted);
    font-family: var(--font-mono);
    font-size: 14px;
    font-weight: 500;
    text-align: left;
  }
  .nav-item:hover {
    background: var(--hover);
    color: var(--text);
  }
  .nav-item.active {
    background: #0b1a30;
    color: var(--accent);
  }
  .prompt {
    color: var(--dim);
  }
  .nav-item.active .prompt {
    color: var(--action);
  }
  .nav-icon {
    position: relative;
    display: none;
    flex: none;
  }
  .dot {
    flex: none;
    margin-left: auto;
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--action);
    box-shadow: 0 0 8px var(--action);
    animation: pulse 1.2s ease-in-out infinite;
  }
  @keyframes pulse {
    50% {
      opacity: 0.35;
    }
  }
  .icon-dot {
    position: absolute;
    right: -3px;
    top: -3px;
    margin: 0;
    border: 2px solid var(--sidebar);
    width: 9px;
    height: 9px;
  }
  .label {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .sidebar-foot {
    margin-top: auto;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .statut {
    display: flex;
    flex-direction: column;
    gap: 3px;
    padding: 10px 12px;
    border: 1px solid var(--line-strong);
    border-radius: 12px 0;
    corner-shape: bevel;
    background: var(--surface);
    color: var(--muted);
    text-align: left;
    font-family: var(--font-mono);
    min-width: 0;
  }
  .statut:hover {
    border-color: var(--accent-2);
  }
  .statut.live {
    border-color: color-mix(in srgb, var(--action) 55%, transparent);
    box-shadow: 0 0 14px color-mix(in srgb, var(--action) 18%, transparent);
  }
  .statut-titre {
    font-size: 11px;
    letter-spacing: 1px;
    text-transform: uppercase;
    color: var(--faint);
  }
  .statut.live .statut-titre {
    color: var(--action);
  }
  .statut-chemin {
    font-size: 12px;
    color: var(--text);
    direction: rtl;
    text-align: left;
  }
  .statut-taille {
    font-family: var(--font-display);
    font-size: 18px;
    font-weight: 600;
    color: #ffffff;
  }
  .statut-barre {
    display: block;
    height: 4px;
    margin-top: 4px;
    border-radius: 2px;
    background: var(--line-soft);
    overflow: hidden;
  }
  .statut-barre span {
    display: block;
    height: 100%;
    background: linear-gradient(90deg, var(--action), #ffb15c);
    box-shadow: 0 0 8px var(--action);
    transition: width 0.3s;
  }
  .statut-barre.indefini span {
    animation: balayage 1.4s ease-in-out infinite;
  }
  @keyframes balayage {
    from {
      transform: translateX(-100%);
    }
    to {
      transform: translateX(300%);
    }
  }
  .version {
    padding: 0 4px;
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--faint);
  }

  .content {
    overflow-y: auto;
    padding: 26px 30px 30px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .banner {
    display: flex;
    align-items: flex-start;
    gap: 12px;
  }
  .fermer {
    margin-left: auto;
    flex: none;
    background: none;
    border: none;
    color: inherit;
    font-size: 18px;
    line-height: 1;
    padding: 0 4px;
  }

  /* Fenêtre étroite : la barre latérale se replie en icônes */
  @media (max-width: 760px) {
    .shell {
      grid-template-columns: 68px minmax(0, 1fr);
    }
    .sidebar {
      padding: 16px 8px;
      align-items: center;
    }
    .brand {
      padding: 0;
      justify-content: center;
    }
    .label,
    .statut {
      display: none;
    }
    .nav-icon {
      display: flex;
    }
    .nav-item {
      justify-content: center;
      padding: 0;
      width: 48px;
      min-height: 48px;
    }
    .brand :global(svg) {
      width: 44px;
      height: 44px;
    }
    .nav {
      align-items: center;
    }
    .content {
      padding: 20px;
    }
  }
</style>
