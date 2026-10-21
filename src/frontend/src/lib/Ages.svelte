<script>
  // Répartition par âge : les fichiers rangés selon leur date de dernière
  // modification, du plus récent au plus ancien.
  import Portee from './Portee.svelte'
  import { call, message } from './state.svelte.js'
  import { octets, nombre, pourcent, pourcentTexte } from './format.js'

  let { gen, mode, selection, portee = $bindable('tout') } = $props()

  // Du cyan (récent) à l'orange puis au rouge (ancien) ; gris pour « inconnue »
  const COULEURS = ['#7df9ff', '#5cc8ff', '#5c8bff', '#b18cff', '#ff8a1f', '#ff5c7a', '#3b5378']

  let ages = $state(null)
  let erreur = $state('')
  let chargement = $state(false)

  const id = $derived(portee === 'selection' && selection ? selection.id : 0)

  $effect(() => {
    const g = gen
    const i = id
    chargement = true
    erreur = ''
    call('Ages', g, i)
      .then((a) => {
        if (g === gen && i === id) ages = a ?? []
      })
      .catch((err) => (erreur = message(err)))
      .finally(() => (chargement = false))
  })

  const val = (x) => (mode === 'disque' ? x.disque : x.taille)
  const total = $derived((ages ?? []).reduce((s, a) => s + val(a), 0))
  const max = $derived(Math.max(0, ...(ages ?? []).map(val)))

  // Ce qui n'a pas bougé depuis plus d'un an (les tranches 1 à 2 ans et au-delà)
  const dormant = $derived((ages ?? []).slice(3, 6).reduce((s, a) => s + val(a), 0))
</script>

<div class="ages">
  <div class="tete">
    <Portee {gen} {selection} bind:portee />
    {#if chargement}<span class="hint">lecture…</span>{/if}
  </div>

  {#if erreur}
    <p class="error">{erreur}</p>
  {:else if ages && total === 0}
    <p class="hint vide">Aucun fichier ici.</p>
  {:else if ages}
    <p class="constat">
      <strong>{octets(dormant)}</strong>, soit {pourcentTexte(pourcent(dormant, total))} de l'espace, n'ont pas été modifiés depuis plus d'un an.
    </p>
    <ul class="liste">
      {#each ages as a, i (a.nom)}
        {@const couleur = COULEURS[i] ?? COULEURS.at(-1)}
        <li class="tranche">
          <span class="pastille" style:background={couleur} style:box-shadow="0 0 8px {couleur}"></span>
          <span class="nom">{a.nom}</span>
          <span class="barre"><span style:width="{pourcent(val(a), max)}%" style:background={couleur}></span></span>
          <span class="num fort">{octets(val(a))}</span>
          <span class="num">{pourcentTexte(pourcent(val(a), total))}</span>
          <span class="num discret">{nombre(a.fichiers)} fichier{a.fichiers > 1 ? 's' : ''}</span>
        </li>
      {/each}
    </ul>
    <p class="hint note">D'après la date de dernière modification de chaque fichier, au moment de l'analyse.</p>
  {/if}
</div>

<style>
  .ages {
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
  .vide {
    padding: 16px;
  }
  .constat {
    margin: 0;
    padding: 14px 16px 6px;
    font-size: 13px;
    line-height: 1.5;
    color: var(--muted);
  }
  .constat strong {
    color: var(--action);
  }
  .liste {
    margin: 0;
    padding: 4px 0;
    list-style: none;
  }
  .tranche {
    display: grid;
    grid-template-columns: 12px minmax(110px, 0.8fr) minmax(80px, 2fr) 84px 56px 110px;
    align-items: center;
    gap: 10px;
    min-height: 40px;
    padding: 0 16px;
    font-size: 13px;
  }
  .tranche + .tranche {
    border-top: 1px solid var(--line-soft);
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
  .barre {
    height: 10px;
    border-radius: 5px;
    background: var(--surface-sunken);
    border: 1px solid var(--line-soft);
    overflow: hidden;
  }
  .barre span {
    display: block;
    height: 100%;
    opacity: 0.85;
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
  .note {
    padding: 8px 16px 14px;
  }

  /* Colonne étroite : on garde les barres, on retire le nombre de fichiers */
  @container (max-width: 640px) {
    .tranche {
      grid-template-columns: 12px minmax(96px, 1fr) minmax(50px, 1fr) 76px 48px;
      padding: 0 12px;
    }
    .discret {
      display: none;
    }
  }
  @container (max-width: 400px) {
    .tranche {
      grid-template-columns: 12px minmax(0, 1fr) 76px 48px;
    }
    .barre {
      display: none;
    }
  }
</style>
