<script>
  // Choix de la portée des statistiques : toute l'analyse, ou le dossier
  // sélectionné (pour des fichiers, celui qui les contient).
  import { call } from './state.svelte.js'

  let { gen, selection, portee = $bindable('tout') } = $props()

  const idSelection = $derived(selection ? selection.id : 0)
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
</script>

<div class="portee" role="radiogroup" aria-label="Portée">
  <button role="radio" aria-checked={portee === 'tout'} class:on={portee === 'tout'} onclick={() => (portee = 'tout')}>toute l'analyse</button>
  <button
    role="radio"
    aria-checked={portee === 'selection'}
    class:on={portee === 'selection'}
    disabled={!idSelection}
    onclick={() => (portee = 'selection')}
    title={idSelection ? nomSelection : "Sélectionne un dossier dans l'arbre ou sur la carte"}
  >
    {idSelection ? `dans ${nomSelection || '…'}` : 'dossier sélectionné'}
  </button>
</div>

<style>
  .portee {
    display: inline-flex;
    min-width: 0;
    padding: 3px;
    border: 1px solid var(--line-strong);
    border-radius: 10px 0;
    corner-shape: bevel;
  }
  button {
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
  button.on {
    background: color-mix(in srgb, var(--accent) 14%, transparent);
    color: var(--accent);
  }
  button:disabled {
    opacity: 0.45;
  }
</style>
