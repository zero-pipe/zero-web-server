<template>
  <div class="player-tabs-wrapper" ref="playerWrapper">
    <div class="player-video-area">
      <jessibuca-player
        ref="jessibuca"
        class="player-instance"
        :has-audio="false"
        :show-button="showButton && !showControlBar"
        @playTimeChange="$emit('playTimeChange', $event)"
        @playStatusChange="onPlayStatusChange"
      />
    </div>
    <!-- 播放控制栏：固定在画面下方，不再是悬浮在画面里的图标。 -->
    <div v-if="showControlBar" class="player-control-bar">
      <button
        class="pcb-btn"
        :title="playerPlaying ? '暂停' : '播放'"
        @click="togglePlay"
      >
        <i :class="playerPlaying ? 'el-icon-video-pause' : 'el-icon-video-play'" />
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
import { PLAYER_LABELS } from '@/utils/playerStrategy'

const URL_KEYS = {
  jessibuca: ['flv', 'https_flv', 'ws_flv', 'wss_flv']
}

export default {
  name: 'PlayerTabs',
  components: { jessibucaPlayer },
  props: {
    hasAudio: { type: Boolean, default: false },
    showButton: { type: Boolean, default: true },
    showTab: { type: Boolean, default: true },
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
    allowedKeys() {
      return ['jessibuca']
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
    activePlayer() {
      this.playerPlaying = false
      this.$nextTick(() => this.syncMuteState())
    }
  },
  methods: {
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
      if (typeof p.isNotMute === 'boolean') {
        this.ctrlMuted = p.isNotMute === false
      }
    },
    togglePlay() {
      const p = this.activeEngine()
      if (!p) return
      if (this.playerPlaying) {
        p.pause && p.pause()
      } else {
        if (p.playBtnClick) {
          p.playBtnClick()
        } else if (this.getUrlByStreamInfo()) {
          this.play()
        }
      }
    },
    toggleMute() {
      const p = this.activeEngine()
      if (!p) return
      if (typeof p.isNotMute === 'boolean') {
        if (p.isNotMute) { p.mute && p.mute() } else { p.cancelMute && p.cancelMute() }
        this.ctrlMuted = !p.isNotMute
      }
    },
    doScreenshot() {
      const p = this.activeEngine()
      if (!p) return
      if (p.screenshot) {
        p.screenshot()
      }
    },
    doRefresh() {
      // 刷新 = 用当前 URL 重新起播。
      if (this.streamInfo) this.play()
    },
    toggleFullscreen() {
      const p = this.activeEngine()
      if (!p) return
      if (p.fullscreenSwich) {
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
    /** 单引擎：恒为 jessibuca */
    ensureActivePlayer() {
      this.activePlayer = 'jessibuca'
    },
    /** URL 优先级：固定 FLV 系列 */
    resolveUrlKeyOrder() {
      return URL_KEYS.jessibuca
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
     * 编码信息到达（如 media_info 轮询）：合并进 streamInfo。单引擎无需切换。
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
      const playerUrl = window.location.origin + '/#/play/share?type=0&url=' + encodeURIComponent(playUrl)
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

/* 起播倒计时遮罩：盖住引擎自带的 loading 圈/文字，只留倒计时数字 */
.player-countdown {
  position: absolute;
  inset: 0;
  z-index: 30;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #1b2430;
  pointer-events: none;
}
/* 倒计时期间隐藏引擎自带的 loading 圈与文字（穿透子组件），避免与数字叠加 */
.player-video-area.is-counting >>> .jessibuca-loading,
.player-video-area.is-counting >>> .jessibuca-loading-text,
.player-video-area.is-counting >>> .play-loading {
  display: none !important;
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
