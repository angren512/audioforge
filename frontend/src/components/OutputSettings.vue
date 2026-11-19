<script setup lang="ts">
import { computed } from 'vue'
import { store } from '../store'

const s = store.state

const active = computed(() => s.formats.find((f) => f.id === s.settings.format))

const qualityKind = computed(() => {
  switch (s.settings.format) {
    case 'mp3':
      return s.settings.useVbr ? 'mp3-vbr' : 'bitrate'
    case 'aac':
    case 'opus':
    case 'wma':
      return 'bitrate'
    case 'ogg':
      return 'vorbis'
    case 'flac':
      return 'flac'
    case 'wav':
    case 'aiff':
      return 'depth'
    default:
      return 'none'
  }
})

const bitrates = computed(() => {
  switch (s.settings.format) {
    case 'aac':
      return ['128k', '192k', '256k']
    case 'opus':
      return ['64k', '96k', '128k', '160k', '192k', '256k']
    case 'wma':
      return ['128k', '192k']
    default:
      return ['128k', '192k', '256k', '320k']
  }
})

function pickFormat(id: string) {
  s.settings.format = id
  if (!bitrates.value.includes(s.settings.bitrate)) {
    s.settings.bitrate = bitrates.value[bitrates.value.length - 1]
  }
  store.persist()
}
</script>

<template>
  <aside class="panel card">
    <div class="block">
      <div class="section-title">输出格式</div>
      <div class="formats">
        <button
          v-for="f in s.formats"
          :key="f.id"
          class="fmt"
          :class="{ on: s.settings.format === f.id }"
          @click="pickFormat(f.id)"
        >
          <span class="fmt-name">{{ f.name }}</span>
          <span class="fmt-hint">{{ f.hint }}</span>
          <span v-if="f.lossless" class="tag">无损</span>
        </button>
      </div>
    </div>

    <div class="block">
      <div class="section-title">音质</div>

      <template v-if="qualityKind === 'bitrate'">
        <div class="seg">
          <button
            v-for="b in bitrates"
            :key="b"
            :class="{ on: s.settings.bitrate === b }"
            @click="((s.settings.bitrate = b), store.persist())"
          >
            {{ b.replace('k', '') }}
          </button>
        </div>
        <div class="hint">kbps，数值越高音质越好、体积越大</div>
      </template>

      <template v-else-if="qualityKind === 'mp3-vbr'">
        <div class="seg">
          <button
            v-for="q in [0, 1, 2, 3, 4]"
            :key="q"
            :class="{ on: s.settings.quality === q }"
            @click="((s.settings.quality = q), store.persist())"
          >
            V{{ q }}
          </button>
        </div>
        <div class="hint">V0 质量最高，V4 体积最小</div>
      </template>

      <template v-else-if="qualityKind === 'vorbis'">
        <div class="seg">
          <button
            v-for="q in [3, 5, 6, 8, 10]"
            :key="q"
            :class="{ on: s.settings.quality === q }"
            @click="((s.settings.quality = q), store.persist())"
          >
            q{{ q }}
          </button>
        </div>
        <div class="hint">Vorbis 质量等级，q6 约等于 192kbps</div>
      </template>

      <template v-else-if="qualityKind === 'flac'">
        <div class="seg">
          <button
            v-for="c in [0, 5, 8, 12]"
            :key="c"
            :class="{ on: s.settings.compression === c }"
            @click="((s.settings.compression = c), store.persist())"
          >
            L{{ c }}
          </button>
        </div>
        <div class="hint">压缩级别，不影响音质，只影响体积与耗时</div>
      </template>

      <template v-else-if="qualityKind === 'depth'">
        <div class="seg">
          <button
            v-for="d in [16, 24]"
            :key="d"
            :class="{ on: s.settings.bitDepth === d }"
            @click="((s.settings.bitDepth = d), store.persist())"
          >
            {{ d }} bit
          </button>
        </div>
        <div class="hint">位深越高动态范围越大</div>
      </template>

      <template v-else>
        <div class="hint">{{ active?.name }} 为固定无损编码，无需调整音质</div>
      </template>

      <label v-if="s.settings.format === 'mp3'" class="switch">
        <input
          type="checkbox"
          v-model="s.settings.useVbr"
          @change="store.persist()"
        />
        <span>使用 VBR 可变码率</span>
      </label>
    </div>

    <div class="block">
      <div class="section-title">采样与声道</div>
      <div class="row">
        <span class="label">采样率</span>
        <select v-model.number="s.settings.sampleRate" @change="store.persist()">
          <option :value="0">保持原始</option>
          <option :value="44100">44100 Hz</option>
          <option :value="48000">48000 Hz</option>
        </select>
      </div>
      <div class="row">
        <span class="label">声道</span>
        <select v-model.number="s.settings.channels" @change="store.persist()">
          <option :value="0">保持原始</option>
          <option :value="1">单声道</option>
          <option :value="2">立体声</option>
        </select>
      </div>
    </div>

    <div class="block">
      <div class="section-title">输出位置</div>
      <div class="dir">
        <input :value="s.settings.outputDir || '与原文件相同目录'" readonly />
        <button class="btn" @click="store.chooseDir()">浏览</button>
      </div>
      <div v-if="s.settings.outputDir" class="row" style="margin-top: 8px">
        <span class="label"></span>
        <button class="link" @click="((s.settings.outputDir = ''), store.persist())">
          改回与原文件同目录
        </button>
      </div>
    </div>

    <div class="block">
      <div class="section-title">并发任务</div>
      <div class="row">
        <input
          class="range"
          type="range"
          min="1"
          max="16"
          v-model.number="s.settings.workers"
          @change="store.persist()"
        />
        <span class="num">{{ s.settings.workers }}</span>
      </div>
      <div class="hint">同时转换的文件数，过高会占满 CPU</div>
    </div>
  </aside>
</template>

<style scoped>
.panel {
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  overflow-y: auto;
  height: 100%;
}

.block {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.formats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 7px;
}

.fmt {
  position: relative;
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 9px 6px 8px;
  background: rgba(255, 255, 255, 0.02);
  text-align: center;
  transition: all 0.16s ease;
}

.fmt:hover {
  border-color: var(--border-strong);
  background: var(--panel-hover);
}

.fmt.on {
  border-color: rgba(108, 140, 255, 0.7);
  background: rgba(108, 140, 255, 0.12);
  box-shadow: 0 0 0 1px rgba(108, 140, 255, 0.25) inset;
}

.fmt-name {
  display: block;
  font-size: 13px;
  font-weight: 600;
}

.fmt-hint {
  display: block;
  font-size: 10px;
  color: var(--muted);
  margin-top: 3px;
  transform: scale(0.95);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tag {
  position: absolute;
  top: -6px;
  right: 6px;
  font-size: 9.5px;
  padding: 1px 5px;
  border-radius: 6px;
  color: var(--ok);
  background: rgba(53, 211, 154, 0.15);
  border: 1px solid rgba(53, 211, 154, 0.3);
}

.seg {
  display: flex;
  gap: 5px;
  flex-wrap: wrap;
}

.seg button {
  flex: 1;
  min-width: 46px;
  height: 32px;
  border-radius: 8px;
  border: 1px solid var(--border);
  font-size: 12.5px;
  color: var(--muted);
  transition: all 0.15s ease;
}

.seg button:hover {
  color: var(--text);
  border-color: var(--border-strong);
}

.seg button.on {
  color: #fff;
  border-color: rgba(108, 140, 255, 0.65);
  background: rgba(108, 140, 255, 0.18);
}

.hint {
  font-size: 11.5px;
  color: var(--muted);
}

.row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.label {
  width: 56px;
  font-size: 12.5px;
  color: var(--muted);
  flex: none;
}

select,
.dir input {
  flex: 1;
  height: 34px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border);
  border-radius: 9px;
  color: var(--text);
  padding: 0 10px;
  font-size: 12.5px;
  outline: none;
  font-family: inherit;
}

select:focus {
  border-color: rgba(108, 140, 255, 0.6);
}

.dir {
  display: flex;
  gap: 8px;
}

.dir input {
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--muted);
}

.dir .btn {
  height: 34px;
  padding: 0 14px;
}

.switch {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12.5px;
  color: var(--muted);
  cursor: pointer;
}

.switch input {
  accent-color: var(--accent);
}

.range {
  flex: 1;
  accent-color: var(--accent);
}

.num {
  width: 26px;
  text-align: right;
  font-variant-numeric: tabular-nums;
  color: var(--text);
}

.link {
  font-size: 12px;
  color: var(--accent);
  padding: 0;
}

.link:hover {
  text-decoration: underline;
}
</style>
