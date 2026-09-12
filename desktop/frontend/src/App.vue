<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { Connect, Disconnect, GetConfig, SaveConfig, GetStatus, GetLogs } from '../wailsjs/go/main/App'

const currentTab = ref('dashboard')
const settingsTab = ref('transport')
const isConnected = ref(false)
const isConnecting = ref(false)
const errorMessage = ref('')
const saveSuccessMessage = ref('')

const config = ref({
  transport: 'yandex',
  doc_urls: '',
  max_token: '',
  max_uid: '',
  secret_key: '',
  socks_port: 1080,
  mode: 'sysproxy',
  theme: 'dark',
  bypass: '<local>;localhost;127.*;192.168.*;10.*',
  auto_start: false,
  start_minimized: false,
  close_to_tray: false,
  debug: false
})

const status = ref({
  connected: false,
  mode: 'sysproxy',
  transport: 'yandex',
  uptime: '00:00:00',
  upload_speed: '0 B/s',
  download_speed: '0 B/s',
  total_upload: '0 B',
  total_download: '0 B',
  ping_ms: -1,
  recent_logs: '',
  current_doc_url: ''
})

const themeSetting = ref('system')
const resolvedTheme = ref('dark')

function getSystemTheme() {
  if (typeof window !== 'undefined' && window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches) {
    return 'light'
  }
  return 'dark'
}

function updateAppliedTheme() {
  const actual = themeSetting.value === 'system' ? getSystemTheme() : themeSetting.value
  resolvedTheme.value = actual
  document.documentElement.setAttribute('data-theme', actual)
}

function applyThemeSetting(setting) {
  themeSetting.value = setting || 'system'
  localStorage.setItem('openflux-theme', themeSetting.value)
  if (config.value) {
    config.value.theme = themeSetting.value
  }
  updateAppliedTheme()
}

function onThemeChange() {
  applyThemeSetting(themeSetting.value)
  saveSettings()
}

if (typeof window !== 'undefined' && window.matchMedia) {
  window.matchMedia('(prefers-color-scheme: light)').addEventListener('change', () => {
    if (themeSetting.value === 'system') {
      updateAppliedTheme()
    }
  })
}

const showKey = ref(false)
const logContainer = ref(null)
const autoScrollLogs = ref(true)

let statusTimer = null

onMounted(async () => {
  try {
    const loadedCfg = await GetConfig()
    if (loadedCfg) {
      config.value = Object.assign({
        transport: 'yandex',
        max_token: '',
        max_uid: '',
        theme: 'system'
      }, loadedCfg)
    }
  } catch (err) {
    console.error('Failed to load config:', err)
  }

  const savedTheme = (config.value && config.value.theme) || localStorage.getItem('openflux-theme') || 'system'
  applyThemeSetting(savedTheme)

  await refreshStatus()
  statusTimer = setInterval(refreshStatus, 1000)
})

onUnmounted(() => {
  if (statusTimer) clearInterval(statusTimer)
})

async function refreshStatus() {
  try {
    const res = await GetStatus()
    if (res) {
      status.value = res
      isConnected.value = res.connected
      if (res.connected) {
        isConnecting.value = false
      }
    }
  } catch (e) {
    console.error('Error fetching status:', e)
  }
}

async function toggleConnection() {
  errorMessage.value = ''
  if (isConnected.value) {
    try {
      await Disconnect()
      isConnected.value = false
      isConnecting.value = false
      await refreshStatus()
    } catch (err) {
      errorMessage.value = String(err)
    }
  } else {
    isConnecting.value = true
    try {
      // First save current config so mode and doc_urls are applied
      await SaveConfig(config.value)
      await Connect()
      isConnected.value = true
      isConnecting.value = false
      await refreshStatus()
    } catch (err) {
      isConnecting.value = false
      errorMessage.value = String(err)
    }
  }
}

function selectMode(m) {
  if (isConnected.value) return
  config.value.mode = m
  saveSettings()
}

async function saveSettings() {
  saveSuccessMessage.value = ''
  errorMessage.value = ''
  try {
    await SaveConfig(config.value)
    saveSuccessMessage.value = 'Настройки успешно сохранены!'
    setTimeout(() => {
      saveSuccessMessage.value = ''
    }, 2500)
  } catch (err) {
    errorMessage.value = 'Ошибка сохранения: ' + String(err)
  }
}

function copyLogs() {
  if (status.value.recent_logs) {
    navigator.clipboard.writeText(status.value.recent_logs)
  }
}

function getPingClass(ping) {
  if (ping <= 0) return ''
  if (ping <= 250) return 'ping-good'
  if (ping <= 500) return 'ping-medium'
  return 'ping-bad'
}

function formatDocUrl(url) {
  if (!url) return ''
  try {
    const u = new URL(url)
    const id = u.searchParams.get('url') || u.searchParams.get('id') || ''
    if (id) {
      const shortId = id.length > 24 ? id.substring(0, 12) + '...' + id.substring(id.length - 8) : id
      return `${u.hostname} (${shortId})`
    }
    const path = u.pathname.replace(/^\/+/, '')
    return u.hostname + (path ? '/' + path : '')
  } catch (_) {
    return url.length > 35 ? url.substring(0, 32) + '...' : url
  }
}
</script>

<template>
  <div class="window-container">
    <!-- Sidebar Navigation -->
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-left">
          <div class="brand-icon">
            <img src="./assets/images/logo.png" class="brand-logo-img" alt="OpenFlux" />
          </div>
          <div class="brand-text">
            <span class="title">OpenFlux</span>
            <span class="subtitle">v1.0.1</span>
          </div>
        </div>
      </div>

      <nav class="nav-menu">
        <button 
          :class="['nav-item', { active: currentTab === 'dashboard' }]" 
          @click="currentTab = 'dashboard'"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
          </svg>
          <span>Туннель</span>
        </button>

        <button 
          :class="['nav-item', { active: currentTab === 'settings' }]" 
          @click="currentTab = 'settings'"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="3" />
            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
          </svg>
          <span>Настройки</span>
        </button>

        <button 
          :class="['nav-item', { active: currentTab === 'logs' }]" 
          @click="currentTab = 'logs'"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="4 17 10 11 4 5" />
            <line x1="12" y1="19" x2="20" y2="19" />
          </svg>
          <span>Журнал событий</span>
        </button>
      </nav>

      <!-- Bottom Status Pill -->
      <div class="sidebar-footer">
        <div :class="['status-badge', isConnected ? 'online' : (isConnecting ? 'pending' : 'offline')]">
          <span class="status-dot"></span>
          <span class="status-text">
            {{ isConnected ? 'Подключено' : (isConnecting ? 'Подключение...' : 'Отключено') }}
          </span>
        </div>
      </div>
    </aside>

    <!-- Main Content Area -->
    <main class="content-area">
      <!-- Error Banner -->
      <div v-if="errorMessage" class="alert-banner danger">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10" />
          <line x1="12" y1="8" x2="12" y2="12" />
          <line x1="12" y1="16" x2="12.01" y2="16" />
        </svg>
        <span>{{ errorMessage }}</span>
        <button class="close-btn" @click="errorMessage = ''">&times;</button>
      </div>

      <!-- TAB 1: DASHBOARD -->
      <section v-if="currentTab === 'dashboard'" class="tab-panel dashboard-view">
        <!-- Big Power Button Area -->
        <div class="power-section">
          <button 
            :class="['power-btn', isConnected ? 'active' : '', isConnecting ? 'loading' : '']"
            :disabled="isConnecting"
            @click="toggleConnection"
          >
            <div class="pulse-ring"></div>
            <svg class="power-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
              <path d="M18.36 6.64a9 9 0 1 1-12.73 0" />
              <line x1="12" y1="2" x2="12" y2="12" />
            </svg>
          </button>
          
          <div class="connection-label">
            <h2>{{ isConnected ? 'Подключено' : (isConnecting ? 'Подключение...' : 'Готов к подключению') }}</h2>
            <div class="hero-badges-row">
              <div class="status-badge transport-badge">
                <span class="pulse-dot-sm"></span>
                <span>
                  {{ config.transport === 'vyandex' ? 'Yandex Volga' : (config.transport === 'oneme' ? 'MAX Messenger' : 'Yandex Docs') }}
                </span>
              </div>
              <div v-if="isConnected && status.current_doc_url" class="status-badge doc-badge" :title="status.current_doc_url">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="doc-icon">
                  <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
                  <polyline points="14 2 14 8 20 8" />
                </svg>
                <span class="doc-name">{{ formatDocUrl(status.current_doc_url) }}</span>
              </div>
            </div>
            <p v-if="isConnected" class="uptime-text">Время в сети: {{ status.uptime }}</p>
            <p v-else class="hint-text">Нажмите кнопку для активации туннеля</p>
          </div>
        </div>

        <!-- Mode Selector Cards -->
        <div class="modes-grid">
          <div 
            :class="['mode-card', { selected: config.mode === 'wintun', disabled: isConnected }]"
            @click="selectMode('wintun')"
          >
            <div class="mode-header">
              <span class="mode-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
                </svg>
              </span>
              <span class="mode-title">Полный VPN (Wintun)</span>
            </div>
            <p class="mode-desc">Весь трафик ПК идет через тоннель.</p>
          </div>

          <div 
            :class="['mode-card', { selected: config.mode === 'sysproxy', disabled: isConnected }]"
            @click="selectMode('sysproxy')"
          >
            <div class="mode-header">
              <span class="mode-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <circle cx="12" cy="12" r="10" />
                  <line x1="2" y1="12" x2="22" y2="12" />
                  <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
                </svg>
              </span>
              <span class="mode-title">Системный прокси</span>
            </div>
            <p class="mode-desc">Автоматическая настройка прокси Windows для всех браузеров. Без установки сетевых драйверов.</p>
          </div>

          <div 
            :class="['mode-card', { selected: config.mode === 'socks', disabled: isConnected }]"
            @click="selectMode('socks')"
          >
            <div class="mode-header">
              <span class="mode-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <rect x="2" y="2" width="20" height="8" rx="2" ry="2" />
                  <rect x="2" y="14" width="20" height="8" rx="2" ry="2" />
                  <line x1="6" y1="6" x2="6.01" y2="6" />
                  <line x1="6" y1="18" x2="6.01" y2="18" />
                </svg>
              </span>
              <span class="mode-title">Только SOCKS5</span>
            </div>
            <p class="mode-desc">Сервер на 127.0.0.1:{{ config.socks_port }} для ручной настройки (Telegram, Proxifier).</p>
          </div>
        </div>

        <!-- Metrics Dashboard -->
        <div class="metrics-grid">
          <div class="metric-card">
            <div class="metric-icon down">↓</div>
            <div class="metric-content">
              <span class="metric-val">{{ status.download_speed }}</span>
              <span class="metric-name">Загрузка (всего: {{ status.total_download }})</span>
            </div>
          </div>

          <div class="metric-card">
            <div class="metric-icon up">↑</div>
            <div class="metric-content">
              <span class="metric-val">{{ status.upload_speed }}</span>
              <span class="metric-name">Отдача (всего: {{ status.total_upload }})</span>
            </div>
          </div>

          <div class="metric-card">
            <div class="metric-icon ping">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M5 12.55a11 11 0 0 1 14.08 0" />
                <path d="M1.42 9a16 16 0 0 1 21.16 0" />
                <path d="M8.53 16.11a6 6 0 0 1 6.95 0" />
                <line x1="12" y1="20" x2="12.01" y2="20" stroke-width="3" />
              </svg>
            </div>
            <div class="metric-content">
              <span :class="['metric-val', getPingClass(status.ping_ms)]">{{ status.ping_ms > 0 ? status.ping_ms + ' мс' : (status.ping_ms === 0 ? '< 1 мс' : (isConnected ? 'Замер...' : '—')) }}</span>
              <span class="metric-name">Задержка (Ping)</span>
            </div>
          </div>
        </div>
      </section>

      <!-- TAB 2: SETTINGS -->
      <section v-if="currentTab === 'settings'" class="tab-panel settings-view">
        <!-- Settings Sub-tabs Navigation -->
        <div class="settings-subtabs">
          <button 
            :class="['subtab-btn', { active: settingsTab === 'transport' }]" 
            @click="settingsTab = 'transport'"
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10" />
              <line x1="2" y1="12" x2="22" y2="12" />
              <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
            </svg>
            <span>Сервер и Транспорт</span>
          </button>

          <button 
            :class="['subtab-btn', { active: settingsTab === 'network' }]" 
            @click="settingsTab = 'network'"
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="2" y="2" width="20" height="8" rx="2" ry="2" />
              <rect x="2" y="14" width="20" height="8" rx="2" ry="2" />
              <line x1="6" y1="6" x2="6.01" y2="6" />
              <line x1="6" y1="18" x2="6.01" y2="18" />
            </svg>
            <span>Сеть и Обход</span>
          </button>

          <button 
            :class="['subtab-btn', { active: settingsTab === 'system' }]" 
            @click="settingsTab = 'system'"
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="3" />
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
            </svg>
            <span>Система и Интерфейс</span>
          </button>
        </div>

        <!-- Subtab 1: Transport -->
        <div v-show="settingsTab === 'transport'">
          <!-- CARD 1: Server and Transport -->
          <div class="settings-card">
            <!-- Transport Selection Dropdown -->
          <div class="form-group">
            <label>Транспортный протокол</label>
            <div class="select-wrapper">
              <select v-model="config.transport" :disabled="isConnected" class="custom-select">
                <option value="yandex">Яндекс.Документы</option>
                <option value="vyandex">Яндекс.Волга</option>
                <option value="oneme">MAX Messenger</option>
              </select>
              <span class="select-arrow">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="6 9 12 15 18 9" />
                </svg>
              </span>
            </div>
          </div>

          <!-- Yandex Docs / Volga URLs -->
          <div v-if="config.transport !== 'oneme'" class="form-group">
            <label>
              Ссылки на Яндекс.Документы (Multi-URL пул)
              <span class="label-hint">Ссылки на рабочие документы через перенос строки или запятую</span>
            </label>
            <textarea 
              v-model="config.doc_urls" 
              placeholder="https://docs.yandex.ru/docs/view?url=...&#10;https://docs.yandex.ru/docs/view?url=..."
              rows="3"
              :disabled="isConnected"
            ></textarea>
          </div>

          <!-- MAX Messenger Credentials -->
          <div v-if="config.transport === 'oneme'" class="form-row">
            <div class="form-group flex-1">
              <label>
                MAX Web Token
                <span class="label-hint">Токен авторизации web.max</span>
              </label>
              <input 
                type="text" 
                v-model="config.max_token" 
                placeholder="Вставьте токен web.max..."
                :disabled="isConnected"
              />
            </div>

            <div class="form-group w-140">
              <label>
                MAX User ID
                <span class="label-hint">ID пользователя</span>
              </label>
              <input 
                type="text" 
                v-model="config.max_uid" 
                placeholder="79001234567"
                :disabled="isConnected"
              />
            </div>
          </div>

          <div class="form-group">
            <label>Секретный ключ (ChaCha20-Poly1305)</label>
            <div class="password-input">
              <input 
                :type="showKey ? 'text' : 'password'" 
                v-model="config.secret_key" 
                placeholder="Ключ E2E шифрования..."
                :disabled="isConnected"
              />
              <button class="toggle-eye" @click="showKey = !showKey" :title="showKey ? 'Скрыть ключ' : 'Показать ключ'">
                <svg v-if="showKey" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24" />
                  <line x1="1" y1="1" x2="23" y2="23" />
                </svg>
                <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8z" />
                  <circle cx="12" cy="12" r="3" />
                </svg>
              </button>
            </div>
          </div>
        </div>
      </div>

        <!-- Subtab 2: Network and Routing -->
        <div v-show="settingsTab === 'network'">
          <!-- CARD 2: Network and Routing -->
          <div class="settings-card">

            <div class="form-group">
              <label>
                SOCKS5 Порт
                <span class="label-hint">Локальный порт прокси для приложений и браузеров (по умолчанию 1080)</span>
              </label>
              <input 
                type="number" 
                v-model.number="config.socks_port" 
                placeholder="1080"
                :disabled="isConnected"
              />
            </div>

            <div class="form-group">
              <label>
                Исключения обхода (Bypass)
                <span class="label-hint">Сайты и подсети напрямую (без туннеля)</span>
              </label>
              <input 
                type="text" 
                v-model="config.bypass" 
                placeholder="&lt;local&gt;;localhost;127.*;192.168.*;10.*"
                :disabled="isConnected"
              />
            </div>
          </div>
        </div>

        <!-- Subtab 3: Interface and System -->
        <div v-show="settingsTab === 'system'">
          <!-- CARD 3: Interface and System -->
          <div class="settings-card">

            <div class="form-group">
              <label>Тема оформления</label>
              <div class="select-wrapper">
                <select v-model="themeSetting" @change="onThemeChange" class="custom-select">
                  <option value="light">Светлая</option>
                  <option value="dark">Тёмная</option>
                  <option value="system">Системная</option>
                </select>
                <span class="select-arrow">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="6 9 12 15 18 9" />
                  </svg>
                </span>
              </div>
            </div>

            <div class="toggles-list">
              <label class="toggle-item">
                <input type="checkbox" v-model="config.auto_start" />
                <span class="checkbox-box"></span>
                <div class="toggle-info">
                  <span class="toggle-title">Запускать вместе с Windows</span>
                  <span class="toggle-desc">Автоматический запуск приложения при входе в систему</span>
                </div>
              </label>

              <label class="toggle-item">
                <input type="checkbox" v-model="config.start_minimized" />
                <span class="checkbox-box"></span>
                <div class="toggle-info">
                  <span class="toggle-title">Запускать свёрнутым в трей</span>
                  <span class="toggle-desc">Не открывать главное окно при автозапуске</span>
                </div>
              </label>

              <label class="toggle-item">
                <input type="checkbox" v-model="config.close_to_tray" />
                <span class="checkbox-box"></span>
                <div class="toggle-info">
                  <span class="toggle-title">Сворачивать в трей при закрытии</span>
                  <span class="toggle-desc">При нажатии на крестик прятать окно в трей вместо выхода</span>
                </div>
              </label>

              <label class="toggle-item">
                <input type="checkbox" v-model="config.debug" />
                <span class="checkbox-box"></span>
                <div class="toggle-info">
                  <span class="toggle-title">Подробный журнал (Debug Logging)</span>
                  <span class="toggle-desc">Выводить детальные диагностические сообщения в журнал</span>
                </div>
              </label>
            </div>
          </div>
        </div>

        <!-- Settings Save Button -->
        <div class="settings-actions">
          <button class="btn primary" @click="saveSettings">Сохранить</button>
          <span v-if="saveSuccessMessage" class="save-success-badge">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" style="width: 16px; height: 16px;">
              <polyline points="20 6 9 17 4 12" />
            </svg>
            {{ saveSuccessMessage }}
          </span>
        </div>
      </section>

      <!-- TAB 3: LOGS -->
      <section v-if="currentTab === 'logs'" class="tab-panel logs-view">
        <div class="section-header row-header">
          <div>
            <h2>Журнал ядра (Live Logs)</h2>
            <p>Диагностика соединений, статус реконнектов и переключения документов</p>
          </div>
          <div class="logs-actions">
            <button class="btn secondary sm" @click="copyLogs">Скопировать журнал</button>
          </div>
        </div>

        <div class="terminal-box" ref="logContainer">
          <pre v-if="status.recent_logs">{{ status.recent_logs }}</pre>
          <div v-else class="empty-logs">Журнал пока пуст...</div>
        </div>
      </section>
    </main>

    <!-- Floating Toast Notification -->
    <Transition name="toast">
      <div v-if="saveSuccessMessage" class="toast-notification">
        <div class="toast-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <polyline points="20 6 9 17 4 12" />
          </svg>
        </div>
        <div class="toast-content">
          <span class="toast-title">Успешно</span>
          <span class="toast-msg">{{ saveSuccessMessage }}</span>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.window-container {
  display: flex;
  width: 100%;
  height: 100%;
}

/* Sidebar */
.sidebar {
  width: 250px;
  background: var(--bg-glass, rgba(11, 15, 25, 0.9));
  border-right: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  padding: 24px 16px;
  user-select: none;
  backdrop-filter: blur(16px);
}

.brand {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 4px 20px 4px;
  border-bottom: 1px solid var(--border-color);
}

.brand-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.theme-toggle-btn {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-sm);
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s ease;
  flex-shrink: 0;
}

.theme-toggle-btn:hover {
  background: rgba(255, 255, 255, 0.12);
  color: var(--text-primary);
  border-color: var(--border-active);
}

.theme-toggle-btn svg {
  width: 16px;
  height: 16px;
}

.brand-icon {
  width: 38px;
  height: 38px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  box-shadow: 0 4px 14px rgba(6, 182, 212, 0.35);
}

.brand-logo-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.brand-text .title {
  display: block;
  font-size: 17px;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: -0.5px;
}

.brand-text .subtitle {
  display: block;
  font-size: 11px;
  color: var(--text-muted);
}

.nav-menu {
  margin-top: 24px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  text-align: left;
}

.nav-item svg {
  width: 18px;
  height: 18px;
}

.nav-item:hover {
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-primary);
}

.nav-item.active {
  background: rgba(56, 189, 248, 0.12);
  color: var(--accent-cyan);
  border: 1px solid rgba(56, 189, 248, 0.2);
}

.sidebar-footer {
  padding-top: 16px;
  border-top: 1px solid var(--border-color);
}

.status-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.03);
  font-size: 12px;
  font-weight: 500;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--text-muted);
}

.status-badge.online .status-dot {
  background: var(--accent-emerald);
  box-shadow: 0 0 8px var(--accent-emerald);
}
.status-badge.online {
  color: var(--accent-emerald);
  background: rgba(16, 185, 129, 0.1);
}

.status-badge.pending .status-dot {
  background: var(--accent-amber);
  animation: pulse-ring 1.5s infinite;
}
.status-badge.pending {
  color: var(--accent-amber);
  background: rgba(245, 158, 11, 0.1);
}

.status-badge.offline {
  color: var(--text-muted);
}

/* Content Area */
.content-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 32px 36px;
  overflow-y: auto;
  position: relative;
}

.tab-panel {
  display: flex;
  flex-direction: column;
  flex: 1;
}

/* Alerts */
.alert-banner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-radius: var(--radius-sm);
  margin-bottom: 20px;
  font-size: 13px;
}
.alert-banner svg {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}
.alert-banner.danger {
  background: rgba(244, 63, 94, 0.15);
  border: 1px solid rgba(244, 63, 94, 0.3);
  color: #fecdd3;
}
.alert-banner.success {
  background: rgba(16, 185, 129, 0.15);
  border: 1px solid rgba(16, 185, 129, 0.3);
  color: #a7f3d0;
}
.close-btn {
  margin-left: auto;
  background: transparent;
  border: none;
  color: inherit;
  font-size: 18px;
  cursor: pointer;
}

/* Power Section */
.power-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 20px 0 32px 0;
}

.power-btn {
  position: relative;
  width: 100px;
  height: 100px;
  border-radius: 50%;
  border: 2px solid var(--border-control);
  background: linear-gradient(145deg, #1e293b, #0f172a);
  color: var(--text-muted);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.power-btn:hover {
  transform: scale(1.04);
  border-color: var(--border-control-hover);
  color: var(--text-primary);
}

.power-icon {
  width: 44px;
  height: 44px;
  transition: all 0.3s ease;
}

.power-btn.active {
  border-color: var(--accent-emerald);
  background: linear-gradient(145deg, #064e3b, #022c22);
  color: #34d399;
  box-shadow: 0 0 32px rgba(16, 185, 129, 0.4);
}

.power-btn.active .power-icon {
  filter: drop-shadow(0 0 8px rgba(52, 211, 153, 0.8));
}

.power-btn.loading {
  border-color: var(--accent-amber);
  color: var(--accent-amber);
}

/* Light theme for Power Button */
[data-theme="light"] .power-btn {
  border: 2px solid rgba(15, 23, 42, 0.18);
  background: linear-gradient(145deg, #ffffff, #e2e8f0);
  color: #64748b;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.08), inset 0 2px 4px rgba(255, 255, 255, 0.9);
}

[data-theme="light"] .power-btn:hover {
  border-color: var(--accent-cyan);
  color: var(--accent-cyan);
  box-shadow: 0 10px 28px rgba(2, 132, 199, 0.2), inset 0 2px 4px rgba(255, 255, 255, 0.9);
}

[data-theme="light"] .power-btn.active {
  border-color: #047857;
  background: linear-gradient(145deg, #10b981, #059669);
  color: #ffffff;
  box-shadow: 0 8px 28px rgba(16, 185, 129, 0.45);
}

[data-theme="light"] .power-btn.active .power-icon {
  filter: drop-shadow(0 2px 6px rgba(0, 0, 0, 0.25));
}

[data-theme="light"] .power-btn.loading {
  border-color: var(--accent-amber);
  color: var(--accent-amber);
  background: linear-gradient(145deg, #fffbeb, #fef3c7);
}

.connection-label {
  margin-top: 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.connection-label h2 {
  font-size: 16px;
  font-weight: 700;
  letter-spacing: 0.5px;
  color: var(--text-primary);
}

.hero-badges-row {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin: 6px 0 2px 0;
  max-width: 100%;
}

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 26px;
  padding: 0 12px;
  border-radius: 13px;
  font-size: 11px;
  font-weight: 600;
  box-sizing: border-box;
  line-height: 1;
  white-space: nowrap;
}

.status-badge.transport-badge {
  background: rgba(56, 189, 248, 0.12);
  border: 1px solid rgba(56, 189, 248, 0.35);
  color: var(--accent-cyan);
}

.status-badge.doc-badge {
  background: var(--bg-card);
  border: 1px solid var(--border-control);
  color: var(--text-secondary);
  max-width: 280px;
  box-shadow: var(--shadow-sm);
}

.status-badge.doc-badge .doc-icon {
  width: 13px;
  height: 13px;
  flex-shrink: 0;
  color: var(--accent-cyan);
}

.status-badge.doc-badge .doc-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pulse-dot-sm {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--accent-cyan);
  flex-shrink: 0;
}

.uptime-text {
  font-size: 13px;
  color: var(--accent-emerald);
  margin-top: 4px;
}

.hint-text {
  font-size: 13px;
  color: var(--text-muted);
  margin-top: 4px;
}

/* Modes Grid */
.modes-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.mode-card {
  background: var(--bg-card);
  border: 1.5px solid var(--border-control);
  border-radius: var(--radius-md);
  padding: 16px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.mode-card:hover:not(.disabled) {
  background: var(--bg-card-hover);
  border-color: var(--border-control-hover);
}

.mode-card.selected {
  border-color: var(--accent-cyan);
  background: rgba(6, 182, 212, 0.08);
}

.mode-card.disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.mode-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.mode-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-secondary);
  transition: all 0.2s ease;
}

.mode-icon svg {
  width: 16px;
  height: 16px;
}

.mode-card:hover:not(.disabled) .mode-icon {
  color: var(--text-primary);
  background: rgba(255, 255, 255, 0.1);
}

.mode-card.selected .mode-icon {
  background: rgba(6, 182, 212, 0.2);
  color: var(--accent-cyan);
}

.mode-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.mode-desc {
  font-size: 12px;
  color: var(--text-muted);
  line-height: 1.4;
}

/* Metrics Grid */
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}

.metric-card {
  background: var(--bg-card);
  border: 1.5px solid var(--border-control);
  border-radius: var(--radius-md);
  padding: 14px 18px;
  display: flex;
  align-items: center;
  gap: 14px;
}

.metric-icon {
  width: 38px;
  height: 38px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  font-weight: bold;
}

.metric-icon.down {
  background: rgba(56, 189, 248, 0.15);
  color: var(--accent-cyan);
}

.metric-icon.up {
  background: rgba(16, 185, 129, 0.15);
  color: var(--accent-emerald);
}

.metric-icon.ping {
  background: rgba(245, 158, 11, 0.15);
  color: var(--accent-amber);
}

.metric-icon.ping svg {
  width: 18px;
  height: 18px;
}

.metric-content {
  display: flex;
  flex-direction: column;
}

.metric-val {
  font-size: 16px;
  font-weight: 700;
  color: var(--text-primary);
  transition: color 0.2s ease;
}

.metric-val.ping-good {
  color: var(--accent-emerald);
}

.metric-val.ping-medium {
  color: var(--accent-amber);
}

.metric-val.ping-bad {
  color: var(--accent-rose);
}

.metric-name {
  font-size: 11px;
  color: var(--text-muted);
}

/* Settings View */
.section-header {
  margin-bottom: 24px;
}

.section-header h2 {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-primary);
}

.section-header p {
  font-size: 13px;
  color: var(--text-muted);
  margin-top: 4px;
}

.section-header.row-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

/* Settings Sub-tabs */
.settings-subtabs {
  display: flex;
  gap: 8px;
  margin-bottom: 20px;
  background: var(--bg-card);
  border: 1.5px solid var(--border-control);
  padding: 6px;
  border-radius: var(--radius-md);
}

.subtab-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: var(--radius-sm);
  background: transparent;
  border: none;
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.subtab-btn svg {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}

.subtab-btn:hover {
  color: var(--text-primary);
  background: var(--bg-card-hover);
}

.subtab-btn.active {
  background: rgba(56, 189, 248, 0.15);
  color: var(--accent-cyan);
  font-weight: 600;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

/* Settings Actions */
.settings-actions {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-top: 4px;
  margin-bottom: 24px;
}

.save-success-badge {
  color: var(--accent-emerald);
  font-size: 13px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 6px;
  background: rgba(16, 185, 129, 0.12);
  padding: 6px 12px;
  border-radius: var(--radius-sm);
  border: 1px solid rgba(16, 185, 129, 0.3);
}

/* Settings Cards */
.settings-card {
  background: var(--bg-card);
  border: 1.5px solid var(--border-control);
  border-radius: var(--radius-md);
  padding: 20px;
  margin-bottom: 20px;
  box-shadow: var(--shadow-sm);
}

.settings-card .toggles-list {
  margin: 16px 0 0 0;
  padding: 0;
  background: transparent;
  border: none;
}

.form-group {
  margin-bottom: 20px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-group label {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.label-hint {
  font-size: 11px;
  font-weight: normal;
  color: var(--text-muted);
}

.form-row {
  display: flex;
  gap: 16px;
}

.flex-1 { flex: 1; }
.w-120 { width: 120px; }
.w-140 { width: 140px; }

.select-wrapper {
  position: relative;
  width: 100%;
}

.custom-select {
  appearance: none;
  background: var(--bg-input, rgba(15, 23, 42, 0.85));
  border: 1.5px solid var(--border-control);
  border-radius: var(--radius-sm);
  padding: 10px 38px 10px 14px;
  color: var(--text-primary);
  font-size: 13px;
  font-family: inherit;
  transition: all 0.2s ease;
  width: 100%;
  cursor: pointer;
}

.custom-select:hover:not(:disabled) {
  border-color: var(--border-control-hover);
}

.custom-select:focus {
  outline: none;
  border-color: var(--accent-cyan);
  box-shadow: 0 0 0 2px rgba(6, 182, 212, 0.25);
}

.custom-select:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.custom-select option {
  background: var(--bg-card, #0f172a);
  color: var(--text-primary);
  padding: 8px;
}

.select-arrow {
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
  pointer-events: none;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
}

.select-arrow svg {
  width: 16px;
  height: 16px;
}


input[type="text"],
input[type="password"],
input[type="number"],
textarea {
  background: var(--bg-input, rgba(15, 23, 42, 0.85));
  border: 1.5px solid var(--border-control);
  border-radius: var(--radius-sm);
  padding: 10px 14px;
  color: var(--text-primary);
  font-size: 13px;
  font-family: inherit;
  transition: all 0.2s ease;
  width: 100%;
}

input:hover:not(:disabled), textarea:hover:not(:disabled) {
  border-color: var(--border-control-hover);
}

input:focus, textarea:focus {
  outline: none;
  border-color: var(--accent-cyan);
  box-shadow: 0 0 0 2px rgba(6, 182, 212, 0.25);
}

input:disabled, textarea:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.password-input {
  position: relative;
  display: flex;
  align-items: center;
}

.toggle-eye {
  position: absolute;
  right: 10px;
  background: transparent;
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
  transition: color 0.2s;
  padding: 4px;
}

.toggle-eye:hover {
  color: var(--text-primary);
}

.toggle-eye svg {
  width: 16px;
  height: 16px;
}

.toggles-list {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin: 24px 0;
  padding: 16px;
  background: var(--bg-card);
  border: 1.5px solid var(--border-control);
  border-radius: var(--radius-md);
}

.toggle-item {
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
}

.toggle-item input[type="checkbox"] {
  display: none;
}

.checkbox-box {
  width: 20px;
  height: 20px;
  border: 2px solid var(--border-control);
  border-radius: 5px;
  background: var(--bg-checkbox);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  flex-shrink: 0;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.25);
}

.toggle-item:hover .checkbox-box {
  border-color: var(--border-control-hover);
}

.toggle-item input[type="checkbox"]:checked + .checkbox-box {
  background: var(--accent-cyan);
  border-color: var(--accent-cyan);
  box-shadow: 0 0 12px rgba(6, 182, 212, 0.4);
}

.toggle-item input[type="checkbox"]:checked + .checkbox-box::after {
  content: '✓';
  color: #fff;
  font-size: 13px;
  font-weight: 900;
  line-height: 1;
}

.toggle-info {
  display: flex;
  flex-direction: column;
}

.toggle-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-primary);
}

.toggle-desc {
  font-size: 11px;
  color: var(--text-muted);
}

.form-actions {
  display: flex;
  justify-content: flex-end;
}

.btn {
  padding: 10px 22px;
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-weight: 600;
  border: none;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn.primary {
  background: linear-gradient(135deg, var(--accent-cyan), var(--accent-blue));
  color: #fff;
  box-shadow: 0 4px 14px rgba(6, 182, 212, 0.3);
}

.btn.primary:hover {
  transform: translateY(-1px);
  box-shadow: 0 6px 18px rgba(6, 182, 212, 0.4);
}

.btn.secondary {
  background: rgba(255, 255, 255, 0.08);
  color: var(--text-primary);
}

.btn.secondary:hover {
  background: rgba(255, 255, 255, 0.15);
}

[data-theme="light"] .btn.secondary {
  background: rgba(15, 23, 42, 0.08);
  color: var(--text-primary);
  border: 1px solid rgba(15, 23, 42, 0.12);
}

[data-theme="light"] .btn.secondary:hover {
  background: rgba(15, 23, 42, 0.15);
}

.btn.sm {
  padding: 6px 14px;
  font-size: 12px;
}

/* Terminal View */
.terminal-box {
  flex: 1;
  background: #060911;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 16px;
  overflow-y: auto;
  font-family: "Cascadia Code", Consolas, Menlo, monospace;
  font-size: 12px;
  line-height: 1.5;
  color: #38bdf8;
  max-height: 440px;
}

.terminal-box pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}

.empty-logs {
  color: var(--text-muted);
  font-style: italic;
}

/* Floating Toast */
.toast-notification {
  position: fixed;
  bottom: 24px;
  right: 28px;
  z-index: 9999;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 18px;
  background: rgba(15, 23, 42, 0.95);
  border: 1px solid rgba(16, 185, 129, 0.45);
  border-radius: var(--radius-md);
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.5), 0 0 20px rgba(16, 185, 129, 0.25);
  backdrop-filter: blur(16px);
  pointer-events: none;
}

.toast-icon {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: rgba(16, 185, 129, 0.2);
  color: var(--accent-emerald);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.toast-icon svg {
  width: 16px;
  height: 16px;
}

.toast-content {
  display: flex;
  flex-direction: column;
}

.toast-title {
  font-size: 13px;
  font-weight: 700;
  color: #fff;
}

.toast-msg {
  font-size: 12px;
  color: var(--accent-emerald);
}

.toast-enter-active,
.toast-leave-active {
  transition: all 0.35s cubic-bezier(0.16, 1, 0.3, 1);
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(16px) scale(0.95);
}
</style>
