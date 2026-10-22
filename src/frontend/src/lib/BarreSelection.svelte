<script>
  // Barre d'actions de la sélection multiple, collée en bas de la page.
  // Le récapitulatif vient du moteur, qui écarte les doublons (un fichier
  // contenu dans un dossier déjà sélectionné ne compte qu'une fois).
  import Icon from './Icon.svelte'
  import { app, call, message, toutDeselectionner } from './state.svelte.js'
  import { octets, nombre } from './format.js'

  let { gen, mode } = $props()

  let resume = $state(null)
  let erreur = $state('')

  $effect(() => {
    const g = gen
    const cibles = $state.snapshot(app.marques)
    if (cibles.length === 0) {
      resume = null
      return
    }
    call('Resumer', g, cibles)
      .then((r) => {
        if (g === gen) {
          resume = r
          erreur = ''
        }
      })
      .catch((err) => (erreur = message(err)))
  })

  function clavier(evt) {
    if (evt.key === 'Escape' && app.marques.length > 0 && !evt.defaultPrevented) toutDeselectionner()
  }
</script>

<svelte:window onkeydown={clavier} />

{#if app.marques.length > 0}
  <div class="barre" role="region" aria-label="Sélection">
    <span class="compte">{nombre(resume?.elements ?? app.marques.length)}</span>
    <div class="texte">
      <span class="titre">
        élément{(resume?.elements ?? app.marques.length) > 1 ? 's' : ''} sélectionné{(resume?.elements ?? app.marques.length) > 1 ? 's' : ''}
        {#if resume}· <strong>{octets(mode === 'disque' ? resume.disque : resume.taille)}</strong>{/if}
      </span>
      {#if erreur}
        <span class="error">{erreur}</span>
      {:else if resume}
        <span class="detail">
          {#if resume.dossiers > 0}{nombre(resume.dossiers)} dossier{resume.dossiers > 1 ? 's' : ''}{' · '}{/if}{nombre(resume.fichiers)} fichier{resume.fichiers > 1 ? 's' : ''} concerné{resume.fichiers > 1 ? 's' : ''}
          {#if app.marques.length > resume.elements}{' · '}{nombre(app.marques.length - resume.elements)} déjà compris dans un dossier sélectionné{/if}
        </span>
        {#if resume.filtre && resume.dossiers > 0}
          <span class="avert"><Icon name="warn" size={12} /> Analyse filtrée : les dossiers contiennent peut-être davantage que ce qui est compté ici.</span>
        {/if}
      {/if}
    </div>
    <span class="aide">Ctrl+clic : ajouter · Maj+clic : étendre</span>
    <button class="btn-ghost small" onclick={toutDeselectionner} title="Échap">Tout désélectionner</button>
  </div>
{/if}

<style>
  .barre {
    position: sticky;
    bottom: 0;
    z-index: 3;
    display: flex;
    align-items: center;
    gap: 14px;
    margin: 0 -6px -10px;
    padding: 12px 16px;
    background: color-mix(in srgb, var(--bg) 94%, transparent);
    border: 1px solid color-mix(in srgb, var(--action) 55%, transparent);
    border-radius: 12px 0;
    corner-shape: bevel;
    box-shadow: 0 -8px 24px rgba(0, 0, 0, 0.45), 0 0 16px color-mix(in srgb, var(--action) 14%, transparent);
    backdrop-filter: blur(6px);
  }
  .compte {
    flex: none;
    min-width: 40px;
    padding: 6px 10px;
    border-radius: 8px 0;
    corner-shape: bevel;
    background: var(--action);
    color: var(--on-action);
    font-family: var(--font-display);
    font-size: 18px;
    font-weight: 700;
    text-align: center;
  }
  .texte {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .titre {
    font-size: 13px;
    color: var(--text);
  }
  .titre strong {
    color: #ffffff;
  }
  .detail {
    font-size: 12px;
    color: var(--faint);
  }
  .avert {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-size: 12px;
    color: var(--warn);
  }
  .aide {
    flex: none;
    font-size: 11px;
    color: var(--dim);
  }
  @media (max-width: 1100px) {
    .aide {
      display: none;
    }
  }
</style>
