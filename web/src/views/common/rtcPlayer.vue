<template>
  <div :id="'rtcPlayer-' + _uid" ref="wrapper" class="rtc-player-wrapper">
    <video
      :id="'webRtcPlayerBox-' + _uid"
      ref="video"
      class="rtc-player-video"
      :controls="showControls"
      autoplay
      muted
      playsinline
      webkit-playsinline
    />
  </div>
</template>

<script>
const webrtcPlayer = {}
import dragZoom from '../../mixins/dragZoom'
export default {
  name: 'RtcPlayer',
  mixins: [dragZoom],
  props: {
    videoUrl: { type: String, default: '' },
    error: { default: '' },
    hasaudio: { type: Boolean, default: false },
    showControls: { type: Boolean, default: true }
  },
  data() {
    return {
      timer: null
    }
  },
  mounted() {},
  destroyed() {
    clearTimeout(this.timer)
    this.pause()
  },
  methods: {
    play: function(url, attempt) {
      if (typeof attempt !== 'number') {
        attempt = 0
      }
      if (!url) {
        console.warn('[RtcPlayer] empty url, skip play')
        return
      }
      clearTimeout(this.timer)
      if (webrtcPlayer[this._uid]) {
        this.pause()
      }
      const el = this.$refs.video || document.getElementById('webRtcPlayerBox-' + this._uid)
      if (!el) {
        if (attempt < 12) {
          this.timer = setTimeout(() => this.play(url, attempt + 1), 80)
        } else {
          console.warn('[RtcPlayer] video element missing')
        }
        return
      }
      // 必须用平台下发的绝对地址（streamIp，不能是 127.0.0.1），SDP/ICE 才能对上媒体口
      console.log('[RtcPlayer] play', url, 'attempt=', attempt)
      webrtcPlayer[this._uid] = new ZLMRTCClient.Endpoint({
        element: el,
        debug: true,
        zlmsdpUrl: url,
        simulcast: false,
        useCamera: false,
        audioEnable: true,
        videoEnable: true,
        recvOnly: true,
        usedatachannel: false
      })
      const player = webrtcPlayer[this._uid]
      player.on(ZLMRTCClient.Events.WEBRTC_ICE_CANDIDATE_ERROR, (e) => {
        console.error('ICE 协商出错', e)
        this.eventcallbacK('ICE ERROR', 'ICE 协商出错')
      })

      player.on(ZLMRTCClient.Events.WEBRTC_ON_REMOTE_STREAMS, (e) => {
        console.log('播放成功', e.streams)
        this.ensureVideoPlaying()
        this.eventcallbacK('playing', '播放成功')
        this.$emit('playStatusChange', true)
      })

      player.on(ZLMRTCClient.Events.WEBRTC_OFFER_ANWSER_EXCHANGE_FAILED, (e) => {
        console.error('offer anwser 交换失败', e)
        this.eventcallbacK('OFFER ANSWER ERROR ', 'offer anwser 交换失败')
        if (attempt < 10) {
          this.timer = setTimeout(() => {
            this.play(url, attempt + 1)
          }, attempt < 3 ? 300 : 800)
        }
      })

      player.on(ZLMRTCClient.Events.WEBRTC_ON_LOCAL_STREAM, (s) => {
        this.eventcallbacK('LOCAL STREAM', '获取到了本地流')
      })
    },
    ensureVideoPlaying() {
      const el = this.$refs.video || document.getElementById('webRtcPlayerBox-' + this._uid)
      if (!el) return
      el.muted = true
      const p = el.play && el.play()
      if (p && typeof p.catch === 'function') {
        p.catch(err => console.warn('[RtcPlayer] video.play()', err))
      }
    },
    pause: function() {
      if (webrtcPlayer[this._uid]) {
        webrtcPlayer[this._uid].close()
        webrtcPlayer[this._uid] = null
      }
      const el = this.$refs.video || document.getElementById('webRtcPlayerBox-' + this._uid)
      if (el) {
        try {
          el.srcObject = null
        } catch (e) { /* ignore */ }
      }
    },
    stop: function() {
      this.pause()
    },
    eventcallbacK: function(type, message) {
      console.log('player 事件回调')
      console.log(type)
      console.log(message)
    },
    getVideoElement() {
      return this.$refs.video || document.getElementById('webRtcPlayerBox-' + this._uid)
    },
    getVideoRect() {
      const video = this.getVideoElement()
      if (!video) return null
      const rect = video.getBoundingClientRect()
      if (video.videoWidth && video.videoHeight) {
        const natRatio = video.videoWidth / video.videoHeight
        const disRatio = rect.width / rect.height
        let w, h, x, y
        if (natRatio > disRatio) {
          w = rect.width
          h = w / natRatio
          x = 0
          y = (rect.height - h) / 2
        } else {
          h = rect.height
          w = h * natRatio
          x = (rect.width - w) / 2
          y = 0
        }
        return {
          left: rect.left + x, top: rect.top + y,
          right: rect.left + x + w, bottom: rect.top + y + h,
          width: w, height: h
        }
      }
      return rect
    }
  }
}
</script>

<style>
    .LodingTitle {
        min-width: 70px;
    }
    .rtc-player-wrapper{
        width: 100%;
        height: 100%;
        position: relative;
        background: #0f172a;
    }
    .rtc-player-video{
        width: 100%;
        height: 100%;
        max-height: 100%;
        object-fit: contain;
        background-color: #0f172a;
    }
</style>
