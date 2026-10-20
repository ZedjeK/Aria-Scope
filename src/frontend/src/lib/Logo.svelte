<script>
  // Logo d'Aria Scope : le A glitché de la famille Aria, avec les jumelles orange.
  // Pendant une analyse, les jumelles balaient l'horizon. Chaque instance a ses propres
  // identifiants SVG (découpes, filtres).
  let { size = 36, actif = false, label = '' } = $props()

  const id = `logo${Math.random().toString(36).slice(2, 10)}`
  const ref = (name) => `${id}-${name}`
</script>

<svg
  width={size}
  height={size}
  viewBox="-8 -8 176 176"
  class:actif
  role={label ? 'img' : undefined}
  aria-label={label || undefined}
  aria-hidden={label ? undefined : 'true'}
>
  <defs>
    <pattern id={ref('s')} width="4" height="4" patternUnits="userSpaceOnUse">
      <rect width="4" height="1.5" fill="#7DF9FF" opacity="0.07" />
    </pattern>
    <polygon id={ref('a')} points="30,130 70,30 90,30 130,130 110,130 80,54 50,130" />
    <clipPath id={ref('bl')}><rect x="0" y="72" width="80" height="90" /></clipPath>
    <clipPath id={ref('br')}><rect x="80" y="72" width="80" height="90" /></clipPath>
    <clipPath id={ref('t')}><rect x="0" y="0" width="160" height="72" /></clipPath>
    <filter id={ref('g')} x="-40%" y="-40%" width="180%" height="180%">
      <feGaussianBlur stdDeviation="3.2" result="b" />
      <feMerge><feMergeNode in="b" /><feMergeNode in="b" /><feMergeNode in="SourceGraphic" /></feMerge>
    </filter>
    <filter id={ref('g2')} x="-20%" y="-20%" width="140%" height="140%">
      <feGaussianBlur stdDeviation="2" result="b" />
      <feMerge><feMergeNode in="b" /><feMergeNode in="SourceGraphic" /></feMerge>
    </filter>
  </defs>

  <polygon points="24,0 136,0 160,24 160,136 136,160 24,160 0,136 0,24" fill="#030812" />
  <polygon points="24,0 136,0 160,24 160,136 136,160 24,160 0,136 0,24" fill="url(#{ref('s')})" />
  <polygon
    points="26,7 134,7 153,26 153,134 134,153 26,153 7,134 7,26"
    fill="none"
    stroke="#4D7CFF"
    stroke-width="1.6"
    opacity="0.6"
    filter="url(#{ref('g2')})"
  />
  <g filter="url(#{ref('g')})" stroke-linejoin="round">
    <g clip-path="url(#{ref('bl')})" fill="none" stroke="#7DF9FF" stroke-width="4"><use href="#{ref('a')}" /></g>
    <g clip-path="url(#{ref('br')})" fill="none" stroke="#5C8BFF" stroke-width="4"><use href="#{ref('a')}" /></g>
    <g transform="translate(7,0)">
      <g clip-path="url(#{ref('t')})" fill="none" stroke="#FFFFFF" stroke-width="4"><use href="#{ref('a')}" /></g>
    </g>
  </g>
  <g class="jumelles" filter="url(#{ref('g')})">
    <g transform="translate(80 104) scale(0.95) translate(-80 -104)">
      <g fill="#FF8A1F">
        <rect x="66" y="87" width="8" height="7" rx="1.5" /><path d="M64,94 h12 l2,6 v17 q0,4 -4,4 h-12 q-4,0 -4,-4 v-13 z" />
        <g transform="translate(160 0) scale(-1 1)"><rect x="66" y="87" width="8" height="7" rx="1.5" /><path d="M64,94 h12 l2,6 v17 q0,4 -4,4 h-12 q-4,0 -4,-4 v-13 z" /></g>
        <rect x="76" y="97" width="8" height="10" rx="2" />
      </g>
      <g fill="#030812" opacity="0.7">
        <ellipse cx="68" cy="118" rx="6" ry="2.2" /><ellipse cx="92" cy="118" rx="6" ry="2.2" /><circle cx="80" cy="102" r="1.8" />
      </g>
    </g>
  </g>
</svg>

<style>
  svg {
    display: block;
    flex: none;
  }
  .jumelles {
    transform-origin: 80px 104px;
  }
  /* Analyse en cours : les jumelles balaient de gauche à droite */
  .actif .jumelles {
    animation: balayer 2.4s ease-in-out infinite;
  }
  @keyframes balayer {
    0%,
    100% {
      transform: translateX(0) rotate(0deg);
    }
    25% {
      transform: translateX(-4px) rotate(-6deg);
    }
    75% {
      transform: translateX(4px) rotate(6deg);
    }
  }
</style>
