<script>
  // Doublons vérifiés : regroupés par taille, puis empreinte rapide, puis
  // SHA-256 complet. Un clic coche ou décoche une copie ; au moins une copie
  // de chaque groupe reste toujours non cochée.
  import { onMount, onDestroy } from 'svelte'
  import Icon from './Icon.svelte'
  import Portee from './Portee.svelte'
  import { app, call, message, montrer } from './state.svelte.js'
  import { octets, nombre, dateCourte, dateLongue, duree } from './format.js'
  import { cibleDe, cleMarque } from './selection.js'

  let { gen, mode, selection = $bindable(null) } = $props()

  const UNITES = { Ko: 1024, Mo: 1024 ** 2, Go: 1024 ** 3 }
  const PAS = 50 // groupes affichés à la fois
  const PHASES = {
    inventaire: 'Inventaire des fichiers',
    rapide: 'Empreintes rapides (début et fin)',
    complete: 'Vérification SHA-256 complète',
  }

  const c = app.critDoublons
  let suivi = $state(null)
  let groupes = $state(null)
  let erreur = $state('')
  let visibles = $state(PAS)
  let refus = $state(null) // groupe où l'on a voulu cocher la dernière copie
  let sondage

  onMount(() => suivre().catch((err) => (erreur = message(err))))
  onDestroy(() => clearInterval(sondage))

  const enCours = $derived(suivi?.etat === 'scan')
  const tailleMin = $derived(Math.max(1, Math.round((Number(c.min) || 0) * UNITES[c.unite])))

  async function chercher() {
    erreur = ''
    const dans = c.portee === 'selection' && selection ? selection.id : 0
    try {
      await call('ChercherDoublons', gen, dans, tailleMin)
      groupes = null
      visibles = PAS
      await suivre()
    } catch (err) {
      erreur = message(err)
    }
  }

  async function suivre() {
    clearInterval(sondage)
    const lire = async () => {
      const s = await call('SuiviDoublons')
      if (!s?.gen || s.gen !== gen) {
        suivi = null
        clearInterval(sondage)
        return
      }
      suivi = s
      if (s.etat !== 'scan') {
        clearInterval(sondage)
        await charger()
      }
    }
    await lire()
    if (suivi?.etat === 'scan') sondage = setInterval(() => lire().catch((err) => (erreur = message(err))), 300)
  }

  async function charger() {
    try {
      groupes = (await call('GroupesDoublons', gen)) ?? []
      // Pour le garde-fou de la barre de ménage, valable dans tous les onglets
      app.doublons = {
        gen,
        groupes: groupes.map((g) => ({ nom: g.fichiers[0].nom, cles: g.fichiers.map((f) => cleMarque(cibleDe(f))) })),
      }
    } catch (err) {
      erreur = message(err)
    }
  }

  // Après un ménage, les groupes se mettent à jour.
  let premier = true
  $effect(() => {
    app.revision
    if (premier) {
      premier = false
      return
    }
    if (suivi && suivi.etat !== 'scan') charger()
  })

  const arreter = () => call('ArreterDoublons').catch((err) => (erreur = message(err)))

  // ---------- Coches ----------
  const marquees = $derived(new Set(app.marques.map(cleMarque)))
  const cle = (f) => cleMarque(cibleDe(f))
  const coche = (f) => marquees.has(cle(f))

  function basculer(g, f) {
    refus = null
    const k = cle(f)
    const i = app.marques.findIndex((m) => cleMarque(m) === k)
    if (i >= 0) {
      app.marques.splice(i, 1)
      return
    }
    const restantes = g.fichiers.filter((x) => x !== f && !coche(x)).length
    if (restantes === 0) {
      refus = g.empreinte
      return
    }
    app.marques.push(cibleDe(f))
    app.ancre = cibleDe(f)
  }

  // Présélection : dans chaque groupe concerné, garder ce que choisit garder()
  // (au moins une copie) et cocher le reste.
  function preselectionner(garder) {
    refus = null
    const parCle = new Map(app.marques.map((m) => [cleMarque(m), m]))
    for (const g of groupes ?? []) {
      const gardees = garder(g)
      if (!gardees || gardees.length === 0 || gardees.length === g.fichiers.length) continue
      for (const f of g.fichiers) {
        if (gardees.includes(f)) parCle.delete(cle(f))
        else parCle.set(cle(f), cibleDe(f))
      }
    }
    app.marques = [...parCle.values()]
  }
  const garderAncien = () => preselectionner((g) => [g.fichiers[0]])
  const garderRecent = () => preselectionner((g) => [g.fichiers[g.fichiers.length - 1]])
  const garderDossier = (dossier) => preselectionner((g) => g.fichiers.filter((f) => f.dossier === dossier))

  const total = $derived.by(() => {
    let copies = 0
    let place = 0
    for (const g of groupes ?? []) {
      copies += g.fichiers.length - 1
      place += g.recuperable
    }
    return { copies, place }
  })
  const progression = $derived.by(() => {
    if (!suivi) return 0
    if (suivi.phase === 'inventaire') return suivi.dossiers > 0 ? suivi.lus / suivi.dossiers : 0
    return suivi.aHacher > 0 ? Math.min(1, suivi.haches / suivi.aHacher) : 0
  })
</script>

<form class="card criteres" onsubmit={(e) => (e.preventDefault(), chercher())}>
  <p class="card-text">
    Les fichiers de même taille sont comparés par une empreinte rapide, puis vérifiés par un SHA-256 complet : un doublon annoncé est identique octet pour octet.
    Deux noms pour un même fichier (lien physique) ne comptent pas.
  </p>
  <div class="rangee">
    <label class="field taille">
      <span class="field-label">Taille minimale</span>
      <div class="duo">
        <input type="number" min="0" step="any" bind:value={c.min} aria-label="Taille minimale" />
        <select bind:value={c.unite} aria-label="Unité">{#each Object.keys(UNITES) as u (u)}<option>{u}</option>{/each}</select>
      </div>
    </label>
    <Portee {gen} {selection} bind:portee={c.portee} />
    <div class="actions">
      {#if enCours}
        <button type="button" class="btn-primary" onclick={arreter}><Icon name="stop" size={16} /> Arrêter</button>
      {:else}
        <button type="submit" class="btn-primary"><Icon name="copy" size={16} /> Chercher les doublons</button>
      {/if}
    </div>
  </div>
</form>

{#if erreur}
  <p class="error">{erreur}</p>
{/if}

{#if suivi && enCours}
  <div class="etat" aria-live="polite">
    <div class="etat-tete"><strong>{PHASES[suivi.phase] ?? '…'}</strong><span>{Math.floor(progression * 100)} %</span></div>
    <div class="jauge"><span style="width: {progression * 100}%"></span></div>
    <span class="hint">
      {#if suivi.phase === 'inventaire'}
        {nombre(suivi.lus)} / {nombre(suivi.dossiers)} dossiers relus · {nombre(suivi.candidats)} fichiers retenus
      {:else}
        {octets(suivi.haches)} lus sur {octets(suivi.aHacher)}
      {/if}
    </span>
  </div>
{:else if suivi && groupes}
  {#if suivi.etat === 'arrete'}
    <p class="hint">Recherche arrêtée avant la fin : rien n'est annoncé tant que la vérification n'est pas complète.</p>
  {:else if groupes.length === 0}
    <p class="hint ok"><Icon name="check" size={14} /> Aucun doublon de {octets(suivi.tailleMin)} ou plus. Durée : {duree(suivi.dureeMs)}.</p>
  {:else}
    <div class="synthese">
      <p class="hint resume">
        <strong>{nombre(groupes.length)}</strong> groupe{groupes.length > 1 ? 's' : ''} · <strong>{nombre(total.copies)}</strong> copie{total.copies > 1 ? 's' : ''} en trop ·
        <strong class="accent">{octets(total.place)}</strong> récupérables · {duree(suivi.dureeMs)}
        {#if suivi.illisibles > 0}· {nombre(suivi.illisibles)} illisible{suivi.illisibles > 1 ? 's' : ''}{/if}
      </p>
      <div class="presel">
        <span class="eyebrow">cocher les copies en trop, en gardant dans chaque groupe :</span>
        <button class="btn-ghost small" onclick={garderAncien} title="Dans chaque groupe, garder le fichier modifié le plus tôt (à date égale, le premier chemin par ordre alphabétique)">le plus ancien</button>
        <button class="btn-ghost small" onclick={garderRecent} title="Dans chaque groupe, garder le fichier modifié le plus récemment">le plus récent</button>
      </div>
    </div>

    <ol class="groupes">
      {#each groupes.slice(0, visibles) as g (g.empreinte)}
        <li class="groupe">
          <div class="g-tete">
            <span><strong>{nombre(g.fichiers.length)} copies</strong> · {octets(g.taille)} chacune · <strong class="accent">{octets(g.recuperable)}</strong> récupérables</span>
            <span class="empreinte" title="SHA-256 : {g.empreinte}">SHA-256 {g.empreinte.slice(0, 12)}…</span>
          </div>
          {#if refus === g.empreinte}
            <p class="refus"><Icon name="shield" size={13} /> Au moins une copie reste toujours non cochée.</p>
          {/if}
          {#each g.fichiers as f, i (cle(f))}
            {@const oui = coche(f)}
            <div
              class="copie"
              class:cochee={oui}
              role="checkbox"
              tabindex="0"
              aria-checked={oui}
              onclick={() => basculer(g, f)}
              onkeydown={(e) => (e.key === ' ' || e.key === 'Enter') && (e.preventDefault(), basculer(g, f))}
              ondblclick={() => montrer(gen, f.id, f.nom)}
            >
              <span class="case" aria-hidden="true">{#if oui}<Icon name="check" size={13} />{/if}</span>
              <span class="c-fichier">
                <span class="c-nom" title={f.nom}>{f.nom}</span>
                <span class="c-dossier" title={f.dossier}><bdi>{f.dossier}</bdi></span>
              </span>
              <span class="c-modif" title="Modifié le {dateLongue(f.modif)}">{dateCourte(f.modif)}{#if i === 0 && f.modif < g.fichiers[g.fichiers.length - 1].modif}<span class="age">plus ancien</span>{/if}</span>
              <span class="c-act">
                <button class="mini" title="Garder les copies de ce dossier (dans tous les groupes)" aria-label="Garder les copies de ce dossier" onclick={(e) => (e.stopPropagation(), garderDossier(f.dossier))}>
                  <Icon name="shield" size={14} />
                </button>
                <button class="mini" title="Montrer dans l'Explorateur" aria-label="Montrer dans l'Explorateur" onclick={(e) => (e.stopPropagation(), montrer(gen, f.id, f.nom))}>
                  <Icon name="external" size={14} />
                </button>
              </span>
            </div>
          {/each}
        </li>
      {/each}
    </ol>
    {#if groupes.length > visibles}
      <button class="btn-ghost plus" onclick={() => (visibles += PAS)}>Afficher {nombre(Math.min(PAS, groupes.length - visibles))} groupes de plus ({nombre(groupes.length - visibles)} restants)</button>
    {/if}
  {/if}
{/if}

<style>
  .criteres {
    gap: 12px;
  }
  .rangee {
    display: flex;
    align-items: flex-end;
    gap: 12px;
    flex-wrap: wrap;
  }
  .taille {
    flex: 0 1 220px;
  }
  .duo {
    display: flex;
    gap: 6px;
  }
  .duo input {
    flex: 1 1 0;
    min-width: 0;
  }
  select {
    flex: none;
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
  .actions {
    margin-left: auto;
  }
  .actions button {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }
  .etat {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .etat-tete {
    display: flex;
    justify-content: space-between;
    font-size: 13px;
    color: var(--text);
    font-variant-numeric: tabular-nums;
  }
  .jauge {
    height: 6px;
    border-radius: 3px;
    background: var(--line);
    overflow: hidden;
  }
  .jauge span {
    display: block;
    height: 100%;
    background: var(--action);
    box-shadow: 0 0 10px var(--action);
    transition: width 0.25s linear;
  }
  .ok {
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--accent);
  }
  .synthese {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .resume {
    margin: 0;
  }
  .resume strong {
    color: var(--text);
  }
  .accent {
    color: var(--accent) !important;
  }
  .presel {
    display: flex;
    align-items: center;
    gap: 6px 8px;
    flex-wrap: wrap;
  }
  .presel .eyebrow {
    flex-basis: 100%;
  }
  .presel button {
    min-height: 32px;
    padding: 0 10px;
  }
  .groupes {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .groupe {
    container-type: inline-size;
    background: var(--surface);
    border: 1px solid var(--line);
    border-radius: 12px 0;
    corner-shape: bevel;
    overflow: clip;
  }
  .g-tete {
    display: flex;
    justify-content: space-between;
    gap: 10px;
    flex-wrap: wrap;
    padding: 8px 12px;
    background: var(--surface-sunken);
    font-size: 12px;
    color: var(--muted);
  }
  .g-tete strong {
    color: var(--text);
  }
  .empreinte {
    font-size: 11px;
    color: var(--dim);
  }
  .refus {
    display: flex;
    align-items: center;
    gap: 6px;
    margin: 0;
    padding: 6px 12px;
    font-size: 12px;
    color: var(--warn);
  }
  .copie {
    display: grid;
    grid-template-columns: 26px minmax(0, 1fr) 110px 64px;
    align-items: center;
    gap: 8px;
    min-height: 44px;
    padding-top: 4px;
    padding-bottom: 4px;
    padding: 0 8px 0 10px;
    border-top: 1px solid var(--line-soft);
    font-size: 13px;
    cursor: pointer;
    user-select: none;
  }
  .copie:hover {
    background: var(--hover);
  }
  .copie:focus-visible {
    outline: 2px solid var(--accent);
    outline-offset: -2px;
  }
  .copie.cochee {
    background: color-mix(in srgb, var(--danger) 9%, transparent);
  }
  .copie.cochee .c-nom {
    color: var(--danger);
    text-decoration: line-through;
    text-decoration-color: color-mix(in srgb, var(--danger) 60%, transparent);
  }
  .case {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 18px;
    height: 18px;
    border: 1px solid var(--line-strong);
    border-radius: 4px;
    color: var(--on-action);
  }
  .cochee .case {
    border-color: var(--danger);
    background: var(--danger);
  }
  .c-fichier {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .c-nom {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--text);
  }
  .c-dossier {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    direction: rtl;
    text-align: left;
    font-size: 12px;
    color: var(--faint);
  }
  .c-modif {
    display: flex;
    flex-direction: column;
    font-size: 12px;
    color: var(--muted);
    font-variant-numeric: tabular-nums;
  }
  .age {
    font-size: 10px;
    letter-spacing: 0.5px;
    color: var(--accent);
  }
  .c-act {
    display: flex;
    justify-content: flex-end;
    gap: 2px;
  }
  .mini {
    display: flex;
    padding: 5px;
    border: 1px solid transparent;
    border-radius: 6px 0;
    corner-shape: bevel;
    background: none;
    color: var(--dim);
  }
  .mini:hover {
    color: var(--action);
    border-color: var(--line-strong);
  }
  .plus {
    align-self: center;
  }
</style>
