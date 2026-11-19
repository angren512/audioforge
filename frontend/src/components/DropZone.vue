<script setup lang="ts">
import { store } from '../store'

const emit = defineEmits<{ (e: 'pick'): void }>()
</script>

<template>
  <div class="drop card">
    <div class="glow"></div>
    <div class="inner">
      <div class="icon">
        <svg width="46" height="46" viewBox="0 0 24 24" fill="none" stroke="url(#g)" stroke-width="1.4">
          <defs>
            <linearGradient id="g" x1="0" y1="0" x2="1" y2="1">
              <stop offset="0%" stop-color="#6c8cff" />
              <stop offset="100%" stop-color="#b06cff" />
            </linearGradient>
          </defs>
          <path d="M9 18V6l10-2v12" stroke-linecap="round" stroke-linejoin="round" />
          <circle cx="6.5" cy="18" r="2.5" />
          <circle cx="16.5" cy="16" r="2.5" />
        </svg>
      </div>
      <h2>把音频拖进窗口</h2>
      <p class="muted">支持文件夹，数量不限；也可直接选择文件</p>
      <button class="btn btn-primary pick" @click="emit('pick')">选择文件</button>
      <div class="supported">
        <span class="pill pill-enc">QQ音乐 .mgg / .mflac</span>
        <span class="pill pill-enc">酷狗 .kgm / .kgg</span>
        <span class="pill pill-enc">酷我 .kwm / .tm0</span>
        <span class="pill pill-enc">网易云 .ncm</span>
        <span class="pill">普通音频 21 种</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.drop {
  position: relative;
  overflow: hidden;
  padding: 2px;
  background:
    linear-gradient(135deg, rgba(108, 140, 255, 0.35), rgba(176, 108, 255, 0.15), rgba(108, 140, 255, 0.05));
}

.inner {
  background: rgba(11, 13, 18, 0.72);
  border-radius: 12px;
  padding: 26px 24px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  border: 1px dashed rgba(255, 255, 255, 0.12);
}

.glow {
  position: absolute;
  inset: -40% -10% auto -10%;
  height: 160px;
  background: radial-gradient(closest-side, rgba(108, 140, 255, 0.35), transparent);
  filter: blur(24px);
  pointer-events: none;
}

.icon {
  margin-bottom: 2px;
}

h2 {
  margin: 0;
  font-size: 17px;
  font-weight: 600;
  letter-spacing: 0.4px;
}

p {
  margin: 0;
  font-size: 12.5px;
}

.pick {
  margin-top: 10px;
}

.supported {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 6px;
  margin-top: 14px;
}
</style>
