<template>
  <div class="player-tabs-wrapper" ref="playerWrapper">
    <el-tabs v-if="showTab && playerList.length > 1" v-model="activePlayer" type="card" :stretch="true" @tab-click="changePlayer">
      <el-tab-pane v-for="p in playerList" :key="p.key" :label="p.label" :name="p.key"></el-tab-pane>
    </el-tabs>
    <div class="player-video-area">
      <jessibucaPlayer
        v-if="activePlayer === 'jessibuca'"
        ref="jessibuca"
        class="player-instance"
        :has-audio="hasAudio"
        :show-button="showButton && !showControlBar"
        fluent autoplay live
        @playTimeChange="$emit('playTimeChange', $event)"
        @playStatusChange="onPlayStatusChange"
      />
      <rtc-player
        v-if="activePlayer === 'webRTC'"
        ref="webRTC"
        class="player-instance"
        :has-audio="hasAudio"
        :show-controls="!showControlBar"
        fluent autoplay live
        @playTimeChange="$emit('playTimeChange', $event)"
        @playStatusChange="onPlayStatusChange"
      />
      <h265web
        v-if="activePlayer === 'h265web'"
        ref="h265web"
        class="player-instance"
        :has-audio="hasAudio"
        :show-button="showButton && !showControlBar"
        fluent autoplay live
        @playTimeChange="$emit('playTimeChange', $event)"
        @playStatusChange="onPlayStatusChange"
      />
      <!-- 起播倒计时：点击播放后叠一个 3→2→1，真正出画即撤掉（所以常常 3 还没到 2 就开播了）。 -->
      <transition name="cd-fade">
        <div v-if="countdownVisible" class="player-countdown">
          <div class="player-countdown-num">{{ countdownNum }}</div>
        </div>
      </transition>
    </div>
    <!-- 播放控制栏：固定在画面下方，不再是悬浮在画面里的图标。 -->
    <div v-if="showControlBar" class="player-control-bar">
      <button
        v-if="activePlayer !== 'webRTC'"
        class="pcb-btn"
        :title="playerPlaying ? '暂停' : '播放'"
        @click="togglePlay"
      >
        <i :class="playerPlaying ? 'el-icon-video-pause' : 'el-icon-video-play'" />
      </button>
      <button class="pcb-btn" :title="ctrlMuted ? '取消静音' : '静音'" @click="toggleMute">
        <i :class="ctrlMuted ? 'iconfont icon-audio-mute' : 'iconfont icon-audio-high'" />
      </button>
      <button class="pcb-btn" title="截图" @click="doScreenshot">
        <i class="el-icon-camera" />
      </button>
      <button class="pcb-btn" title="刷新重连" @click="doRefresh">
        <i class="el-icon-refresh-right" />
      </button>
      <span class="pcb-spacer" />
      <button class="pcb-btn" :title="docFull ? '退出全屏' : '全屏'" @click="toggleFullscreen">
        <i :class="docFull ? 'el-icon-close' : 'el-icon-full-screen'" />
      </button>
    </div>
  </div>
</template>

<script>
import jessibucaPlayer from './jessibuca.vue'
import rtcPlayer from './rtcPlayer.vue'
import h265web from './h265web.vue'
import { resolvePlayerStrategy, PLAYER_LABELS, normalizeVideoCodec } from '@/utils/playerStrategy'

const URL_KEYS = {
  jessibuca: ['flv', 'https_flv', 'ws_flv', 'wss_flv'],
  h265web: ['flv', 'https_flv', 'ws_flv', 'wss_flv'],
  webRTC: ['rtc', 'rtcs']
}

export default {
  name: 'PlayerTabs',
  components: { jessibucaPlayer, rtcPlayer, h265web },
  props: {
    hasAudio: { type: Boolean, default: false },
    showButton: { type: Boolean, default: true },
    showTab: { type: Boolean, default: true },
    /** 限定可用播放器（覆盖策略），如 ['jessibuca'] */
    allowedPlayers: { type: Array, default: null },
    /** 覆盖默认 URL 优先级，如 ['flv','ws_flv'] */
    urlPriority: { type: Array, default: null },
    /** 首选播放器（覆盖策略），如 jessibuca / h265web / webRTC */
    preferredPlayer: { type: String, default: '' },
    /** 是否展示底部播放控制栏（并隐藏引擎内悬浮按钮）。弹窗播放器用。 */
    showControlBar: { type: Boolean, default: false }
  },
  data() {
    return {
      streamInfo: null,
      activePlayer: 'jessibuca',
      playerPlaying: false,
      ctrlMuted: false,
      docFull: false,
      countdownVisible: false,
      countdownNum: 3,
      countdownTimer: null
    }
  },
  computed: {
    /** ZMS 是否具备 WebRTC 能力：点播结果带 rtc/rtcs 即代表可用 */
    rtcAvailable() {
      const s = this.streamInfo
      return !!(s && (s.rtc || s.rtcs))
    },
    strategy() {
      return resolvePlayerStrategy({
        videoCodec: this.resolveVideoCodec(this.streamInfo),
        audioCodec: this.resolveAudioCodec(this.streamInfo),
        hasAudio: this.hasAudio,
        rtcAvailable: this.rtcAvailable
      })
    },
    allowedKeys() {
      if (this.allowedPlayers && this.allowedPlayers.length) {
        return this.allowedPlayers
      }
      return this.strategy.allowedPlayers
    },
    playerList() {
      return this.allowedKeys.map(key => ({ key, label: PLAYER_LABELS[key] || key }))
    },
    playerCount() {
      return this.playerList.length
    }
  },
  created() {
    if (this.playerCount === 1) {
      this.activePlayer = this.playerList[0].key
    }
  },
  mounted() {
    document.addEventListener('fullscreenchange', this.onFullscreenChange)
  },
  beforeDestroy() {
    document.removeEventListener('fullscreenchange', this.onFullscreenChange)
    this.stopCountdown()
  },
  watch: {
    // 全局播放模式变化（任意页面切换后）：按可用能力重新落位到具体引擎
    preferredPlayer() {
      this.$nextTick(() => this.reapplyPreference())
    },
    activePlayer() {
      this.playerPlaying = false
      this.$nextTick(() => this.syncMuteState())
    }
  },
  methods: {
    /** 按当前 preferredPlayer/策略重新选择引擎；仅在变化时重启播放。 */
    reapplyPreference() {
      const before = this.activePlayer
      this.ensureActivePlayer()
      if (this.activePlayer === before) return
      if (this.$refs[before] && this.$refs[before].pause) {
        this.$refs[before].pause()
      }
      this.$nextTick(() => {
        this.play()
        this.syncPlayerSize()
      })
      this.$emit('player-changed', this.activePlayer)
    },
    getPlayerList() {
      return this.playerList
    },
    /* ---------------- 播放控制栏（面向用户，隐藏引擎内悬浮按钮） ---------------- */
    activeEngine() {
      return this.$refs[this.activePlayer]
    },
    onPlayStatusChange(playing) {
      if (playing) {
        this.stopCountdown()
      }
      this.playerPlaying = !!playing
      this.$emit('playStatusChange', playing)
    },
    syncMuteState() {
      const p = this.activeEngine()
      if (!p) return
      if (this.activePlayer === 'webRTC') {
        const v = p.$refs && p.$refs.video
        this.ctrlMuted = v ? !!v.muted : false
      } else if (typeof p.isNotMute === 'boolean') {
        this.ctrlMuted = p.isNotMute === false
      }
    },
    togglePlay() {
      if (this.activePlayer === 'webRTC') return
      const p = this.activeEngine()
      if (!p) return
      if (this.playerPlaying) {
        p.pause && p.pause()
      } else if (this.activePlayer === 'h265web') {
        p.unPause && p.unPause()
      } else {
        p.playBtnClick && p.playBtnClick()
      }
    },
    toggleMute() {
      const p = this.activeEngine()
      if (!p) return
      if (this.activePlayer === 'webRTC') {
        const v = p.$refs && p.$refs.video
        if (v) { v.muted = !v.muted; this.ctrlMuted = v.muted }
        return
      }
      if (typeof p.isNotMute === 'boolean') {
        if (p.isNotMute) { p.mute && p.mute() } else { p.cancelMute && p.cancelMute() }
        this.ctrlMuted = !p.isNotMute
      }
    },
    doScreenshot() {
      const p = this.activeEngine()
      if (!p) return
      if (this.activePlayer === 'webRTC') {
        this.screenshotWebRTC(p)
      } else if (p.screenshot) {
        p.screenshot()
      }
    },
    screenshotWebRTC(p) {
      const v = p.$refs && p.$refs.video
      if (!v || !v.videoWidth) return
      const canvas = document.createElement('canvas')
      canvas.width = v.videoWidth
      canvas.height = v.videoHeight
      canvas.getContext('2d').drawImage(v, 0, 0, canvas.width, canvas.height)
      const link = document.createElement('a')
      link.download = 'screenshot.png'
      link.href = canvas.toDataURL('image/png')
      link.click()
    },
    doRefresh() {
      // 刷新 = 用当前 URL 重新起播。
      if (this.streamInfo) this.play()
    },
    toggleFullscreen() {
      const p = this.activeEngine()
      if (!p) return
      if (this.activePlayer === 'webRTC') {
        const wrap = p.$refs && (p.$refs.wrapper || p.$refs.video)
        if (!document.fullscreenElement) {
          wrap && wrap.requestFullscreen && wrap.requestFullscreen()
        } else {
          document.exitFullscreen && document.exitFullscreen()
        }
      } else if (p.fullscreenSwich) {
        p.fullscreenSwich()
      }
    },
    onFullscreenChange() {
      this.docFull = !!document.fullscreenElement
    },
    startCountdown() {
      this.stopCountdown()
      this.countdownNum = 3
      this.countdownVisible = true
      this.countdownTimer = setInterval(() => {
        this.countdownNum -= 1
        if (this.countdownNum <= 0) {
          this.stopCountdown()
        }
      }, 500)
    },
    stopCountdown() {
      if (this.countdownTimer) {
        clearInterval(this.countdownTimer)
        this.countdownTimer = null
      }
      this.countdownVisible = false
    },
    getActivePlayer() {
      return this.activePlayer
    },
    switchPlayer(key) {
      if (this.activePlayer === key) return
      this.activePlayer = key
      if (this.streamInfo) {
        this.play()
      }
    },
    resolveVideoCodec(streamInfo) {
      if (!streamInfo) return ''
      const direct = streamInfo.videoCodec || ''
      if (direct) return String(direct).toUpperCase()
      const nested = streamInfo.mediaInfo && streamInfo.mediaInfo.videoCodec
      return nested ? String(nested).toUpperCase() : ''
    },
    resolveAudioCodec(streamInfo) {
      if (!streamInfo) return ''
      const direct = streamInfo.audioCodec || ''
      if (direct) return String(direct).toUpperCase()
      const nested = streamInfo.mediaInfo && streamInfo.mediaInfo.audioCodec
      return nested ? String(nested).toUpperCase() : ''
    },
    isH265Codec(codec) {
      return normalizeVideoCodec(codec) === 'H265'
    },
    /** 依据策略决定当前播放器：优先策略首选，其次默认第一个可用 */
    ensureActivePlayer() {
      const pref = this.preferredPlayer && this.allowedKeys.includes(this.preferredPlayer)
        ? this.preferredPlayer
        : (this.allowedKeys.includes(this.strategy.preferredPlayer)
          ? this.strategy.preferredPlayer
          : this.allowedKeys[0])
      this.activePlayer = pref || 'jessibuca'
    },
    /** URL 优先级：props 覆盖 > 策略（RTC 播放器强制只用 rtc） */
    resolveUrlKeyOrder() {
      if (this.activePlayer === 'webRTC') {
        const secure = location.protocol === 'https:'
        return secure ? ['rtcs', 'rtc'].filter(Boolean) : ['rtc']
      }
      if (this.urlPriority && this.urlPriority.length) {
        return this.urlPriority
      }
      return URL_KEYS[this.activePlayer] || URL_KEYS.jessibuca
    },
    getUrlByStreamInfo() {
      if (!this.streamInfo) return ''
      const keys = this.resolveUrlKeyOrder()
      for (let i = 0; i < keys.length; i++) {
        const url = this.streamInfo[keys[i]]
        if (url) return url
      }
      return ''
    },
    invokePlayerPlay(playUrl, retries) {
      const player = this.$refs[this.activePlayer]
      if (player && player.play) {
        player.play(playUrl)
        return
      }
      if (retries > 0) {
        setTimeout(() => this.invokePlayerPlay(playUrl, retries - 1), 80)
      }
    },
    changePlayer(tab) {
      const prev = this.activePlayer
      this.activePlayer = tab.name
      if (prev !== tab.name && this.$refs[prev] && this.$refs[prev].pause) {
        this.$refs[prev].pause()
      }
      this.$nextTick(() => {
        this.play()
        this.syncPlayerSize()
      })
      this.$emit('player-changed', this.activePlayer)
    },
    syncPlayerSize() {
      const player = this.$refs[this.activePlayer]
      if (!player) return
      if (player.updatePlayerDomSize) {
        player.updatePlayerDomSize()
      } else if (player.resize) {
        player.resize()
      }
    },
    setStreamInfo(streamInfo) {
      this.streamInfo = streamInfo
      this.playerPlaying = false
      this.ensureActivePlayer()
      if (this.showControlBar) {
        this.startCountdown()
      }
      // 分屏首次从「无信号」切到播放器时，等 DOM/布局完成再播，避免 Jessibuca 黑屏
      this.$nextTick(() => {
        requestAnimationFrame(() => {
          this.play()
          this.syncPlayerSize()
          setTimeout(() => this.syncPlayerSize(), 200)
        })
      })
    },
    /**
     * 编码信息到达（如 media_info 轮询）：合并进 streamInfo，并做“纠正式”切换——
     * 仅当当前播放器已不在可用列表（例如发现是 H265 而正在用 Jessibuca）才自动切换，避免打扰。
     * @param {{videoCodec?:string, audioCodec?:string}} info
     */
    updateCodec(info) {
      if (!info) return
      const v = info.videoCodec || ''
      const a = info.audioCodec || ''
      if (!v && !a) return
      this.streamInfo = Object.assign({}, this.streamInfo || {}, {
        videoCodec: v || (this.streamInfo && this.streamInfo.videoCodec),
        audioCodec: a || (this.streamInfo && this.streamInfo.audioCodec)
      })
      if (!this.allowedKeys.includes(this.activePlayer)) {
        const next = this.allowedKeys.includes(this.strategy.preferredPlayer)
          ? this.strategy.preferredPlayer
          : this.allowedKeys[0]
        if (next && next !== this.activePlayer) {
          const prev = this.activePlayer
          this.activePlayer = next
          if (this.$refs[prev] && this.$refs[prev].pause) this.$refs[prev].pause()
          this.$nextTick(() => {
            this.play()
            this.syncPlayerSize()
          })
          this.$emit('player-changed', this.activePlayer)
        }
      }
    },
    play() {
      const playUrl = this.getUrlByStreamInfo()
      if (!playUrl) {
        console.warn('[PlayerTabs] empty play url, activePlayer=', this.activePlayer)
        return
      }
      setTimeout(() => {
        this.invokePlayerPlay(playUrl, 12)
        this.syncPlayerSize()
      }, 80)
      const typeMap = { jessibuca: 0, webRTC: 1, h265web: 2 }
      const type = typeMap[this.activePlayer] || 0
      const playerUrl = window.location.origin + '/#/play/share?type=' + type + '&url=' + encodeURIComponent(playUrl)
      this.$emit('playerChanged', { playUrl, playerUrl })
    },
    stop() {
      if (this.$refs[this.activePlayer]) {
        this.$refs[this.activePlayer].pause()
      }
    },
    pause() {
      if (this.$refs[this.activePlayer]) {
        this.$refs[this.activePlayer].pause()
      }
    },
    destroy() {
      const player = this.$refs[this.activePlayer]
      if (player && player.destroy) {
        player.destroy()
      }
    },
    setPlaybackRate(rate) {
      const player = this.$refs[this.activePlayer]
      if (player && player.setPlaybackRate) {
        player.setPlaybackRate(rate)
      }
    },
    resize(width, height) {
      const player = this.$refs[this.activePlayer]
      if (player && player.resize) {
        player.resize(width, height)
      }
    },
    screenshot() {
      const player = this.$refs[this.activePlayer]
      if (player && player.screenshot) {
        return player.screenshot()
      }
    },
    getVideoRect() {
      const player = this.$refs[this.activePlayer]
      return player && player.getVideoRect ? player.getVideoRect() : null
    },
    startDragZoom(callback) {
      const player = this.$refs[this.activePlayer]
      if (player && player.startDragZoom) {
        player.startDragZoom(callback)
      }
    }
  }
}
</script>

<style scoped>
.player-tabs-wrapper {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: #3a4556;
}
.player-tabs-wrapper .el-tabs {
  margin-bottom: 0;
  flex-shrink: 0;
}
.player-tabs-wrapper .el-tabs >>> .el-tabs__header {
  margin-bottom: 0;
}
.player-video-area {
  position: relative;
  flex: 1;
  min-height: 0;
  width: 100%;
  background: #3a4556;
  overflow: hidden;
}
.player-instance {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
}

/* 底部播放控制栏 */
.player-control-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  flex-shrink: 0;
  background: #2b333f;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
}
.pcb-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 30px;
  padding: 0;
  border: none;
  outline: none;
  background: transparent;
  color: #cfd6df;
  cursor: pointer;
  border-radius: 6px;
  font-size: 16px;
  transition: background 0.15s, color 0.15s;
}
.pcb-btn:hover {
  background: rgba(255, 255, 255, 0.12);
  color: #fff;
}
.pcb-spacer {
  flex: 1;
}

/* 起播倒计时遮罩 */
.player-countdown {
  position: absolute;
  inset: 0;
  z-index: 5;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: rgba(15, 23, 42, 0.42);
  pointer-events: none;
}
.player-countdown-num {
  font-size: 40px;
  font-weight: 800;
  line-height: 1;
  color: #fff;
  text-shadow: 0 2px 12px rgba(0, 0, 0, 0.5);
}
.cd-fade-enter-active,
.cd-fade-leave-active {
  transition: opacity 0.2s;
}
.cd-fade-enter,
.cd-fade-leave-to {
  opacity: 0;
}
</style>
