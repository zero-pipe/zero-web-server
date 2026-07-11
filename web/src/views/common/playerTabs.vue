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

export default {
  name: 'PlayerTabs',
  components: { jessibucaPlayer, rtcPlayer, h265web },
  props: {
    hasAudio: { type: Boolean, default: false },
    showButton: { type: Boolean, default: true },
    showTab: { type: Boolean, default: true },
    /** 限定可用播放器，如 ['jessibuca'] 或 ['h265web'] */
    allowedPlayers: { type: Array, default: null },
    /** 覆盖默认 URL 优先级，如 ['flv','ws_flv'] */
    urlPriority: { type: Array, default: null },
    /** 首选播放器，如 jessibuca / h265web / webRTC */
    preferredPlayer: { type: String, default: '' }
  },
  data() {
    return {
      streamInfo: null,
      activePlayer: 'jessibuca',
      player: {
        jessibuca: ['flv', 'https_flv', 'ws_flv', 'wss_flv'],
        webRTC: ['rtc', 'rtcs'],
        h265web: ['flv', 'https_flv', 'ws_flv', 'wss_flv']
      },
      allPlayerList: [
        { key: 'jessibuca', label: 'Jessibuca' },
        { key: 'webRTC', label: 'WebRTC' },
        { key: 'h265web', label: 'H265web' }
      ]
    }
  },
  computed: {
    playerList() {
      if (this.allowedPlayers && this.allowedPlayers.length) {
        return this.allPlayerList.filter(p => this.allowedPlayers.includes(p.key))
      }
      return this.allPlayerList
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
    getUrlByStreamInfo() {
      if (!this.streamInfo) return ''
      if (this.urlPriority && this.urlPriority.length) {
        for (let i = 0; i < this.urlPriority.length; i++) {
          const url = this.streamInfo[this.urlPriority[i]]
          if (url) return url
        }
      }
      const keys = this.player[this.activePlayer]
      if (!keys || !keys.length) return ''
      const secure = location.protocol === 'https:'
      const ordered = secure
        ? [keys[1], keys[0], keys[3], keys[2]].filter(Boolean)
        : keys
      for (let i = 0; i < ordered.length; i++) {
        const url = this.streamInfo[ordered[i]]
        if (url) return url
      }
      return ''
    },
    resolveVideoCodec(streamInfo) {
      if (!streamInfo) return ''
      const direct = streamInfo.videoCodec || ''
      if (direct) return String(direct).toUpperCase()
      const nested = streamInfo.mediaInfo && streamInfo.mediaInfo.videoCodec
      return nested ? String(nested).toUpperCase() : ''
    },
    isH265Codec(codec) {
      return codec === 'H265' || codec === 'HEVC' || codec.indexOf('265') >= 0
    },
    warnIfPlayerMismatch(playerKey) {
      const codec = this.resolveVideoCodec(this.streamInfo)
      if (playerKey === 'jessibuca' && this.isH265Codec(codec)) {
        this.$message.warning('当前为 H265 编码，Jessibuca 无法播放，请使用 H265web')
        return true
      }
      return false
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
    applyPreferredPlayer() {
      if (this.preferredPlayer && this.playerList.some(p => p.key === this.preferredPlayer)) {
        this.activePlayer = this.preferredPlayer
        return
      }
      this.selectPlayerForStream(this.streamInfo)
    },
    selectPlayerForStream(streamInfo) {
      const codec = this.resolveVideoCodec(streamInfo)
      if (this.isH265Codec(codec)) {
        this.activePlayer = 'h265web'
        return
      }
      if (codec === 'H264' || codec === 'AVC') {
        this.activePlayer = 'jessibuca'
      }
    },
    changePlayer(tab) {
      const prev = this.activePlayer
      this.activePlayer = tab.name
      if (prev !== tab.name && this.$refs[prev] && this.$refs[prev].pause) {
        this.$refs[prev].pause()
      }
      this.warnIfPlayerMismatch(tab.name)
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
      this.applyPreferredPlayer()
      // 分屏首次从「无信号」切到播放器时，等 DOM/布局完成再播，避免 Jessibuca 黑屏
      this.$nextTick(() => {
        requestAnimationFrame(() => {
          this.play()
          this.syncPlayerSize()
          setTimeout(() => this.syncPlayerSize(), 200)
        })
      })
    },
    play() {
      const playUrl = this.getUrlByStreamInfo()
      if (!playUrl) {
        console.warn('[PlayerTabs] empty play url, activePlayer=', this.activePlayer)
        if (this.activePlayer === 'webRTC') {
          this.$message.error('WebRTC 播放地址为空，请确认点播接口返回 rtc 字段')
        }
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
