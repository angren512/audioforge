export interface Settings {
  outputDir: string
  format: string
  bitrate: string
  useVbr: boolean
  quality: number
  sampleRate: number
  channels: number
  compression: number
  bitDepth: number
  workers: number
  acceptedDisclaimer: boolean
  kggDbPath: string
}

export interface FormatSpec {
  id: string
  name: string
  ext: string
  lossless: boolean
  cover: boolean
  hint: string
}

export interface Bootstrap {
  settings: Settings
  formats: FormatSpec[]
  encryptedExts: string[]
  plainExts: string[]
  version: string
  logPath: string
}

export interface TaskInfo {
  container: string
  codec: string
  duration: number
  sampleRate: number
  channels: number
  bitRate: number
  hasCover: boolean
}

export type TaskState =
  | 'queued'
  | 'decrypting'
  | 'probing'
  | 'converting'
  | 'done'
  | 'failed'
  | 'canceled'
  | 'skipped'

export interface Task {
  id: string
  src: string
  name: string
  ext: string
  encrypted: boolean
  state: TaskState
  progress: number
  message: string
  output: string
  info: TaskInfo | null
}

export interface Stats {
  total: number
  queued: number
  running: number
  done: number
  failed: number
  canceled: number
  skipped: number
}
