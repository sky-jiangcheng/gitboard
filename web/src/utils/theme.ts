export type ThemeMode = 'light' | 'dark' | 'system'

const STORAGE_KEY = 'gitboard-theme'

function getSystemTheme(): 'light' | 'dark' {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export function getEffectiveTheme(mode: ThemeMode): 'light' | 'dark' {
  if (mode === 'system') return getSystemTheme()
  return mode
}

export function getStoredTheme(): ThemeMode {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw === 'light' || raw === 'dark' || raw === 'system') return raw
  } catch {
    // ignore
  }
  return 'system'
}

export function storeTheme(mode: ThemeMode) {
  try {
    localStorage.setItem(STORAGE_KEY, mode)
  } catch {
    // ignore
  }
}

export function applyTheme(mode: ThemeMode) {
  const effective = getEffectiveTheme(mode)
  document.documentElement.setAttribute('data-theme', effective)
}

export function listenSystemTheme(callback: (theme: 'light' | 'dark') => void) {
  const mql = window.matchMedia('(prefers-color-scheme: dark)')
  const handler = (e: MediaQueryListEvent | MediaQueryList) => {
    callback(e.matches ? 'dark' : 'light')
  }
  mql.addEventListener('change', handler)
  return () => mql.removeEventListener('change', handler)
}
