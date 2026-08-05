<template>
  <div id="cloudRecordPlayer" class="cloud-record-player-root">
    <div class="cloud-record-playBox">
      <video
        v-if="useNativeMp4"
        ref="nativeVideo"
        class="cloud-record-native-video"
        playsinline
        @timeupdate="onNativeTimeUpdate"
        @play="playing = true"
        @pause="playing = false"
        @ended="playing = false"
        @loadedmetadata="onNativeLoaded"
      />
      <playerTabs
        v-else
        ref="recordVideoPlayer"
        :show-button="false"
        :showTab="false"
        @playTimeChange="showPlayTimeChange"
        @playStatusChange="playingChange"
      />
    </div>
    <div class="cloud-record-player-option-box">
      <div class="cloud-record-show-time">
        {{showPlayTimeValue}}
      </div>
      <div class="cloud-record-time-process" ref="timeProcess" @click="timeProcessClick($event)"
           @mouseenter="timeProcessMouseEnter($event)" @mousemove="timeProcessMouseMove($event)"
           @mouseleave="timeProcessMouseLeave($event)">
        <div v-if="streamInfo">
          <div class="cloud-record-time-process-value" :style="playTimeValue"></div>
          <transition name="el-fade-in-linear">
            <div v-show="showTimeLeft" class="cloud-record-time-process-title" :style="playTimeTitleStyle" >{{showPlayTimeTitle}}</div>
          </transition>
        </div>
      </div>
      <div class="cloud-record-show-time">
        {{showPlayTimeTotal}}
      </div>
    </div>
    <div class="cloud-record-control-bar">
      <div style="text-align: left;">
        <div class="cloud-record-record-play-control" style="background-color: transparent; box-shadow: 0 0 10px transparent">
          <a v-if="showListCallback" target="_blank" class="cloud-record-record-play-control-item iconfont icon-list" title="列表" @click="sidebarControl()" />
          <a target="_blank" class="cloud-record-record-play-control-item iconfont icon-camera1196054easyiconnet" title="截图" @click="snap()" />
        </div>
      </div>
      <div style="text-align: center;">
        <div class="cloud-record-record-play-control">
          <a v-if="!lastDiable" target="_blank" class="cloud-record-record-play-control-item iconfont icon-diyigeshipin" title="上一个" @click="playLast()" />
          <a v-else style="color: #acacac; cursor: not-allowed" target="_blank" class="cloud-record-record-play-control-item iconfont icon-diyigeshipin" title="上一个" />
          <a target="_blank" class="cloud-record-record-play-control-item iconfont icon-kuaijin" title="快退五秒" @click="seekBackward()" />
          <a target="_blank" class="cloud-record-record-play-control-item iconfont icon-stop1" style="font-size: 14px" title="停止" @click="stopPLay()" />
          <a v-if="playing" target="_blank" class="cloud-record-record-play-control-item iconfont icon-zanting" title="暂停" @click="pausePlay()" />
          <a v-if="!playing" target="_blank" class="cloud-record-record-play-control-item iconfont icon-kaishi" title="播放" @click="play()" />
          <a target="_blank" class="cloud-record-record-play-control-item iconfont icon-houtui" title="快进五秒" @click="seekForward()" />
          <a v-if="!nextDiable" target="_blank" class="cloud-record-record-play-control-item iconfont icon-zuihouyigeshipin" title="下一个" @click="playNext()" />
          <a v-else style="color: #acacac; cursor: not-allowed" target="_blank" class="cloud-record-record-play-control-item iconfont icon-zuihouyigeshipin" title="下一个" @click="playNext()" />
          <el-dropdown @command="changePlaySpeed" :popper-append-to-body='false' >
            <a target="_blank" class="cloud-record-record-play-control-item record-play-control-speed" title="倍速播放">{{ playSpeed }}X</a>
            <el-dropdown-menu slot="dropdown">
              <el-dropdown-item
                v-for="item in playSpeedRange"
                :key="item"
                :command="item"
              >{{ item }}X</el-dropdown-item>
            </el-dropdown-menu>
          </el-dropdown>
        </div>
      </div>
      <div style="text-align: right;">
        <div class="cloud-record-record-play-control" style="background-color: transparent; box-shadow: 0 0 10px transparent">
          <a v-if="!hideFullscreen && !isFullScreen" target="_blank" class="cloud-record-record-play-control-item iconfont icon-fangdazhanshi" title="全屏" @click="fullScreen()" />
          <a v-if="!hideFullscreen && isFullScreen" target="_blank" class="cloud-record-record-play-control-item iconfont icon-suoxiao1" title="全屏" @click="fullScreen()" />
        </div>
      </div>
    </div>
  </div>
</template>

<script>

import playerTabs from '../common/playerTabs.vue'
import moment from 'moment'
import momentDurationFormatSetup from 'moment-duration-format'
import screenfull from 'screenfull'

momentDurationFormatSetup(moment)

export default {
  name: 'CloudRecordPlayer',
  components: { playerTabs },
  props: {
    showListCallback: { type: Function, default: null },
    showNextCallback: { type: Function, default: null },
    showLastCallback: { type: Function, default: null },
    lastDiable: { type: Boolean, default: false },
    nextDiable: { type: Boolean, default: false },
    hidePlayerSwitch: { type: Boolean, default: false },
    hideFullscreen: { type: Boolean, default: false }
  },
  data() {
    return {
      showSidebar: false,
      streamInfo: null,
      timeLen: null,
      startTime: null,
      showTimeLeft: null,
      isMousedown: false,
      loading: false,
      playerTime: null,
      playSpeed: 1,
      playLoading: false,
      isFullScreen: false,
      playing: false,
      initTime: null,
      playSpeedRange: [1, 2, 4, 6, 8, 16, 20]
    }
  },
  computed: {
    useNativeMp4() {
      return !!(this.streamInfo && (this.streamInfo.mp4 || (this.streamInfo.flv && /\.mp4(\?|$|#)/i.test(this.streamInfo.flv))))
    },
    nativeMp4Url() {
      if (!this.streamInfo) return ''
      return this.streamInfo.mp4 || this.streamInfo.flv || ''
    },
    showPlayTimeValue() {
      return this.streamInfo === null ? '--:--:--' : moment.duration(this.playerTime, 'milliseconds').format('hh:mm:ss', {
        trim: false
      })
    },
    playTimeValue() {
      const duration = this.streamInfo && this.streamInfo.duration
      if (!duration) return { width: '0%' }
      return { width: (this.playerTime || 0) / duration * 100 + '%' }
    },
    showPlayTimeTotal() {
      if (this.streamInfo === null) {
        return '--:--:--'
      }else {
        return moment.duration(this.streamInfo.duration, 'milliseconds').format('hh:mm:ss', {
          trim: false
        })
      }
    },
    playTimeTotal() {
      const duration = this.streamInfo && this.streamInfo.duration
      if (!duration) return { left: 'calc(0% - 6px)' }
      return { left: `calc(${(this.playerTime || 0) / duration}*100% - 6px)` }
    },
    playTimeTitleStyle() {
      return { left: (this.showTimeLeft - 16) + 'px' }
    },
    showPlayTimeTitle() {
      if (this.showTimeLeft) {
        let time = this.showTimeLeft / this.$refs.timeProcess.clientWidth * this.streamInfo.duration
        let realTime = this.timeLen/this.streamInfo.duration * time + this.startTime
        return `${moment(time).format('mm:ss')}(${moment(realTime).format('HH:mm:ss')})`
      }else {
        return ''
      }
    }
  },
  created() {
    document.addEventListener('mousemove', this.timeProcessMousemove)
    document.addEventListener('mouseup', this.timeProcessMouseup)
  },
  destroyed() {
    this.$destroy('recordVideoPlayer')
  },
  methods: {
    timeProcessMouseup(event) {
      this.isMousedown = false
    },
    timeProcessMousemove(event) {

    },
    timeProcessClick(event) {
      if (!this.streamInfo || !this.streamInfo.duration) {
        return
      }
      let x = event.offsetX
      let clientWidth = this.$refs.timeProcess.clientWidth
      this.seekRecord(x / clientWidth * this.streamInfo.duration)
    },
    timeProcessMousedown(event) {
      this.isMousedown = true
    },
    timeProcessMouseEnter(event) {
      this.showTimeLeft = event.offsetX
    },
    timeProcessMouseMove(event) {
      this.showTimeLeft = event.offsetX
    },
    timeProcessMouseLeave(event) {
      this.showTimeLeft = null
    },
    sidebarControl() {
      this.showSidebar = !this.showSidebar
      this.showListCallback(this.showSidebar)
    },
    snap() {
      if (this.useNativeMp4 && this.$refs.nativeVideo) {
        try {
          const v = this.$refs.nativeVideo
          const canvas = document.createElement('canvas')
          canvas.width = v.videoWidth || 1280
          canvas.height = v.videoHeight || 720
          canvas.getContext('2d').drawImage(v, 0, 0, canvas.width, canvas.height)
          const a = document.createElement('a')
          a.href = canvas.toDataURL('image/jpeg')
          a.download = 'snap.jpg'
          a.click()
        } catch (e) {
          console.warn(e)
        }
        return
      }
      if (this.$refs.recordVideoPlayer) {
        this.$refs.recordVideoPlayer.screenshot()
      }
    },
    refresh() {
      if (this.$refs.recordVideoPlayer) {
        this.$refs.recordVideoPlayer.destroy()
      }
    },
    playLast() {
      this.showLastCallback()
    },
    playNext() {
      this.showNextCallback()
    },
    changePlaySpeed(speed) {
      this.playSpeed = speed
      if (this.useNativeMp4 && this.$refs.nativeVideo) {
        this.$refs.nativeVideo.playbackRate = speed
        return
      }
      this.$store.dispatch('cloudRecord/speed', {
        mediaServerId: this.streamInfo.mediaServerId,
        app: this.streamInfo.app,
        stream: this.streamInfo.stream,
        key: this.streamInfo.key,
        speed: this.playSpeed,
        schema: 'ts'
      })
      if (this.$refs.recordVideoPlayer) {
        this.$refs.recordVideoPlayer.setPlaybackRate(this.playSpeed)
      }
    },
    seekBackward() {
      this.seekRecord(this.playerTime - 5 * 1000)
    },
    seekForward() {
      this.seekRecord(this.playerTime + 5 * 1000)
    },
    stopPLay() {
      if (this.$refs.nativeVideo) {
        this.$refs.nativeVideo.pause()
        this.$refs.nativeVideo.removeAttribute('src')
        this.$refs.nativeVideo.load()
      }
      if (this.$refs.recordVideoPlayer) {
        this.$refs.recordVideoPlayer.destroy()
      }
      const box = this.$el && this.$el.querySelector('.cloud-record-playBox')
      if (box) {
        box.style.aspectRatio = ''
      }
      this.streamInfo = null
      this.playerTime = null
      this.playSpeed = 1
      this.playing = false
    },
    pausePlay() {
      if (this.useNativeMp4 && this.$refs.nativeVideo) {
        this.$refs.nativeVideo.pause()
        return
      }
      if (this.$refs.recordVideoPlayer) {
        this.$refs.recordVideoPlayer.pause()
      }
    },
    play() {
      if (!this.streamInfo) {
        return
      }
      if (this.useNativeMp4) {
        this.$nextTick(() => this.playNativeMp4())
        return
      }
      if (!this.$refs.recordVideoPlayer) {
        return
      }
      const active = this.$refs.recordVideoPlayer.getActivePlayer()
      const inst = this.$refs.recordVideoPlayer.$refs[active]
      if (inst && inst.loaded) {
        if (typeof inst.unPause === 'function') {
          inst.unPause()
        } else if (typeof inst.playBtnClick === 'function') {
          inst.playBtnClick()
        } else {
          this.$refs.recordVideoPlayer.play()
        }
        return
      }
      this.$refs.recordVideoPlayer.setStreamInfo(this.streamInfo)
    },
    playNativeMp4() {
      const v = this.$refs.nativeVideo
      if (!v || !this.nativeMp4Url) {
        return
      }
      if (v.src !== this.nativeMp4Url) {
        v.src = this.nativeMp4Url
      }
      const p = v.play()
      if (p && typeof p.catch === 'function') {
        p.catch(() => {})
      }
    },
    onNativeLoaded() {
      const v = this.$refs.nativeVideo
      if (!v || !this.streamInfo) {
        return
      }
      if ((!this.streamInfo.duration || this.streamInfo.duration <= 0) && v.duration && isFinite(v.duration)) {
        this.streamInfo.duration = v.duration * 1000
      }
      v.playbackRate = this.playSpeed || 1
      // 弹窗画幅按真实分辨率，避免左右黑边
      if (v.videoWidth > 0 && v.videoHeight > 0) {
        const box = this.$el && this.$el.querySelector('.cloud-record-playBox')
        if (box && this.$el.classList.contains('cloud-record-play-stage')) {
          box.style.aspectRatio = `${v.videoWidth} / ${v.videoHeight}`
        }
      }
    },
    onNativeTimeUpdate() {
      const v = this.$refs.nativeVideo
      if (!v) {
        return
      }
      this.playerTime = Math.max(0, v.currentTime * 1000)
    },
    fullScreen() {
      if (this.isFullScreen) {
        screenfull.exit()
        this.isFullScreen = false
        return
      }
      const playerWrapper = this.$refs.recordVideoPlayer ? this.$refs.recordVideoPlayer.$refs.playerWrapper : null
      const playerWidth = playerWrapper ? playerWrapper.clientWidth : 0
      const playerHeight = playerWrapper ? playerWrapper.clientHeight : 0
      screenfull.request(document.getElementById('cloudRecordPlayer'))
      screenfull.on('change', (event) => {
        if (this.$refs.recordVideoPlayer) {
          this.$refs.recordVideoPlayer.resize(playerWidth, playerHeight)
        }
        this.isFullScreen = screenfull.isFullscreen
      })
      this.isFullScreen = true
    },
    setStreamInfo(streamInfo, timeLen, startTime) {
      if (streamInfo && (!streamInfo.duration || streamInfo.duration <= 0) && timeLen) {
        streamInfo.duration = timeLen
      }
      this.streamInfo = streamInfo
      this.timeLen = timeLen
      this.startTime = startTime
      this.playerTime = 0
      this.$nextTick(() => {
        if (this.useNativeMp4) {
          this.playNativeMp4()
          return
        }
        if (this.$refs.recordVideoPlayer) {
          this.$refs.recordVideoPlayer.setStreamInfo(streamInfo)
          this.$nextTick(() => this.$refs.recordVideoPlayer.syncPlayerSize())
        }
      })
    },
    seekRecord(playSeekValue, callback) {
      if (!this.streamInfo) {
        return
      }
      let ms = playSeekValue
      if (ms < 0) ms = 0
      if (this.streamInfo.duration && ms > this.streamInfo.duration) {
        ms = this.streamInfo.duration
      }
      if (this.useNativeMp4 && this.$refs.nativeVideo) {
        this.$refs.nativeVideo.currentTime = ms / 1000
        this.playerTime = ms
        if (callback) callback(ms)
        return
      }
      this.$store.dispatch('cloudRecord/seek', {
        mediaServerId: this.streamInfo.mediaServerId,
        app: this.streamInfo.app,
        stream: this.streamInfo.stream,
        seek: ms,
        schema: 'fmp4'
      })
        .then(() => {
          this.playerTime = ms
          if (callback) {
            callback(ms)
          }
        })
        .catch((error) => {
          console.log(error)
        })
    },
    showPlayTimeChange(val) {
      if (Number(val)) {
        this.playerTime = Number(val)
      }
    },
    playingChange(val) {
      this.playing = val
    }
  }
}
</script>

<style>
.cloud-record-player-root {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background-color: #111;
}
.cloud-record-playBox {
  flex: 1;
  min-height: 0;
  width: 100%;
  background-color: #000;
  display: flex;
  align-items: stretch;
  justify-content: center;
  overflow: hidden;
}
.cloud-record-playBox > * {
  width: 100%;
  height: 100%;
  min-height: 0;
}
.cloud-record-native-video {
  width: 100%;
  height: 100%;
  object-fit: contain;
  background: #000;
  outline: none;
}
/* 弹窗：画幅与窗口同宽，默认 16:9，metadata 后按真实比例覆盖 */
.cloud-record-play-stage .cloud-record-player-root {
  height: auto;
}
.cloud-record-play-stage .cloud-record-playBox {
  flex: none;
  aspect-ratio: 16 / 9;
  height: auto;
}
.cloud-record-play-stage .cloud-record-native-video {
  object-fit: contain;
}
.cloud-record-control-bar {
  height: 40px;
  flex-shrink: 0;
  background-color: #2a2a2a;
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  border-top: 1px solid #333;
}
.cloud-record-record-play-control {
  height: 32px;
  line-height: 32px;
  display: inline-block;
  width: fit-content;
  padding: 0 10px;
  box-shadow: none;
  background-color: transparent;
  margin: 4px 0;
}
.cloud-record-record-play-control-item {
  display: inline-block;
  padding: 0 10px;
  color: #fff;
  margin-right: 2px;
}
.cloud-record-record-play-control-item:hover {
  color: #4da3ff;
}
.cloud-record-record-play-control-speed {
  font-weight: bold;
  color: #fff;
  user-select: none;
}
.cloud-record-player-option-box {
  height: 20px;
  width: 100%;
  flex-shrink: 0;
  display: grid;
  grid-template-columns: 70px auto 70px;
  background-color: #1f1f1f;
}
.cloud-record-time-process {
  width: 100%;
  height: 8px;
  margin: 6px 0 ;
  border-radius: 4px;
  border: 1px solid #505050;
  background-color: rgb(56, 56, 56);
  cursor: pointer;
}
.cloud-record-show-time {
  color: #FFFFFF;
  text-align: center;
  font-size: 14px;
  line-height: 20px
}
.cloud-record-time-process-value {
  width: 100%;
  height: 6px;
  background-color: rgb(162, 162, 162);
}
.cloud-record-time-process-value1::after {
  content: '';
  display: block;
  width: 12px;
  height: 12px;
  background-color: rgb(192 190 190);
  border-radius: 5px;
  position: relative;
  top: -3px;
  right: -6px;
  float: right;
}
.cloud-record-time-process-title {
  width: fit-content;
  text-align: center;
  position: relative;
  top: -35px;
  color: rgb(217, 217, 217);
  font-size: 14px;
  text-shadow:
    -1px -1px 0 black,
    1px -1px 0 black,
    -1px 1px 0 black,
    1px 1px 0 black;
}
</style>
