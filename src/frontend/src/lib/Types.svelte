<script>
  // Répartition par type de fichier : familles (Vidéos, Archives…) dépliables
  // en extensions. Pour toute l'analyse, ou pour le dossier sélectionné.
  import Icon from './Icon.svelte'
  import { call, message } from './state.svelte.js'
  import { octets, nombre, pourcent, pourcentTexte } from './format.js'

  let { gen, mode, selection } = $props()

  const COULEURS = {
    Vidéos: '#ff6fae',
    Images: '#b18cff',
    Audio: '#59c3ff',
    Archives: '#ffd166',
    'Images disque et VM': '#ff8a1f',
    Documents: '#7df9ff',
    Code: '#3fe0c5',
    'Programmes et bibliothèques': '#5c8bff',
    Jeux: '#b8f35a',
    '3D et impression 3D': '#ff9d6b',
    'Bases de données': '#a9b8ff',
    Polices: '#d4a5ff',
    'Temporaires et journaux': '#ff5c7a',
    'Sans extension': '#93a8c8',
    Autres: '#3b5378',
  }

  let portee = $state('tout') // "tout" ou "selection"
  let familles = $state(null)
  let erreur = $state('')
  let chargement = $state(false)
  let ouvertes = $state({})

  // Le dossier de la sélection (des fichiers : celui qui les contient)
  const idSelection = $derived(selection ? selection.id : 0)
  const id = $derived(portee === 'selection' ? idSelection : 0)
  let nomSelection = $state('')

  $effect(() => {
    const g = gen
    const i = idSelection
    if (!i) {
      nomSelection = ''
      if (portee === 'selection') portee = 'tout'
      return
    }
    call('Fil', g, i)
      .then((f) => g === gen && i === idSelection && (nomSelection = f?.at(-1)?.nom ?? ''))
      .catch(() => {})
  })

  $effect(() => {
    const g = gen
    const i = id
    const m = mode
    chargement = true
    erreur = ''
    call('Types', g, i, m)
      .then((f) => {
        if (g === gen && i === id && m === mode) familles = f ?? []
      })
      .catch((err) => (erreur = message(err)))
      .finally(() => (chargement = false))
  })

  const val = (x) => (mode === 'disque' ? x.disque : x.taille)
  const total = $derived((familles ?? []).reduce((s, f) => s + val(f), 0))
  const max = $derived(familles?.length ? val(familles[0]) : 0)
</script>

<div class="types">
  <div class="tete">
    <div class="portee" role="radiogroup" aria-label="Portée">
      <button role="radio" aria-checked={portee === 'tout'} class:on={portee === 'tout'} onclick={() => (portee = 'tout')}>toute l'analyse</button>
      <button role="radio" aria-checked={portee === 'selection'} class:on={portee === 'selection'} disabled={!idSelection} onclick={() => (portee = 'selection')} title={idSelection ? nomSelection : 'Sélectionne un dossier dans l\'arbre ou sur la carte'}>
        {idSelection ? `dans ${nomSelection || '…'}` : 'dossier sélectionné'}
      </button>
    </div>
    {#if chargement}<span class="hint">lecture…</span>{/if}
  </div>

  {#if erreur}
    <p class="error">{erreur}</p>
  {:else if familles && familles.length === 0}
    <p class="hint">Aucun fichier ici.</p>
  {:else if familles}
    <ul class="liste">
      {#each familles as f (f.nom)}
        {@const p = pourcent(val(f), total)}
        {@const couleur = COULEURS[f.nom] ?? '#93a8c8'}
        <li>
          <button class="famille" aria-expanded={!!ouvertes[f.nom]} onclick={() => (ouvertes[f.nom] = !ouvertes[f.nom])}>
            <span class="pli" class:ouvert={ouvertes[f.nom]}><Icon name="chevron" size={14} /></span>
            <span class="pastille" style:background={couleur} style:box-shadow="0 0 8px {couleur}"></span>
            <span class="nom">{f.nom}</span>
            <span class="barre"><span style:width="{pourcent(val(f), max)}%" style:background={couleur}></span></span>
            <span class="num fort">{octets(val(f))}</span>
            <span class="num">{pourcentTexte(p)}</span>
            <span class="num discret">{nombre(f.fichiers)} fichier{f.fichiers > 1 ? 's' : ''}</span>
          </button>
          {#if ouvertes[f.nom]}
            <ul class="exts">
              {#each f.extensions as x (x.ext)}
                <li class="ext">
                  <span class="nom mono">{x.ext ? `.${x.ext}` : '(aucune)'}</span>
                  <span class="barre fine"><span style:width="{pourcent(val(x), val(f))}%" style:background={couleur}></span></span>
                  <span class="num">{octets(val(x))}</span>
                  <span class="num">{pourcentTexte(pourcent(val(x), total))}</span>
                  <span class="num discret">{nombre(x.fichiers)}</span>
                </li>
              {/each}
              {#if f.extensions.length === 30}<li class="hint plus">… les 30 plus lourdes seulement</li>{/if}
            </ul>
          {/if}
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .types {
    container-type: inline-size;
    background: var(--surface);
    border: 1px solid var(--line);
    border-radius: var(--cut) 0;
    corner-shape: bevel;
    overflow: clip;
  }
  .tete {
    display: flex;
    align-items: center;
    gap: 10px;
    min-height: 46px;
    padding: 0 10px;
    background: var(--surface-sunken);
    border-bottom: 1px solid var(--line-soft);
  }
  .portee {
    display: inline-flex;
    padding: 3px;
    border: 1px solid var(--line-strong);
    border-radius: 10px 0;
    corner-shape: bevel;
  }
  .portee button {
    max-width: 260px;
    min-height: 30px;
    padding: 0 12px;
    border: none;
    border-radius: 8px 0;
    corner-shape: bevel;
    background: none;
    color: var(--muted);
    font-family: var(--font-mono);
    font-size: 12px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .portee button.on {
    background: color-mix(in srgb, var(--accent) 14%, transparent);
    color: var(--accent);
  }
  .portee button:disabled {
    opacity: 0.45;
  }
  .liste,
  .exts {
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .liste > li + li {
    border-top: 1px solid var(--line-soft);
  }
  .famille,
  .ext {
    width: 100%;
    display: grid;
    grid-template-columns: 18px 12px minmax(120px, 1fr) minmax(60px, 1.2fr) 84px 56px 110px;
    align-items: center;
    gap: 10px;
    min-height: 40px;
    padding: 0 12px;
    font-size: 13px;
  }
  .famille {
    border: none;
    background: none;
    color: var(--text);
    font-family: inherit;
    text-align: left;
  }
  .famille:hover {
    background: var(--hover);
  }
  .ext {
    grid-template-columns: 18px 12px minmax(120px, 1fr) minmax(60px, 1.2fr) 84px 56px 110px;
    min-height: 30px;
    font-size: 12px;
    color: var(--muted);
  }
  .ext .nom {
    grid-column: 3;
  }
  .exts {
    padding-bottom: 8px;
    background: var(--surface-sunken);
  }
  .pli {
    display: flex;
    color: var(--faint);
    transition: transform 0.15s;
  }
  .pli.ouvert {
    transform: rotate(90deg);
    color: var(--accent);
  }
  .pastille {
    width: 12px;
    height: 12px;
    border-radius: 3px;
  }
  .nom {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .mono {
    font-family: var(--font-mono);
  }
  .barre {
    height: 8px;
    border-radius: 4px;
    background: var(--surface-sunken);
    border: 1px solid var(--line-soft);
    overflow: hidden;
  }
  .barre span {
    display: block;
    height: 100%;
    opacity: 0.85;
  }
  .barre.fine {
    height: 5px;
  }
  .num {
    text-align: right;
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
    color: var(--muted);
  }
  .fort {
    color: var(--text);
    font-weight: 600;
  }
  .discret {
    color: var(--faint);
    font-size: 12px;
  }
  .plus {
    padding: 4px 52px;
  }

  @container (max-width: 640px) {
    .famille,
    .ext {
      grid-template-columns: 18px 12px minmax(0, 1fr) 80px 52px;
    }
    .barre,
    .discret {
      display: none;
    }
  }
</style>
