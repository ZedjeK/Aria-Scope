<script>
  // Édition des filtres. Avec une racine, « Appliquer » relance l'analyse ;
  // sans (page des disques), les filtres valent pour la prochaine analyse.
  import Icon from './Icon.svelte'
  import { app, lancer } from './state.svelte.js'
  import { TAILLES, AGES, decouper, joindre, filtresVides, actifs } from './filtres.js'

  let { racine = null, onfermer } = $props()

  // Brouillon : modifié librement, appliqué seulement sur demande.
  let exclure = $state(joindre(app.filtres.exclure))
  let inclure = $state(joindre(app.filtres.inclure))
  let regex = $state(app.filtres.regex)
  let tailleMin = $state(app.filtres.tailleMin)
  let ageMinJours = $state(app.filtres.ageMinJours)
  let occupe = $state(false)

  const brouillon = $derived({ exclure: decouper(exclure), inclure: decouper(inclure), regex: regex.trim(), tailleMin, ageMinJours })

  async function appliquer(f) {
    const precedents = app.filtres
    app.filtres = f
    if (!racine) {
      onfermer?.()
      return
    }
    occupe = true
    const ok = await lancer(racine)
    occupe = false
    if (ok) onfermer?.()
    else app.filtres = precedents // regex invalide, motif refusé… : on garde le panneau ouvert
  }

  function effacer() {
    exclure = inclure = regex = ''
    tailleMin = ageMinJours = 0
    appliquer(filtresVides())
  }
</script>

<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<form class="card panneau" onsubmit={(e) => (e.preventDefault(), appliquer(brouillon))} onkeydown={(e) => e.key === 'Escape' && onfermer?.()}>
  <div class="card-head">
    <h2>filtres</h2>
    <button type="button" class="fermer" onclick={() => onfermer?.()} aria-label="Fermer">×</button>
  </div>
  <p class="card-text">
    Les filtres s'appliquent pendant l'analyse : l'arbre, la carte, les gros fichiers et les statistiques ne comptent que ce qui passe.
  </p>

  <div class="champs">
    <label class="field">
      <span class="field-label">Exclure</span>
      <input bind:value={exclure} placeholder="node_modules, $RECYCLE.BIN, *.tmp" spellcheck="false" />
      <span class="hint">Dossiers ou fichiers ignorés complètement. Un dossier exclu n'est même pas parcouru.</span>
    </label>
    <label class="field">
      <span class="field-label">Seulement</span>
      <input bind:value={inclure} placeholder="*.mkv, *.mp4, *.iso" spellcheck="false" />
      <span class="hint">Si rempli, seuls ces fichiers sont comptés. Les dossiers qui n'en ont pas disparaissent.</span>
    </label>
    <label class="field">
      <span class="field-label">Nom contenant (regex)</span>
      <input bind:value={regex} placeholder="^IMG_\d+|sauvegarde" spellcheck="false" />
      <span class="hint">Expression régulière, sans tenir compte des majuscules.</span>
    </label>
    <div class="ligne-2">
      <label class="field">
        <span class="field-label">Taille</span>
        <select bind:value={tailleMin}>
          {#each TAILLES as [v, texte] (v)}<option value={v}>{texte}</option>{/each}
        </select>
      </label>
      <label class="field">
        <span class="field-label">Modifiés il y a</span>
        <select bind:value={ageMinJours}>
          {#each AGES as [v, texte] (v)}<option value={v}>{texte}</option>{/each}
        </select>
      </label>
    </div>
  </div>

  <div class="actions">
    <button type="submit" class="btn-primary" disabled={occupe}>
      <Icon name={racine ? 'refresh' : 'filter'} size={16} />
      {racine ? (occupe ? 'Relance…' : 'Appliquer et relancer') : 'Garder pour la prochaine analyse'}
    </button>
    <button type="button" class="btn-ghost" disabled={occupe || (!actifs(brouillon) && !actifs(app.filtres))} onclick={effacer}>Effacer les filtres</button>
  </div>
</form>

<style>
  .panneau {
    gap: 14px;
  }
  .fermer {
    background: none;
    border: none;
    color: var(--muted);
    font-size: 20px;
    line-height: 1;
    padding: 0 4px;
  }
  .fermer:hover {
    color: var(--text);
  }
  .champs {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }
  .ligne-2 {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 14px;
  }
  select {
    width: 100%;
    min-height: 40px;
    padding: 0 10px;
    border-radius: 8px 0;
    corner-shape: bevel;
    border: 1px solid var(--line-strong);
    background: var(--surface-sunken);
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 13px;
  }
  select:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: 1px;
  }
  .actions button {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }
</style>
