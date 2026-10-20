<script>
  // Logo d'Aria Scope : le A glitché de la famille Aria, avec le viseur orange.
  // Pendant une analyse, le viseur pulse. Chaque instance a ses propres
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
  <g class="viseur" filter="url(#{ref('g')})" stroke="#FF8A1F" stroke-linecap="round" fill="none">
    <circle cx="80" cy="106" r="13" stroke-width="3" />
    <path d="M80,88 v9 M80,115 v9 M62,106 h9 M89,106 h9" stroke-width="2.6" />
    <circle cx="80" cy="106" r="3.4" fill="#FF8A1F" stroke="none" />
  </g>
</svg>

<style>
  svg {
    display: block;
    flex: none;
  }
  .viseur {
    transform-origin: 80px 106px;
  }
  .actif .viseur {
    animation: viser 1.6s ease-in-out infinite;
  }
  @keyframes viser {
    0%,
    100% {
      transform: scale(1) rotate(0deg);
      opacity: 1;
    }
    50% {
      transform: scale(1.12) rotate(45deg);
      opacity: 0.75;
    }
  }
</style>
