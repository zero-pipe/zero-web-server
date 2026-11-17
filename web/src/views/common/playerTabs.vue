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
        :show-button="showButton"
        fluent autoplay live
        @playTimeChange="$emit('playTimeChange', $event)"
        @playStatusChange="$emit('playStatusChange', $event)"
      />
      <rtc-player
        v-if="activePlayer === 'webRTC'"
        ref="webRTC"
        class="player-instance"
        :has-audio="hasAudio"
        :show-button="showButton"
        fluent autoplay live
        @playTimeChange="$emit('playTimeChange', $event)"
        @playStatusChange="$emit('playStatusChange', $event)"
      />
      <h265web
        v-if="activePlayer === 'h265web'"
        ref="h265web"
        class="player-instance"
        :has-audio="hasAudio"
        :show-button="showButton"
        fluent autoplay live
        @playTimeChange="$emit('playTimeChange', $event)"
        @playStatusChange="$emit('playStatusChange', $event)"
      />
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
    preferredPlayer: { type: String, default: '' }
  },
  data() {
    return {
      streamInfo: null,
      activePlayer: 'jessibuca'
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
  methods: {
    getPlayerList() {
      return this.playerList
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
</style>
