<script>
  // Icônes au trait, toutes en 24×24.
  let { name, size = 20 } = $props()
</script>

<svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
  {#if name === 'home'}
    <path d="M4 11l8-7 8 7" /><path d="M6 10v10h12V10" /><path d="M10 20v-5h4v5" />
  {:else if name === 'scope'}
    <circle cx="12" cy="12" r="7" /><path d="M12 2v4" /><path d="M12 18v4" /><path d="M2 12h4" /><path d="M18 12h4" /><circle cx="12" cy="12" r="1.5" fill="currentColor" stroke="none" />
  {:else if name === 'disk'}
    <rect x="3" y="5" width="18" height="14" rx="2" /><path d="M3 14h18" /><path d="M7 17h.01" /><path d="M11 17h.01" />
  {:else if name === 'usb'}
    <rect x="7" y="9" width="10" height="12" rx="2" /><path d="M9 9V4h6v5" /><path d="M11 6h.01" /><path d="M13 6h.01" />
  {:else if name === 'network'}
    <rect x="4" y="3" width="16" height="8" rx="2" /><path d="M12 11v5" /><path d="M5 20h14" /><circle cx="12" cy="18" r="2" />
  {:else if name === 'disc'}
    <circle cx="12" cy="12" r="9" /><circle cx="12" cy="12" r="2.5" />
  {:else if name === 'folder'}
    <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />
  {:else if name === 'folder-search'}
    <path d="M11 19H5a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v2" /><circle cx="17" cy="16" r="3" /><path d="M21 20l-1.8-1.8" />
  {:else if name === 'files'}
    <path d="M8 4h7l4 4v10a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2z" /><path d="M15 4v4h4" /><path d="M3 8v11a2 2 0 0 0 2 2h9" />
  {:else if name === 'file'}
    <path d="M7 3h7l5 5v11a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2z" /><path d="M14 3v5h5" />
  {:else if name === 'more'}
    <circle cx="6" cy="12" r="1" fill="currentColor" /><circle cx="12" cy="12" r="1" fill="currentColor" /><circle cx="18" cy="12" r="1" fill="currentColor" />
  {:else if name === 'chevron'}
    <path d="M9 6l6 6-6 6" />
  {:else if name === 'external'}
    <path d="M14 4h6v6" /><path d="M20 4l-9 9" /><path d="M18 14v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h5" />
  {:else if name === 'stop'}
    <rect x="6" y="6" width="12" height="12" rx="1.5" />
  {:else if name === 'refresh'}
    <path d="M20 11a8 8 0 1 0-2.3 5.7" /><path d="M20 4v7h-7" />
  {:else if name === 'map'}
    <rect x="3" y="4" width="18" height="16" rx="1.5" /><path d="M11 4v16" /><path d="M11 13h10" /><path d="M16 13v7" />
  {:else if name === 'up'}
    <path d="M12 19V5" /><path d="M6 11l6-6 6 6" />
  {:else if name === 'filter'}
    <path d="M4 5h16l-6 7.5V19l-4 2v-8.5z" />
  {:else if name === 'trash'}
    <path d="M4 7h16" /><path d="M9 7V4h6v3" /><path d="M6 7l1 13h10l1-13" /><path d="M10 11v6" /><path d="M14 11v6" />
  {:else if name === 'move'}
    <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v3" /><path d="M3 7v10a2 2 0 0 0 2 2h7" /><path d="M15 17h7" /><path d="M19 14l3 3-3 3" />
  {:else if name === 'copy'}
    <rect x="8" y="8" width="12" height="12" rx="2" /><path d="M16 8V6a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h2" />
  {:else if name === 'shield'}
    <path d="M12 3l8 3v6c0 4.5-3.4 8-8 9-4.6-1-8-4.5-8-9V6z" />
  {:else if name === 'check'}
    <path d="M5 12.5l4.5 4.5L19 7.5" />
  {:else if name === 'warn'}
    <path d="M12 3l10 18H2z" /><path d="M12 10v5" /><path d="M12 18h.01" />
  {/if}
</svg>
