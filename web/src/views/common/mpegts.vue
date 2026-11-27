<template>
  <div
    ref="container"
    class="mpegts-container"
    style="width:100%; height:100%; background-color:#000; position:relative;"
    @dblclick="fullscreenSwich"
  >
    <video
      ref="video"
      class="mpegts-video"
      controls
      muted
      playsinline
    />
    <div class="mpegts-info">
      <span class="mpegts-bps">{{ kBps }} kb/s</span>
      <span v-if="playError" class="mpegts-err">{{ playError }}</span>
    </div>
  </div>
</template>

<script>
const mpegtsPlayer = {}

export default {
  name: 'MpegtsPlayer',
  props: ['videoUrl', 'error', 'hasAudio', 'height', 'showButton'],
  data() {
    return {
      playing: false,
      kBps: 0,
      playError: '',
      fullscreen: false
    }
  },
  beforeDestroy() {
    this.destroy()
  },
  destroyed() {
    this.destroy()
  },
  methods: {
    create(url) {
      const video = this.$refs.video
      if (!video) return
      if (!window.mpegts) {
        console.warn('[Mpegts] mpegts.js 未加载')
        this.playError = '播放器库未加载'
        return
      }
      if (!window.mpegts.getFeatureList().mseLivePlayback) {
        console.warn('[Mpegts] 浏览器不支持 MSE Live')
        this.playError = '浏览器不支持 MSE'
        return
      }
      this.destroy()

      const player = window.mpegts.createPlayer(
        { type: 'flv', isLive: true, url: url },
        {
          enableWorker: true,
          enableStashBuffer: false,
          liveBufferLatencyChasing: true,
          lazyLoad: false,
          autoCleanupSourceBuffer: true
        }
      )
      mpegtsPlayer[this._uid] = player
      player.attachMediaElement(video)

      player.on(window.mpegts.Events.ERROR, (type, detail, info) => {
        console.warn('[Mpegts] error', type, detail, info)
        this.playing = false
        this.playError = '播放错误(' + detail + (info && info.msg ? ':' + info.msg : '') + ')'
        this.$emit('playStatusChange', false)
      })
      player.on(window.mpegts.Events.MEDIA_INFO, (mi) => {
        // 拿到媒体信息即认为已出画面
        this.playing = true
        this.playError = ''
        this.$emit('playStatusChange', true)
      })
      player.on(window.mpegts.Events.STATISTICS_INFO, (si) => {
        this.kBps = si.speed ? Math.round(si.speed) : 0 // KB/s
      })
      player.load()
      const p = video.play && video.play()
      if (p && p.catch) p.catch(() => {})
      video.addEventListener('playing', () => {
        this.playing = true
        this.playError = ''
        this.$emit('playStatusChange', true)
      })
    },
    play(url) {
      if (!url) {
        console.warn('[Mpegts] empty url')
        return
      }
      // 容器尺寸为 0 时等布局完成再初始化
      const c = this.$refs.container
      if (c && (c.clientWidth < 2 || c.clientHeight < 2)) {
        this._sizeWaitTries = (this._sizeWaitTries || 0) + 1
        if (this._sizeWaitTries > 40) {
          this._sizeWaitTries = 0
        } else {
          clearTimeout(this._sizeWaitTimer)
          this._sizeWaitTimer = setTimeout(() => this.play(url), 50)
          return
        }
      }
      this._sizeWaitTries = 0
      this.localVideoUrl = url
      this.create(url)
    },
    pause() {
      const p = mpegtsPlayer[this._uid]
      if (p) { try { p.pause() } catch (e) {} }
      this.playing = false
    },
    stop() {
      const p = mpegtsPlayer[this._uid]
      if (p) { try { p.pause() } catch (e) {} }
      this.playing = false
    },
    destroy() {
      clearTimeout(this._sizeWaitTimer)
      const p = mpegtsPlayer[this._uid]
      if (p) {
        try { p.pause(); p.unload(); p.detachMediaElement(); p.destroy() } catch (e) {}
      }
      mpegtsPlayer[this._uid] = null
      this.playing = false
    },
    screenshot() {
      const v = this.$refs.video
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
    mute() {},
    cancelMute() {},
    fullscreenSwich() {
      const el = this.$refs.container
      if (!document.fullscreenElement) {
        el.requestFullscreen && el.requestFullscreen()
        this.fullscreen = true
      } else {
        document.exitFullscreen && document.exitFullscreen()
        this.fullscreen = false
      }
    },
    resize() {},
    getVideoElement() {
      return this.$refs.video
    },
    getVideoRect() {
      const v = this.$refs.video
      return v ? v.getBoundingClientRect() : (this.$refs.container ? this.$refs.container.getBoundingClientRect() : null)
    }
  }
}
</script>

<style scoped>
.mpegts-container {
  position: relative;
}
.mpegts-video {
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: #000;
}
.mpegts-info {
  position: absolute;
  left: 8px;
  bottom: 34px;
  font-size: 12px;
  color: #cfd6df;
  pointer-events: none;
}
.mpegts-bps {
  background: rgba(0, 0, 0, 0.5);
  padding: 1px 6px;
  border-radius: 4px;
}
.mpegts-err {
  display: inline-block;
  margin-left: 6px;
  color: #f87171;
  background: rgba(0, 0, 0, 0.5);
  padding: 1px 6px;
  border-radius: 4px;
}
</style>
