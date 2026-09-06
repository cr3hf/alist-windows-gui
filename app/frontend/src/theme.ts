// 主题管理：深色 / 浅色 / 自动（跟随系统），持久化到 localStorage。
// 实现方式：在 <html> 根元素切换 .theme-dark / .theme-light 类，
// CSS 变量在 App.vue 中按类定义，组件只引用变量。

export type ThemeMode = 'light' | 'dark' | 'auto'

const KEY = 'alistwin-theme'

export function loadThemeMode(): ThemeMode {
  const v = localStorage.getItem(KEY)
  if (v === 'light' || v === 'dark' || v === 'auto') return v
  return 'auto'
}

export function saveThemeMode(m: ThemeMode) {
  localStorage.setItem(KEY, m)
}

// resolveMode 把 auto 解析成具体的 light/dark。
export function resolveMode(m: ThemeMode): 'light' | 'dark' {
  if (m !== 'auto') return m
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

let currentMode: ThemeMode = loadThemeMode()
let mediaSub: ((e: MediaQueryListEvent) => void) | null = null

function apply() {
  const effective = resolveMode(currentMode)
  const root = document.documentElement
  root.classList.remove('theme-light', 'theme-dark')
  root.classList.add('theme-' + effective)
}

// initTheme 挂载时调用：立即应用 + 订阅系统主题变化（auto 模式下实时跟随）。
export function initTheme() {
  apply()
  const mq = window.matchMedia('(prefers-color-scheme: dark)')
  if (mediaSub) mq.removeEventListener('change', mediaSub)
  mediaSub = () => { if (currentMode === 'auto') apply() }
  mq.addEventListener('change', mediaSub)
}

// setThemeMode 切换并持久化。
export function setThemeMode(m: ThemeMode) {
  currentMode = m
  saveThemeMode(m)
  apply()
}

export function getThemeMode(): ThemeMode { return currentMode }
