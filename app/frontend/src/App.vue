<script lang="ts" setup>
import { ref, reactive, onMounted, onBeforeUnmount, computed, watch, nextTick } from 'vue'
import {
  GetState, GetLogs, Start, Stop, Restart, OpenBrowser,
  ResetAdmin, SetAutoStart, SetAutoStartService, GetAbout, OpenDataDir,
} from '../wailsjs/go/main/App'
import { EventsOn } from '../wailsjs/runtime/runtime'
import { initTheme, setThemeMode, getThemeMode, type ThemeMode } from './theme'

interface LogLine { time: string; level: string; msg: string }
interface State {
  status: string; isPortable: boolean; autoStart: boolean; autoStartService: boolean
  address: string; httpPort: number; httpsPort: number
  url: string; account: string
  dataDir: string; exePath: string
}
interface About {
  product: string; version: string; kernel: string; upstream: string
  license: string; thirdParty: string; disclaimer: string
}

const state = reactive<State>({
  status: 'stopped', isPortable: false, autoStart: false, autoStartService: true,
  address: '0.0.0.0', httpPort: 5244, httpsPort: -1,
  url: '', account: 'admin',
  dataDir: '', exePath: '',
})
const logs = ref<LogLine[]>([])
const MAX_LOGS = 2000 // 日志上限：超过则自动丢弃最旧记录，避免内存与渲染膨胀
// 日志事件先入缓冲，由 flushLogs 每秒批量刷入，避免高频 push 触发卡死
let logBuffer: LogLine[] = []
let logTimer: ReturnType<typeof setInterval> | null = null
function flushLogs() {
  if (logBuffer.length === 0) return
  const next = logs.value.concat(logBuffer)
  logBuffer = []
  if (next.length > MAX_LOGS) next.splice(0, next.length - MAX_LOGS)
  logs.value = next
}
const ready = ref(false)
const confirmReset = ref(false)
const msg = ref('')
const about = ref<About | null>(null)
const showAbout = ref(false)

// 主题（深/浅/自动）
const themeMode = ref<ThemeMode>(getThemeMode())
const themeModes: ThemeMode[] = ['light', 'dark', 'auto']
function applyTheme(m: ThemeMode) { themeMode.value = m; setThemeMode(m) }
const themeLabel = computed(() => themeMode.value === 'light' ? '浅色' : themeMode.value === 'dark' ? '深色' : '自动')

// 日志过滤（5.3）
const levelFilter = ref('all')
const searchKey = ref('')
const autoScroll = ref(true)
const logboxEl = ref<HTMLElement | null>(null)
const filteredLogs = computed(() => {
  let arr = logs.value
  if (levelFilter.value !== 'all') arr = arr.filter(l => l.level === levelFilter.value)
  const k = searchKey.value.trim().toLowerCase()
  if (k) arr = arr.filter(l => l.msg.toLowerCase().includes(k))
  return arr
})
watch(filteredLogs, () => {
  if (!autoScroll.value || !logboxEl.value) return
  nextTick(() => { logboxEl.value!.scrollTop = logboxEl.value!.scrollHeight })
})

const statusText = computed(() => {
  switch (state.status) {
    case 'running': return '运行中'
    case 'starting': return '启动中…'
    case 'stopping': return '停止中…'
    case 'error': return '错误'
    default: return '已停止'
  }
})
const statusClass = computed(() => 'badge ' + state.status)
const canStart = computed(() => state.status === 'stopped' || state.status === 'error')
const canStop = computed(() => state.status === 'running' || state.status === 'starting')

function flash(m: string) { msg.value = m; setTimeout(() => { if (msg.value === m) msg.value = '' }, 2500) }

async function copyText(t: string, label: string) {
  try { await navigator.clipboard.writeText(t); flash(label + ' 已复制') }
  catch { flash('复制失败') }
}

async function refresh() {
  try {
    const s = await GetState()
    Object.assign(state, s)
    ready.value = true
  } catch (e) { flash('获取状态失败: ' + e) }
}

async function doStart() { try { await Start(); } catch (e) { flash('启动失败: ' + e) } }
async function doStop() { try { await Stop(); } catch (e) { flash('停止失败: ' + e) } }
async function doRestart() { try { await Restart(); } catch (e) { flash('重启失败: ' + e) } }
async function doOpen() { try { await OpenBrowser(); } catch (e) { flash('' + e) } }
async function doResetAdmin() {
  if (!confirmReset.value) {
    confirmReset.value = true
    setTimeout(() => { confirmReset.value = false }, 4000)
    return
  }
  confirmReset.value = false
  try {
    await ResetAdmin()
    flash('账号密码已重置为 admin / admin')
    await refresh()
  } catch (e) { flash('重置失败: ' + e) }
}
async function toggleAutoStart() {
  try { await SetAutoStart(!state.autoStart); await refresh() } catch (e) { flash('' + e) }
}
async function toggleAutoStartService() {
  try { await SetAutoStartService(!state.autoStartService); await refresh() } catch (e) { flash('' + e) }
}
function doClearLogs() { logs.value = []; logBuffer = [] }
function doExportLogs() {
  const text = logs.value.map(l => `${l.time} [${l.level.toUpperCase()}] ${l.msg}`).join('\n')
  const blob = new Blob([text], { type: 'text/plain;charset=utf-8' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = `alist_win-${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')}.log`
  a.click()
  URL.revokeObjectURL(a.href)
}
async function doOpenDataDir() { try { await OpenDataDir() } catch (e) { flash('' + e) } }

onMounted(async () => {
  initTheme()
  await refresh()
  try {
    const init = await GetLogs()
    logs.value = init.length > MAX_LOGS ? init.slice(init.length - MAX_LOGS) : init
  } catch {}
  try { about.value = await GetAbout() as unknown as About } catch {}
  EventsOn('alist:state', (s: State) => { Object.assign(state, s) })
  // 日志高频事件先入缓冲，每秒批量刷新一次（flushLogs），避免逐条 push 卡死
  EventsOn('alist:log', (l: LogLine) => { logBuffer.push(l) })
  EventsOn('app:about', () => { showAbout.value = true })
  logTimer = setInterval(flushLogs, 1000)
})

onBeforeUnmount(() => { if (logTimer) clearInterval(logTimer) })
</script>

<template>
  <div class="app">
    <header class="top">
      <div class="brand">
        <span class="logo">A</span>
        <div>
          <h1>AList 桌面版</h1>
          <small class="ver" v-if="about">v{{ about.version }} · 内核 {{ about.kernel }}</small>
        </div>
        <small v-if="state.isPortable" class="tag port">便携模式</small>
        <small v-else class="tag inst">安装模式</small>
      </div>
      <div class="topRight">
        <span :class="statusClass">{{ statusText }}</span>
        <div class="themeSwitch" role="group" aria-label="主题">
          <button v-for="m in themeModes" :key="m"
                  :class="['themeBtn', { active: themeMode === m }]"
                  @click="applyTheme(m)"
                  :title="'主题：' + (m === 'light' ? '浅色' : m === 'dark' ? '深色' : '自动跟随系统')">
            {{ m === 'light' ? '☀' : m === 'dark' ? '☾' : '◐' }}
          </button>
        </div>
      </div>
    </header>

    <section class="card">
      <div class="row">
        <button :disabled="!canStart" @click="doStart">启动</button>
        <button :disabled="!canStop" @click="doStop" class="warn">停止</button>
        <button :disabled="state.status!=='running'" @click="doRestart">重启</button>
        <button :disabled="state.status!=='running'" @click="doOpen" class="primary">浏览器打开</button>
        <span class="spacer"></span>
        <button class="link" @click="doOpenDataDir">数据目录</button>
        <button :class="['link', { danger: confirmReset }]" @click="doResetAdmin" title="重置账号密码为 admin / admin（服务运行中会短暂重启）">
          {{ confirmReset ? '确认重置为 admin？' : '重置密码' }}
        </button>
        <button class="link" @click="showAbout = true">关于</button>
      </div>
      <div class="meta">
        <span>地址：<code class="clickable" @click="copyText(state.url, '地址')">{{ state.url }}</code></span>
        <span>数据目录：<code class="path">{{ state.dataDir }}</code></span>
      </div>
    </section>

    <section class="card">
      <h2>设置</h2>
      <div class="kv">
        <span>自动启动服务</span>
        <label class="switch">
          <input type="checkbox" :checked="state.autoStartService" @change="toggleAutoStartService" />
          <span class="slider"></span>
        </label>
        <span class="muted">打开本软件后约 1 秒自动启动 alist 服务</span>
      </div>
      <div class="kv">
        <span>开机自启</span>
        <label class="switch">
          <input type="checkbox" :checked="state.autoStart" @change="toggleAutoStart" />
          <span class="slider"></span>
        </label>
        <span class="muted">开机静默启动到托盘；关闭窗口即最小化到托盘，托盘"退出"才真正停止</span>
      </div>
    </section>

    <section class="card log">
      <div class="logbar">
        <h2>日志</h2>
        <select v-model="levelFilter">
          <option value="all">全部级别</option>
          <option value="info">INFO</option>
          <option value="warn">WARN</option>
          <option value="error">ERROR</option>
        </select>
        <input v-model="searchKey" class="search" placeholder="搜索…" />
        <label class="chk"><input type="checkbox" v-model="autoScroll" /> 自动滚动</label>
        <span class="spacer"></span>
        <button class="link" @click="doExportLogs">导出</button>
        <button class="link" @click="doClearLogs">清空</button>
      </div>
      <div class="logbox" ref="logboxEl">
        <div v-for="(l, i) in filteredLogs" :key="i" :class="'ln ' + l.level">
          <span class="t">{{ l.time }}</span>
          <span class="lv">{{ l.level }}</span>
          <span class="m">{{ l.msg }}</span>
        </div>
        <div v-if="filteredLogs.length === 0" class="muted">暂无日志</div>
      </div>
    </section>

    <div v-if="showAbout" class="modal-mask" @click.self="showAbout = false">
      <div class="modal">
        <h2>关于</h2>
        <template v-if="about">
          <div class="kv"><span>产品</span><code>{{ about.product }} v{{ about.version }}</code></div>
          <div class="kv"><span>内核</span><code>alist {{ about.kernel }}</code></div>
          <div class="kv"><span>许可证</span><code>{{ about.license }}</code></div>
          <div class="kv"><span>上游</span><code>{{ about.upstream }}</code></div>
          <p class="fine">{{ about.thirdParty }}</p>
          <p class="fine">{{ about.disclaimer }}</p>
        </template>
        <div class="row end"><button @click="showAbout = false">关闭</button></div>
      </div>
    </div>

    <div v-if="msg" class="toast">{{ msg }}</div>
  </div>
</template>

<style>
/* ============================================================
   双主题系统：浅色（冷调纸感） / 深色（深板岩）
   设计原则：所有可见文字均由变量驱动，两套主题各自全量定义，
   保证换主题时背景与文字颜色同步切换、层级分明：
   --text-1 主文字 / --text-2 次要文字 / --text-3 辅助提示
   ============================================================ */

/* ---------- 浅色主题（同时作为无类名时的兜底默认） ---------- */
:root,
:root.theme-light {
  --bg:#eef1f6;
  --bg-image:linear-gradient(180deg,#f8fafd 0%,#eef1f6 100%);
  --card:#ffffff; --card-2:#f6f8fb;
  --line:#e3e8ef; --line-strong:#c9d2de;

  --text-1:#1a2532; --text-2:#4d5a6b; --text-3:#8492a3;

  --primary:#2e6be6; --primary-strong:#1f56c4; --on-primary:#ffffff;
  --ok:#15864a; --warn:#b06c07; --danger:#d23b41;

  --code-bg:#eef2f7; --code-text:#33415a;
  --btn-bg:#ffffff; --btn-text:#2a3646; --btn-line:#c9d2de; --btn-hover:#9fb0c4;
  --input-bg:#fbfcfe; --input-text:#1a2532; --placeholder:#97a3b2;

  --tag-port-bg:rgba(46,107,230,.10); --tag-port-tx:#1f56c4;
  --tag-inst-bg:#e7ebf1; --tag-inst-tx:#5d6b7d;

  --badge-bg:#e9edf3; --badge-tx:#5d6b7d;
  --ok-bg:rgba(21,134,74,.12); --warn-bg:rgba(176,108,7,.12); --danger-bg:rgba(210,59,65,.12);

  --log-bg:#f9fafc; --log-text:#3a4557; --log-time:#96a1b1;
  --lv-info:#15864a; --lv-warn:#b06c07; --lv-error:#d23b41; --lv-debug:#2b62c4;

  --modal-bg:#ffffff; --modal-line:#e3e8ef; --modal-mask:rgba(15,23,32,.45);
  --toast-bg:#1f2b3a; --toast-tx:#ffffff;

  --shadow:0 1px 2px rgba(16,24,40,.06), 0 1px 3px rgba(16,24,40,.10);
  --shadow-lg:0 12px 32px rgba(16,24,40,.18);
  --scroll-thumb:#c3cdda;
}

/* ---------- 深色主题 ---------- */
:root.theme-dark {
  --bg:#0f141a;
  --bg-image:linear-gradient(180deg,#131a22 0%,#0f141a 100%);
  --card:#1a212b; --card-2:#1f2833;
  --line:#29323f; --line-strong:#3d4a5c;

  --text-1:#e9edf3; --text-2:#a7b1bf; --text-3:#7d8996;

  --primary:#6191ff; --primary-strong:#7ba3ff; --on-primary:#ffffff;
  --ok:#4ade80; --warn:#fbbf24; --danger:#f87171;

  --code-bg:#232c38; --code-text:#c6d1e0;
  --btn-bg:#222b37; --btn-text:#e9edf3; --btn-line:#3a4553; --btn-hover:#52616f;
  --input-bg:#1f2833; --input-text:#e9edf3; --placeholder:#6d7a89;

  --tag-port-bg:rgba(97,145,255,.15); --tag-port-tx:#8db0ff;
  --tag-inst-bg:#232c38; --tag-inst-tx:#8b97a6;

  --badge-bg:#242e3b; --badge-tx:#8b97a6;
  --ok-bg:rgba(74,222,128,.15); --warn-bg:rgba(251,191,36,.15); --danger-bg:rgba(248,113,113,.15);

  --log-bg:#0a0e13; --log-text:#cdd5df; --log-time:#6d7885;
  --lv-info:#52d183; --lv-warn:#e8b34b; --lv-error:#ff8a90; --lv-debug:#82b7ff;

  --modal-bg:#1c232d; --modal-line:#29323f; --modal-mask:rgba(0,0,0,.60);
  --toast-bg:#e9edf3; --toast-tx:#12181f;

  --shadow:0 1px 3px rgba(0,0,0,.4);
  --shadow-lg:0 12px 32px rgba(0,0,0,.55);
  --scroll-thumb:#3d4a5c;
}

* { box-sizing: border-box; }
html, body { height: 100%; }
body {
  margin:0;
  font-family: system-ui, "Segoe UI", "Microsoft YaHei", sans-serif;
  background: var(--bg);
  background-image: var(--bg-image);
  color: var(--text-1);
  transition: background .18s ease, color .18s ease;
}

/* ---------- 布局 ---------- */
.app { max-width: 820px; margin: 0 auto; padding: 16px; display:flex; flex-direction:column; gap:12px; }
.top { display:flex; align-items:center; justify-content:space-between; }
.brand { display:flex; align-items:center; gap:10px; }
.brand > div { display:flex; flex-direction:column; }
.topRight { display:flex; align-items:center; gap:10px; }

/* ---------- 品牌区 ---------- */
.logo {
  width:38px; height:38px; border-radius:9px;
  background:linear-gradient(135deg, var(--primary), var(--primary-strong));
  color:var(--on-primary); font-weight:800; font-size:20px;
  display:flex; align-items:center; justify-content:center;
  box-shadow: var(--shadow);
}
h1 { font-size:18px; margin:0; color:var(--text-1); font-weight:700; }
.ver { font-size:11px; color:var(--text-3); }
h2 { font-size:14px; margin:0 8px 0 0; color:var(--text-2); font-weight:600; }
.tag { font-size:11px; padding:1px 7px; border-radius:10px; }
.tag.port { background:var(--tag-port-bg); color:var(--tag-port-tx); }
.tag.inst { background:var(--tag-inst-bg); color:var(--tag-inst-tx); }

/* 主题切换器 */
.themeSwitch { display:flex; gap:2px; background:var(--card-2); border:1px solid var(--line); border-radius:9px; padding:2px; }
.themeBtn { border:none; background:transparent; color:var(--text-3); padding:3px 9px; border-radius:7px; font-size:13px; cursor:pointer; line-height:1.2; }
.themeBtn:hover { color:var(--text-1); }
.themeBtn.active { background:var(--card); color:var(--text-1); box-shadow:var(--shadow); }

/* ---------- 卡片 ---------- */
.card {
  background:var(--card); border:1px solid var(--line); border-radius:12px; padding:14px;
  box-shadow: var(--shadow);
  transition: background .18s ease, border-color .18s ease;
}
.row { display:flex; gap:8px; flex-wrap:wrap; align-items:center; }
.row.sub { margin-top:8px; }
.row.end { justify-content:flex-end; }
.spacer { flex:1; }

/* ---------- 按钮 ---------- */
button {
  border:1px solid var(--btn-line); background:var(--btn-bg); color:var(--btn-text);
  padding:7px 14px; border-radius:8px; cursor:pointer; font-size:13px;
  transition: background .15s ease, border-color .15s ease, color .15s ease;
}
button:hover:not(:disabled) { border-color:var(--btn-hover); color:var(--text-1); }
button:disabled { opacity:.45; cursor:not-allowed; }
button.primary { background:var(--primary); border-color:var(--primary); color:var(--on-primary); }
button.primary:hover:not(:disabled) { background:var(--primary-strong); border-color:var(--primary-strong); color:var(--on-primary); }
button.warn { color:var(--danger); border-color:var(--danger); }
button.link { border:none; background:none; color:var(--primary); padding:2px 6px; }
button.link:hover:not(:disabled) { color:var(--primary-strong); }
button.link.danger { color:var(--danger); }
button.link.danger:hover:not(:disabled) { color:var(--danger); opacity:.8; }

/* ---------- 元信息 / 键值 ---------- */
.meta { display:flex; flex-direction:column; gap:4px; margin-top:10px; font-size:12px; color:var(--text-3); }
.meta .path { word-break:break-all; }
code { background:var(--code-bg); color:var(--code-text); padding:1px 6px; border-radius:5px; font-size:12px; }
code.clickable { cursor:pointer; }
code.clickable:hover { background:var(--btn-hover); }
.kv { display:flex; align-items:center; gap:10px; padding:5px 0; font-size:13px; flex-wrap:wrap; }
.kv > span:first-child { width:64px; color:var(--text-2); flex:none; font-weight:500; }
.muted { color:var(--text-3); font-size:12px; }
.fine { font-size:12px; color:var(--text-2); margin:6px 0; line-height:1.6; }

/* ---------- 状态徽章 ---------- */
.badge { font-size:12px; padding:3px 10px; border-radius:12px; background:var(--badge-bg); color:var(--badge-tx); }
.badge.running { background:var(--ok-bg); color:var(--ok); }
.badge.starting, .badge.stopping { background:var(--warn-bg); color:var(--warn); }
.badge.error { background:var(--danger-bg); color:var(--danger); }

/* ---------- 日志 ---------- */
.logbar { display:flex; align-items:center; gap:8px; margin-bottom:8px; flex-wrap:wrap; }
.logbar select, .logbar input.search {
  border:1px solid var(--btn-line); border-radius:7px; padding:4px 8px; font-size:12px;
  background:var(--input-bg); color:var(--input-text);
}
.logbar input.search { width:140px; }
.chk { font-size:12px; color:var(--text-2); display:flex; gap:4px; align-items:center; }
.logbox {
  height:220px; overflow:auto; border:1px solid var(--line);
  background:var(--log-bg); color:var(--log-text);
  border-radius:8px; padding:8px;
  font-family:ui-monospace,Menlo,Consolas,monospace; font-size:12px;
}
.ln { display:flex; gap:8px; line-height:1.5; }
.ln .t { color:var(--log-time); flex:none; }
.ln .lv { flex:none; width:42px; font-weight:600; }
.ln .m { color:var(--log-text); }
.ln.info .lv { color:var(--lv-info); }
.ln.warn .lv { color:var(--lv-warn); }
.ln.error .lv { color:var(--lv-error); }
.ln.debug .lv { color:var(--lv-debug); }

/* ---------- 开关 ---------- */
.switch { position:relative; display:inline-block; width:40px; height:22px; }
.switch input { opacity:0; width:0; height:0; }
.slider { position:absolute; cursor:pointer; inset:0; background:var(--btn-hover); border-radius:22px; transition:.2s; }
.slider:before { content:""; position:absolute; height:16px; width:16px; left:3px; top:3px; background:#ffffff; border-radius:50%; transition:.2s; }
.switch input:checked + .slider { background:var(--primary); }
.switch input:checked + .slider:before { transform:translateX(18px); }

/* ---------- 输入 ---------- */
input[type=text] {
  flex:1; min-width:180px; border:1px solid var(--btn-line);
  background:var(--input-bg); color:var(--input-text);
  border-radius:8px; padding:7px 10px; font-size:13px;
}
input[type=text]::placeholder { color:var(--placeholder); }

/* ---------- 弹窗 / Toast ---------- */
.modal-mask { position:fixed; inset:0; background:var(--modal-mask); display:flex; align-items:center; justify-content:center; z-index:40; }
.modal { background:var(--modal-bg); border:1px solid var(--modal-line); border-radius:14px; padding:20px; width:440px; max-width:92vw; box-shadow:var(--shadow-lg); }
.modal .kv > span:first-child { width:56px; }
.toast { position:fixed; bottom:18px; left:50%; transform:translateX(-50%); background:var(--toast-bg); color:var(--toast-tx); padding:8px 16px; border-radius:8px; font-size:13px; z-index:50; box-shadow:var(--shadow-lg); }

/* ---------- 滚动条 ---------- */
::-webkit-scrollbar { width:9px; height:9px; }
::-webkit-scrollbar-track { background:transparent; }
::-webkit-scrollbar-thumb { background:var(--scroll-thumb); border-radius:5px; }
</style>
