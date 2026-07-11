<template>
    <div
      ref="container"
      class="jessibuca-container"
      style="width:100%; height: 100%; background-color: #000000;margin:0 auto;position: relative;"
      @dblclick="fullscreenSwich"
      @mouseenter="showBar = true" @mouseleave="showBar = false"
      @click="onUserGestureUnlockAudio"
    >
      <div id="buttonsBox" class="buttons-box" v-if="showButton === undefined || showButton" :style="{ opacity: showBar ? 1 : 0, pointerEvents: showBar ? 'auto' : 'none' }">
        <div class="buttons-box-left">
          <i v-if="!playing" class="iconfont icon-play jessibuca-btn" @click.stop="playBtnClick" />
          <i v-if="playing" class="iconfont icon-pause jessibuca-btn" @click.stop="pause" />
          <i class="iconfont icon-stop jessibuca-btn" @click.stop="stop" />
          <i v-if="isNotMute" class="iconfont icon-audio-high jessibuca-btn" @click.stop="mute()" />
          <i v-if="!isNotMute" class="iconfont icon-audio-mute jessibuca-btn" title="点击开启声音" @click.stop="cancelMute()" />
        </div>
        <div class="buttons-box-right">
          <span class="jessibuca-btn">{{ kBps }} kb/s</span>
          <i
            class="iconfont icon-camera1196054easyiconnet jessibuca-btn"
            style="font-size: 1rem !important"
            @click.stop="screenshot"
          />
          <i class="iconfont icon-shuaxin11 jessibuca-btn" @click.stop="playBtnClick" />
          <i v-if="!fullscreen" class="iconfont icon-weibiaoti10 jessibuca-btn" @click.stop="fullscreenSwich" />
          <i v-if="fullscreen" class="iconfont icon-weibiaoti11 jessibuca-btn" @click.stop="fullscreenSwich" />
        </div>
      </div>
      <div v-if="playing && !isNotMute" class="jessibuca-unmute-tip" @click.stop="cancelMute()">
        点击开启声音
      </div>
    </div>
</template>

<script>
const jessibucaPlayer = {}
import dragZoom from '../../mixins/dragZoom'
export default {
  name: 'Jessibuca',
  mixins: [dragZoom],
  props: ['videoUrl', 'error', 'hasAudio', 'height', 'showButton'],
  data() {
    return {
      playing: false,
      isNotMute: true,
      quieting: false,
      fullscreen: false,
      loaded: false, // mute
      speed: 0,
      performance: '', // 工作情况
      kBps: 0,
      btnDom: null,
      videoInfo: null,
      volume: 1,
      playerTime: 0,
      rotate: 0,
      vod: true, // 点播
      forceNoOffscreen: false,
      localVideoUrl: this.videoUrl,
      showBar: true
    }
  },
  created() {
    this.btnDom = document.getElementById('buttonsBox')
  },
  mounted() {},
  destroyed() {
    if (jessibucaPlayer[this._uid]) {
      jessibucaPlayer[this._uid].videoPTS = 0
      jessibucaPlayer[this._uid].destroy()
    }
    this.playing = false
    this.loaded = false
    this.performance = ''
    this.playerTime = 0
  },
  methods: {
    isLiveFlvUrl(url) {
      return !!url && /\.flv(\?|$|#)/i.test(url)
    },
    isWsFlvUrl(url) {
      return !!url && /^wss?:\/\//i.test(url) && this.isLiveFlvUrl(url)
    },
    create(url) {
      if (jessibucaPlayer[this._uid]) {
        jessibucaPlayer[this._uid].destroy()
      }
      if (this.$refs.container.dataset['jessibuca']) {
        this.$refs.container.dataset['jessibuca'] = undefined
      }

      if (this.$refs.container.getAttribute('data-jessibuca')) {
        this.$refs.container.removeAttribute('data-jessibuca')
      }
      const isFlv = this.isLiveFlvUrl(url)
      const isWsFlv = this.isWsFlvUrl(url)
      // 与 H265web ignoreAudio=0 对齐：播放器始终解 FLV 音轨。
      // 通道 hasAudio 只影响国标是否带音频邀请，不能用来关掉已推流 AAC。
      const options = {
        container: this.$refs.container,
        videoBuffer: isFlv ? 0.2 : 0,
        isResize: true,
        // WS-FLV 走 wasm 解码；HTTP-FLV 可用 MSE（AAC 走浏览器解码）
        useMSE: isFlv && !isWsFlv,
        useWCS: false,
        text: '',
        controlAutoHide: false,
        debug: false,
        hotKey: true,
        decoder: '/static/js/jessibuca/decoder.js',
        timeout: 15,
        recordType: 'mp4',
        isFlv: isFlv,
        vod: !isFlv,
        forceNoOffscreen: true,
        hasAudio: true,
        heartTimeout: 10,
        heartTimeoutReplay: true,
        heartTimeoutReplayTimes: 2,
        hiddenAutoPause: false,
        isFullResize: false,
        isNotMute: true,
        keepScreenOn: true,
        loadingText: '请稍等, 视频加载中......',
        loadingTimeout: 15,
        loadingTimeoutReplay: true,
        loadingTimeoutReplayTimes: 2,
        openWebglAlignment: false,
        operateBtns: {
          fullscreen: false,
          screenshot: false,
          play: false,
          audio: false,
          recorder: false
        },
        showBandwidth: false,
        supportDblclickFullscreen: false,
        useWebFullSreen: true,
        wasmDecodeErrorReplay: true,
        wcsUseVideoRendcer: true
      }
      jessibucaPlayer[this._uid] = new window.Jessibuca(options)

      const jessibuca = jessibucaPlayer[this._uid]
      jessibuca.on('pause', () => {
        this.playing = false
        this.$emit('playStatusChange', false)
      })
      jessibuca.on('play', () => {
        this.playing = true
        this.loaded = true
        this.quieting = jessibuca.quieting
        this.$emit('playStatusChange', true)
        // 浏览器自动播放策略常强制静音；对齐 H265web setVoice(1.0)
        this.ensureAudioOn()
      })
      jessibuca.on('fullscreen', (msg) => {
        this.fullscreen = msg
      })
      jessibuca.on('mute', (msg) => {
        this.isNotMute = !msg
      })
      jessibuca.on('performance', (performance) => {
        let show = '卡顿'
        if (performance === 2) {
          show = '非常流畅'
        } else if (performance === 1) {
          show = '流畅'
        }
        this.performance = show
      })
      jessibuca.on('kBps', (kBps) => {
        this.kBps = Math.round(kBps)
      })
      jessibuca.on('videoInfo', () => {})
      jessibuca.on('audioInfo', () => {})
      jessibuca.on('error', (msg) => {
        console.warn('Jessibuca error:', msg)
      })
      jessibuca.on('timeout', () => {})
      jessibuca.on('loadingTimeout', () => {})
      jessibuca.on('delayTimeout', () => {})
      jessibuca.on('playToRenderTimes', () => {})
      jessibuca.on('timeUpdate', (videoPTS) => {
        if (jessibuca.videoPTS) {
          this.playerTime += (videoPTS - jessibuca.videoPTS)
          this.$emit('playTimeChange', this.playerTime)
        }
        jessibuca.videoPTS = videoPTS
      })
    },
    ensureAudioOn() {
      const player = jessibucaPlayer[this._uid]
      if (!player) {
        return
      }
      try {
        // Jessibuca 文档：Chrome 自动播放必须静音，需真实用户手势调用 audioResume
        if (typeof player.audioResume === 'function') {
          player.audioResume()
        }
        if (typeof player.cancelMute === 'function') {
          player.cancelMute()
        }
        if (typeof player.setVolume === 'function') {
          player.setVolume(1)
        }
        if (typeof player.isMute === 'function') {
          this.isNotMute = !player.isMute()
        } else {
          this.isNotMute = true
        }
      } catch (e) {
        console.warn('Jessibuca unmute failed:', e)
      }
    },
    onUserGestureUnlockAudio() {
      this.ensureAudioOn()
    },
    playBtnClick: function() {
      this.ensureAudioOn()
      this.play(this.videoUrl)
    },
    play: function(url) {
      if (!url) {
        console.warn('Jessibuca -> invalid url, skip play')
        return
      }
      if (this.playing) {
        this.stop()
      }
      this.localVideoUrl = url
      const isFlv = this.isLiveFlvUrl(url)
      const existing = jessibucaPlayer[this._uid]
      if (existing && existing._zmsIsFlv !== isFlv) {
        existing.destroy()
        jessibucaPlayer[this._uid] = null
      }
      if (!jessibucaPlayer[this._uid]) {
        this.create(url)
        jessibucaPlayer[this._uid]._zmsIsFlv = isFlv
      }
      const start = () => {
        const p = jessibucaPlayer[this._uid]
        const ret = p.play(url)
        if (ret && typeof ret.then === 'function') {
          ret.then(() => this.ensureAudioOn()).catch(() => this.ensureAudioOn())
        } else {
          this.$nextTick(() => this.ensureAudioOn())
        }
      }
      if (jessibucaPlayer[this._uid].hasLoaded()) {
        start()
      } else {
        jessibucaPlayer[this._uid].on('load', start)
      }
    },
    pause: function() {
      if (jessibucaPlayer[this._uid]) {
        jessibucaPlayer[this._uid].pause()
      }
      this.playing = false
      this.err = ''
      this.performance = ''
    },
    stop: function() {
      if (jessibucaPlayer[this._uid]) {
        jessibucaPlayer[this._uid].pause()
        jessibucaPlayer[this._uid].clearView()
      }
      this.playing = false
      this.err = ''
      this.performance = ''
    },
    screenshot: function() {
      if (jessibucaPlayer[this._uid]) {
        jessibucaPlayer[this._uid].screenshot()
      }
    },
    mute: function() {
      if (jessibucaPlayer[this._uid]) {
        jessibucaPlayer[this._uid].mute()
      }
      this.isNotMute = false
    },
    cancelMute: function() {
      this.ensureAudioOn()
    },
    destroy: function() {
      if (jessibucaPlayer[this._uid]) {
        jessibucaPlayer[this._uid].destroy()
      }
      // if (document.getElementById('buttonsBox') === null && (typeof this.showButton === 'undefined' || this.showButton)) {
      //   this.$refs.container.appendChild(this.btnDom)
      // }
      jessibucaPlayer[this._uid] = null
      this.playing = false
      this.err = ''
      this.performance = ''
    },
    fullscreenSwich: function() {
      const isFull = this.isFullscreen()
      jessibucaPlayer[this._uid].setFullscreen(!isFull)
      this.fullscreen = !isFull
    },
    isFullscreen: function() {
      return document.fullscreenElement ||
        document.msFullscreenElement ||
        document.mozFullScreenElement ||
        document.webkitFullscreenElement || false
    },
    setPlaybackRate: function() {

    },
    resize(width, height) {
      if (jessibucaPlayer[this._uid]) {
        jessibucaPlayer[this._uid].resize()
      }
    },
    getVideoElement() {
      return this.$refs.container.querySelector('canvas')
    },
    getVideoRect() {
      const container = this.$refs.container
      const canvas = this.getVideoElement()
      return canvas ? canvas.getBoundingClientRect() : container.getBoundingClientRect()
    }
  }
}
</script>

<style>
.jessibuca-container {
  position: relative;
}
.jessibuca-unmute-tip {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  z-index: 12;
  padding: 8px 14px;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.65);
  color: #fff;
  font-size: 13px;
  cursor: pointer;
  user-select: none;
  border: 1px solid rgba(255, 255, 255, 0.25);
}
.jessibuca-unmute-tip:hover {
  background: rgba(61, 139, 253, 0.85);
}
.buttons-box {
  width: 100%;
  height: 28px;
  background-color: rgba(43, 51, 63, 0.7);
  position: absolute;
  transition: opacity 0.3s ease;
  display: flex;
  left: 0;
  bottom: 0;
  user-select: none;
  z-index: 10;
}

.jessibuca-btn {
  width: 20px;
  color: rgb(255, 255, 255);
  line-height: 27px;
  margin: 0px 20px;
  padding: 0px 2px;
  cursor: pointer;
  text-align: center;
  font-size: 0.8rem !important;
}

.buttons-box-right {
  position: absolute;
  right: 0;
}
</style>
