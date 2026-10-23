<script>
  // Bilan de la dernière opération de ménage, au-dessus de l'analyse.
  import Icon from './Icon.svelte'
  import { app, relancer } from './state.svelte.js'
  import { octets, nombre } from './format.js'

  let { mode } = $props()
  let details = $state(false)

  const b = $derived(app.bilan)
  const verbe = { corbeille: 'envoyé{s} à la corbeille', deplacer: 'déplacé{s}', copier: 'copié{s}' }
  const fait = (n) => (verbe[b?.operation] ?? 'traité{s}').replace('{s}', n > 1 ? 's' : '')
  // L'espace n'est libéré que par la corbeille ; un déplacement le libère sur ce disque-ci.
  const suffixe = $derived(b?.operation === 'corbeille' ? ' libérés' : b?.operation === 'deplacer' ? ' déplacés' : ' copiés')
</script>

{#if b}
  <div class="bilan" class:mitige={b.restants?.length > 0 || b.erreur} role="status">
    <span class="icone"><Icon name={b.faits > 0 ? 'check' : 'warn'} size={18} /></span>
    <div class="texte">
      <span class="titre">
        {#if b.faits > 0}
          <strong>{nombre(b.faits)} élément{b.faits > 1 ? 's' : ''}</strong> {fait(b.faits)}{#if b.destination}{' vers '}<span class="dest"><bdi>{b.destination}</bdi></span>{/if}
          · <strong>{octets(mode === 'disque' ? b.disque : b.taille)}</strong>{suffixe}
        {:else}
          Aucun élément n'a été {fait(1)}
        {/if}
      </span>
      {#if b.annule}<span class="detail">Opération interrompue.</span>{/if}
      {#if b.perime}
        <span class="detail">
          L'arrivée fait partie de l'analyse : ce qui y est arrivé n'apparaîtra qu'après une nouvelle analyse.
          <button class="lien" onclick={relancer}>Relancer l'analyse</button>
        </span>
      {/if}
      {#if b.erreur}<span class="detail err">Windows : {b.erreur}</span>{/if}
      {#if b.restants?.length > 0}
        <button class="lien" onclick={() => (details = !details)}>
          {nombre(b.restants.length)} élément{b.restants.length > 1 ? 's' : ''} laissé{b.restants.length > 1 ? 's' : ''} en place {details ? '▲' : '▼'}
        </button>
        {#if details}
          <ul>
            {#each b.restants.slice(0, 20) as r (r.chemin)}<li><bdi>{r.chemin}</bdi> : {r.raison}</li>{/each}
            {#if b.restants.length > 20}<li>… et {b.restants.length - 20} autres</li>{/if}
          </ul>
        {/if}
      {/if}
    </div>
    <button class="fermer" aria-label="Fermer" onclick={() => (app.bilan = null)}>×</button>
  </div>
{/if}

<style>
  .bilan {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    padding: 12px 14px;
    border: 1px solid color-mix(in srgb, var(--accent) 50%, transparent);
    border-radius: 10px 0;
    corner-shape: bevel;
    background: color-mix(in srgb, var(--accent) 7%, transparent);
  }
  .bilan.mitige {
    border-color: color-mix(in srgb, var(--warn) 50%, transparent);
    background: color-mix(in srgb, var(--warn) 7%, transparent);
  }
  .icone {
    flex: none;
    display: flex;
    color: var(--accent);
  }
  .mitige .icone {
    color: var(--warn);
  }
  .texte {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: 13px;
    color: var(--muted);
  }
  .titre strong {
    color: var(--text);
  }
  .detail {
    font-size: 12px;
  }
  .dest {
    color: var(--accent);
    overflow-wrap: anywhere;
  }
  .err {
    color: var(--danger);
  }
  .lien {
    align-self: flex-start;
    padding: 0;
    border: none;
    background: none;
    color: var(--warn);
    font-family: var(--font-mono);
    font-size: 12px;
  }
  ul {
    margin: 0;
    padding-left: 18px;
    font-size: 12px;
    overflow-wrap: anywhere;
  }
  .fermer {
    flex: none;
    background: none;
    border: none;
    color: var(--muted);
    font-size: 18px;
    line-height: 1;
  }
</style>
