<script setup lang="ts">
import { computed } from 'vue'
import { store } from '../store'
import { OpenOutputDir } from '../../wailsjs/go/main/App'

const s = store.state
const pending = computed(() => s.tasks.filter((t) => t.state === 'queued').length)
// stats.queued 包含「已添加但还没开始」的任务，不能当成正在执行。
// 只有真的在跑才算忙；worker 常驻，点过一次开始后新加的文件会自动处理。
const busy = computed(() => s.stats.running > 0)

function openDir() {
  const dir = s.settings.outputDir
  if (dir) OpenOutputDir(dir)
  else {
    const done = s.tasks.find((t) => t.state === 'done' && t.output)
    if (done) OpenOutputDir(done.output.replace(/[\\/][^\\/]+$/, ''))
  }
}
</script>

<template>
  <footer class="statbar">
    <div class="stats">
      <div class="stat">
        <span class="num">{{ s.stats.total }}</span>
        <span class="k muted">总计</span>
      </div>
      <div class="stat">
        <span class="num ok">{{ s.stats.done }}</span>
        <span class="k muted">成功</span>
      </div>
      <div class="stat">
        <span class="num err">{{ s.stats.failed }}</span>
        <span class="k muted">失败</span>
      </div>
      <div class="stat">
        <span class="num run">{{ s.stats.running }}</span>
        <span class="k muted">进行中</span>
      </div>
    </div>

    <div class="actions">
      <button class="btn" @click="store.clearFinished()">清空已完成</button>
      <button class="btn" @click="openDir">打开输出目录</button>
      <button v-if="busy" class="btn btn-danger" @click="store.cancelAll()">取消全部</button>
      <button
        class="btn btn-primary start"
        :disabled="busy || pending === 0"
        @click="store.start()"
      >
        {{ busy ? '转换中…' : pending ? `开始转换 (${pending})` : '开始转换' }}
      </button>
    </div>
  </footer>
</template>

<style scoped>
.statbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 20px;
  border-top: 1px solid var(--border);
  background: rgba(11, 13, 18, 0.72);
  backdrop-filter: blur(12px);
}

.stats {
  display: flex;
  gap: 26px;
}

.stat {
  display: flex;
  align-items: baseline;
  gap: 6px;
}

.num {
  font-size: 17px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.num.ok {
  color: var(--ok);
}

.num.err {
  color: var(--err);
}

.num.run {
  color: var(--accent);
}

.k {
  font-size: 11.5px;
}

.actions {
  display: flex;
  gap: 8px;
}

.start {
  min-width: 132px;
}
</style>
