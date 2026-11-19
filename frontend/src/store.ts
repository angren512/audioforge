import { computed, reactive } from 'vue'
import type { FormatSpec, Settings, Stats, Task } from './types'
import {
  AddFiles,
  CancelAll,
  CancelTask,
  ClearFinished,
  GetBootstrap,
  PickDir,
  PickFiles,
  SaveSettings,
  Snapshot,
  Start,
} from '../wailsjs/go/main/App'
import { EventsOn } from '../wailsjs/runtime/runtime'

const defaultSettings: Settings = {
  outputDir: '',
  format: 'mp3',
  bitrate: '320k',
  useVbr: false,
  quality: 2,
  sampleRate: 0,
  channels: 0,
  compression: 5,
  bitDepth: 16,
  workers: 4,
  acceptedDisclaimer: false,
  kggDbPath: '',
}

const state = reactive({
  ready: false,
  version: '',
  settings: { ...defaultSettings } as Settings,
  formats: [] as FormatSpec[],
  tasks: [] as Task[],
  stats: { total: 0, queued: 0, running: 0, done: 0, failed: 0, canceled: 0, skipped: 0 } as Stats,
  filter: 'all' as 'all' | 'active' | 'done' | 'failed',
  started: false,
  toast: '',
})

let toastTimer: number | undefined

function toast(msg: string) {
  state.toast = msg
  window.clearTimeout(toastTimer)
  toastTimer = window.setTimeout(() => (state.toast = ''), 2600)
}

export function useStore() {
  const busy = computed(() => state.stats.running > 0 || state.stats.queued > 0)
  const canStart = computed(() => state.tasks.some((t) => t.state === 'queued') || !state.started)
  const activeFormat = computed(
    () => state.formats.find((f) => f.id === state.settings.format) || state.formats[0],
  )

  async function init() {
    const boot = await GetBootstrap()
    state.settings = { ...defaultSettings, ...boot.settings }
    state.formats = boot.formats
    state.version = boot.version
    state.ready = true

    EventsOn('task:update', (raw: unknown) => {
      const t = raw as Task
      const idx = state.tasks.findIndex((x) => x.id === t.id)
      if (idx >= 0) state.tasks[idx] = t
      else state.tasks.push(t)
    })
    EventsOn('stats', (s: Stats) => {
      state.stats = s
    })

    const existing = (await Snapshot()) as unknown as Task[]
    if (existing && existing.length) state.tasks = existing
  }

  async function pickFiles() {
    const files = await PickFiles()
    if (files && files.length) await addPaths(files)
  }

  async function addPaths(paths: string[]) {
    const added = (await AddFiles(paths)) as unknown as Task[]
    if (added && added.length) {
      const hasSoda = added.some((t) => t.name.toLowerCase().endsWith('.mdl'))
      if (hasSoda) toast('汽水音乐 .mdl 需要联网鉴权，暂不支持离线解密')
      for (const t of added) {
        if (!state.tasks.some((x) => x.id === t.id)) state.tasks.push(t)
      }
    }
  }

  async function start() {
    if (!state.started) toast('正在准备内置转码组件，首次运行可能需要几秒…')
    try {
      await Start({
        outputDir: state.settings.outputDir,
        format: state.settings.format,
        bitrate: state.settings.bitrate,
        useVbr: state.settings.useVbr,
        quality: state.settings.quality,
        sampleRate: state.settings.sampleRate,
        channels: state.settings.channels,
        compression: state.settings.compression,
        bitDepth: state.settings.bitDepth,
        workers: state.settings.workers,
        kggDbPath: state.settings.kggDbPath,
      })
      state.started = true
      persist()
    } catch (e) {
      toast(String(e))
    }
  }

  async function cancelAll() {
    await CancelAll()
  }

  async function cancelTask(id: string) {
    await CancelTask(id)
  }

  async function clearFinished() {
    await ClearFinished()
    state.tasks = state.tasks.filter(
      (t) => !['done', 'failed', 'canceled', 'skipped'].includes(t.state),
    )
  }

  async function chooseDir() {
    const dir = await PickDir()
    if (dir) {
      state.settings.outputDir = dir
      persist()
    }
  }

  function persist() {
    SaveSettings({ ...state.settings })
  }

  function setFilter(f: 'all' | 'active' | 'done' | 'failed') {
    state.filter = f
  }

  const filtered = computed(() => {
    switch (state.filter) {
      case 'active':
        return state.tasks.filter((t) => !['done', 'failed', 'canceled', 'skipped'].includes(t.state))
      case 'done':
        return state.tasks.filter((t) => t.state === 'done')
      case 'failed':
        return state.tasks.filter((t) => ['failed', 'canceled'].includes(t.state))
      default:
        return state.tasks
    }
  })

  return {
    state,
    busy,
    canStart,
    activeFormat,
    filtered,
    init,
    pickFiles,
    addPaths,
    start,
    cancelAll,
    cancelTask,
    clearFinished,
    chooseDir,
    persist,
    setFilter,
    toast,
  }
}

export const store = useStore()
