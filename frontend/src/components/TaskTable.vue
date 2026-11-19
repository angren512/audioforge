<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { store } from '../store'
import type { Task } from '../types'

const ROW = 62
const body = ref<HTMLElement | null>(null)
const scrollTop = ref(0)
const viewH = ref(400)

const tasks = computed(() => store.filtered.value)

onMounted(() => {
  measure()
  window.addEventListener('resize', measure)
})

function measure() {
  if (body.value) viewH.value = body.value.clientHeight || 400
}

watch(
  () => store.state.filter,
  () => {
    scrollTop.value = 0
    if (body.value) body.value.scrollTop = 0
    nextTick(measure)
  },
)

function onScroll() {
  scrollTop.value = body.value?.scrollTop ?? 0
}

const start = computed(() => Math.max(0, Math.floor(scrollTop.value / ROW) - 4))
const end = computed(() => Math.min(tasks.value.length, Math.ceil((scrollTop.value + viewH.value) / ROW) + 4))
const visible = computed(() => tasks.value.slice(start.value, end.value))
const totalH = computed(() => tasks.value.length * ROW)

const stateText: Record<string, string> = {
  queued: '等待',
  decrypting: '解密中',
  probing: '分析中',
  converting: '转换中',
  done: '完成',
  failed: '失败',
  canceled: '已取消',
  skipped: '跳过',
}

function pillClass(t: Task) {
  if (t.state === 'done') return 'pill-ok'
  if (t.state === 'failed') return 'pill-err'
  if (t.state === 'canceled') return 'pill'
  if (t.state === 'queued') return 'pill'
  return 'pill-run'
}

function fmtDuration(sec?: number) {
  if (!sec || sec <= 0) return ''
  const m = Math.floor(sec / 60)
  const s = Math.round(sec % 60)
  return `${m}:${String(s).padStart(2, '0')}`
}

function fmtSize(info: Task['info']) {
  if (!info?.bitRate) return ''
  const kbps = Math.round(info.bitRate / 1000)
  return `${kbps} kbps · ${info.sampleRate / 1000} kHz`
}

const filters = [
  { id: 'all', label: '全部' },
  { id: 'active', label: '进行中' },
  { id: 'done', label: '已完成' },
  { id: 'failed', label: '失败' },
] as const
</script>

<template>
  <section class="table card">
    <header class="thead">
      <div class="tabs">
        <button
          v-for="f in filters"
          :key="f.id"
          :class="{ on: store.state.filter === f.id }"
          @click="store.setFilter(f.id)"
        >
          {{ f.label }}
        </button>
      </div>
      <div class="count muted">共 {{ tasks.length }} 项</div>
    </header>

    <div v-if="!tasks.length" class="empty">
      <div class="empty-icon">♪</div>
      <p>队列是空的</p>
      <span class="muted">把文件或文件夹拖进窗口，或点上方「选择文件」</span>
    </div>

    <div v-else ref="body" class="tbody" @scroll="onScroll">
      <div class="spacer" :style="{ height: totalH + 'px' }">
        <div class="rows" :style="{ transform: `translateY(${start * ROW}px)` }">
          <div v-for="t in visible" :key="t.id" class="row">
            <div class="col-main">
              <div class="line1">
                <span class="name" :title="t.src">{{ t.name }}</span>
                <span v-if="t.encrypted" class="pill pill-enc">加密</span>
                <span class="pill">{{ t.ext.replace('.', '').toUpperCase() }}</span>
                <span v-if="t.info" class="meta">{{ fmtSize(t.info) }}</span>
                <span v-if="t.info" class="meta">{{ fmtDuration(t.info.duration) }}</span>
              </div>
              <div class="line2">
                <div class="bar">
                  <div
                    class="bar-fill"
                    :class="{
                      run: ['decrypting', 'probing', 'converting'].includes(t.state),
                      ok: t.state === 'done',
                      err: t.state === 'failed',
                    }"
                    :style="{ width: (t.state === 'done' ? 100 : t.progress) + '%' }"
                  ></div>
                </div>
                <span class="msg muted">{{ t.message }}</span>
              </div>
            </div>
            <div class="col-side">
              <span class="pill" :class="pillClass(t)">{{ stateText[t.state] || t.state }}</span>
              <button
                v-if="['queued', 'decrypting', 'probing', 'converting'].includes(t.state)"
                class="mini"
                @click="store.cancelTask(t.id)"
              >
                取消
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.table {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}

.thead {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px 10px;
  border-bottom: 1px solid var(--border);
}

.tabs {
  display: flex;
  gap: 4px;
}

.tabs button {
  height: 28px;
  padding: 0 12px;
  border-radius: 8px;
  font-size: 12.5px;
  color: var(--muted);
  transition: all 0.15s ease;
}

.tabs button:hover {
  color: var(--text);
}

.tabs button.on {
  color: var(--text);
  background: rgba(255, 255, 255, 0.07);
}

.count {
  font-size: 12px;
}

.tbody {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}

.spacer {
  position: relative;
}

.rows {
  position: absolute;
  left: 0;
  right: 0;
  top: 0;
}

.row {
  height: 62px;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 16px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
}

.row:hover {
  background: rgba(255, 255, 255, 0.025);
}

.col-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.line1 {
  display: flex;
  align-items: center;
  gap: 7px;
  min-width: 0;
}

.name {
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 46%;
}

.meta {
  font-size: 11px;
  color: var(--muted);
  white-space: nowrap;
}

.line2 {
  display: flex;
  align-items: center;
  gap: 10px;
}

.bar {
  flex: 1;
  max-width: 320px;
  height: 4px;
  border-radius: 3px;
  background: rgba(255, 255, 255, 0.07);
  overflow: hidden;
}

.bar-fill {
  height: 100%;
  width: 0;
  border-radius: 3px;
  background: rgba(255, 255, 255, 0.25);
  transition: width 0.2s linear;
}

.bar-fill.run {
  background: linear-gradient(90deg, var(--accent), var(--accent-2));
  background-size: 200% 100%;
  animation: flow 1.6s linear infinite;
}

.bar-fill.ok {
  background: var(--ok);
}

.bar-fill.err {
  background: var(--err);
}

@keyframes flow {
  from {
    background-position: 0 0;
  }
  to {
    background-position: 200% 0;
  }
}

.msg {
  font-size: 11px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 300px;
}

.col-side {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: none;
}

.mini {
  font-size: 12px;
  color: var(--muted);
  padding: 4px 8px;
  border-radius: 7px;
  border: 1px solid var(--border);
}

.mini:hover {
  color: var(--err);
  border-color: rgba(248, 113, 113, 0.4);
}

.empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.empty-icon {
  font-size: 30px;
  color: rgba(255, 255, 255, 0.16);
}

.empty p {
  margin: 0;
  font-size: 14px;
}

.empty span {
  font-size: 12px;
}
</style>
