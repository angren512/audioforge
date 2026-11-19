<script setup lang="ts">
import { onMounted, ref } from 'vue'
import DropZone from './components/DropZone.vue'
import OutputSettings from './components/OutputSettings.vue'
import TaskTable from './components/TaskTable.vue'
import StatBar from './components/StatBar.vue'
import { store } from './store'
import { OpenLog } from '../wailsjs/go/main/App'

const showAbout = ref(false)
const showDisclaimer = ref(false)

onMounted(async () => {
  await store.init()
  if (!store.state.settings.acceptedDisclaimer) showDisclaimer.value = true
})

function acceptDisclaimer() {
  store.state.settings.acceptedDisclaimer = true
  store.persist()
  showDisclaimer.value = false
}
</script>

<template>
  <div class="app">
    <header class="topbar">
      <div class="brand">
        <div class="logo">
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="1.8">
            <path d="M9 18V6l10-2v12" stroke-linecap="round" stroke-linejoin="round" />
            <circle cx="6.5" cy="18" r="2.5" />
            <circle cx="16.5" cy="16" r="2.5" />
          </svg>
        </div>
        <div class="titles">
          <h1>音频工坊</h1>
          <span class="muted">AudioForge · 批量解锁与转换</span>
        </div>
      </div>
      <div class="top-actions">
        <button class="btn" @click="OpenLog()">日志</button>
        <button class="btn" @click="showAbout = true">关于</button>
      </div>
    </header>

    <main>
      <div class="left">
        <DropZone @pick="store.pickFiles()" />
        <TaskTable />
      </div>
      <OutputSettings />
    </main>

    <StatBar />

    <transition name="fade">
      <div v-if="store.state.toast" class="toast">{{ store.state.toast }}</div>
    </transition>

    <div v-if="showAbout" class="overlay" @click.self="showAbout = false">
      <div class="modal card">
        <h3>关于音频工坊</h3>
        <p class="muted">
          版本 {{ store.state.version }}。把各音乐平台下载的加密文件还原为普通音频，并批量转换为常用格式。
          所有处理均在本机完成，全程离线，不上传任何文件。
        </p>
        <ul class="muted">
          <li>解密引擎：unlock-music（MIT）</li>
          <li>转码引擎：内置静态 FFmpeg 6.1.1</li>
          <li>加密格式：QQ音乐 / 酷狗 / 酷我 / 网易云 / 虾米 / 喜马拉雅 / 咪咕</li>
        </ul>
        <p class="muted small">
          汽水音乐 .mdl 为分段加密且依赖联网鉴权，目前不支持。请仅对已购买的内容做本地备份。
        </p>
        <div class="modal-actions">
          <button class="btn btn-primary" @click="showAbout = false">知道了</button>
        </div>
      </div>
    </div>

    <div v-if="showDisclaimer" class="overlay">
      <div class="modal card">
        <h3>使用须知</h3>
        <p class="muted">
          本工具用于把你自己已购买或已开通会员期间下载的音乐，还原为通用格式并在本地备份。
          请勿用于传播、售卖或任何侵犯著作权与平台协议的行为，由此产生的责任由使用者自行承担。
        </p>
        <p class="muted small">所有转换均在本机离线完成，程序不会访问网络。</p>
        <div class="modal-actions">
          <button class="btn btn-primary" @click="acceptDisclaimer">我已了解，开始使用</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.app {
  height: 100%;
  display: grid;
  grid-template-rows: auto 1fr auto;
  overflow: hidden;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px 14px;
}

.brand {
  display: flex;
  align-items: center;
  gap: 11px;
}

.logo {
  width: 36px;
  height: 36px;
  border-radius: 11px;
  display: grid;
  place-items: center;
  background: linear-gradient(135deg, var(--accent), var(--accent-2));
  box-shadow: 0 8px 22px rgba(108, 140, 255, 0.32);
}

.titles {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

h1 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  letter-spacing: 0.5px;
}

.titles span {
  font-size: 11.5px;
  letter-spacing: 0.3px;
}

.top-actions {
  display: flex;
  gap: 8px;
}

main {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 356px;
  gap: 14px;
  padding: 0 20px 14px;
  min-height: 0;
  overflow: hidden;
}

.left {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-height: 0;
  overflow: hidden;
}

.toast {
  position: fixed;
  left: 50%;
  bottom: 76px;
  transform: translateX(-50%);
  background: rgba(30, 34, 48, 0.95);
  border: 1px solid var(--border-strong);
  color: var(--text);
  padding: 10px 18px;
  border-radius: 10px;
  font-size: 12.5px;
  box-shadow: 0 12px 30px rgba(0, 0, 0, 0.45);
  z-index: 30;
}

.overlay {
  position: fixed;
  inset: 0;
  background: rgba(5, 6, 10, 0.66);
  backdrop-filter: blur(6px);
  display: grid;
  place-items: center;
  z-index: 20;
}

.modal {
  width: 460px;
  padding: 22px;
  background: #141821;
  border: 1px solid var(--border-strong);
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.modal h3 {
  margin: 0;
  font-size: 15px;
}

.modal p {
  margin: 0;
  font-size: 12.5px;
  line-height: 1.75;
}

.modal ul {
  margin: 0;
  padding-left: 18px;
  font-size: 12px;
  line-height: 1.9;
}

.small {
  font-size: 11.5px;
  opacity: 0.8;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 4px;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
