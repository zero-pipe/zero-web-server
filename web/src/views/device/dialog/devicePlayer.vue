<template>
  <div id="devicePlayer">
    <el-dialog
      v-if="showVideoDialog"
      v-el-drag-dialog
      custom-class="vms-player-dialog"
      top="5vh"
      width="1160px"
      :close-on-click-modal="false"
      :visible.sync="showVideoDialog"
      @close="close()"
      @closed="$emit('closed')"
    >
      <div slot="title" class="vms-player-header">
        <span class="vms-live-badge" :class="{ 'is-idle': !isStreaming }">
          <span class="vms-live-dot" />
          <span class="vms-live-label">{{ isStreaming ? 'LIVE' : 'IDLE' }}</span>
        </span>
        <span v-if="channelName" class="vms-channel-name">{{ channelName }}</span>
        <span v-if="channelName" class="vms-channel-type">{{ ptzTypeText }}</span>
        <span v-if="osdClock" class="vms-header-clock">{{ osdClock }}</span>
      </div>

      <div class="dhsdk-player-body">
        <div class="player-side">
          <div class="player-stage">
            <div
              class="player-container"
              v-loading="isLoging"
              element-loading-text="正在邀请设备推流…"
              element-loading-background="rgba(240, 244, 248, 0.85)"
              element-loading-spinner="el-icon-loading"
            >
              <div v-if="playError" class="player-error-tip">{{ playError }}</div>
              <playerTabs
                ref="playerTabs"
                :has-audio="hasAudio"
                :show-button="false"
                :show-tab="false"
                :show-control-bar="true"
                @playerChanged="playerChanged"
                @playStatusChange="onPlayStatus"
              />
            </div>
          </div>

          <div v-if="supportsPreset" class="player-under panel-block control-extra">
            <div class="panel-block-title">
              <span class="pbt-title"><i class="pbt-bar" />预置位</span>
            </div>
            <ptzPreset :resource-id="resourceId" />
          </div>
        </div>

        <div class="control-side">
          <div class="panel-block">
            <div class="panel-block-title">
              <span class="pbt-title"><i class="pbt-bar" />编码信息</span>
              <el-button
                icon="el-icon-refresh-right"
                circle
                size="mini"
                @click="refreshMediaInfo"
              />
            </div>
            <mediaInfo
              ref="mediaInfo"
              :app="app"
              :stream="streamId"
              :media-server-id="mediaServerId"
              @info="onMediaInfo"
            />
          </div>

          <!-- 云台卡：按可折叠 accordion 渲染；无能力时折叠成一行但保留「展开」兜底。
               NVR 混合挂载或协议误报时，现场人员仍可手动展开调用。 -->
          <div v-if="showPtz" class="panel-block is-ptz" :class="{ 'is-collapsed': !ptzExpanded }">
            <div class="panel-block-title ptz-card-head" @click="ptzExpanded = !ptzExpanded">
              <span class="pbt-title"><i class="pbt-bar" />云台控制</span>
              <span class="ptz-card-toggle">
                <span v-if="!hasPtzCaps" class="ptz-card-hint">该通道未识别到云台</span>
                <i :class="ptzExpanded ? 'el-icon-arrow-up' : 'el-icon-arrow-down'" />
              </span>
            </div>
            <devicePtzPanel
              v-show="ptzExpanded"
              :resource-id="resourceId"
              :caps="effectivePtzCaps"
              @drag-zoom-start="toggleDragZoom"
            />
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import elDragDialog from '@/directive/el-drag-dialog'
import playerTabs from '../../common/playerTabs.vue'
import devicePtzPanel from '../common/devicePtzPanel.vue'
import PtzPreset from './ptzPreset.vue'
import mediaInfo from '../../common/mediaInfo.vue'
import { dragZoomIn, dragZoomOut } from '@/api/device'

import { derivePtzCaps, FULL_PTZ_CAPS } from '@/utils/playerStrategy'

export default {
  name: 'DevicePlayer',
  directives: { elDragDialog },
  components: { playerTabs, devicePtzPanel, PtzPreset, mediaInfo },
  data() {
    return {
      videoUrl: '',
      streamId: '',
      app: '',
      mediaServerId: '',
      deviceId: '',
      channelId: '',
      resourceId: '',
      channelName: '',
      ptzType: 0,
      showPtz: true,
      ptzExpanded: true,
      extraTab: 'media',
      hasAudio: false,
      isLoging: false,
      playError: '',
      firstFrameTimer: null,
      showVideoDialog: false,
      streamInfo: null,
      osdClock: '',
      clockTimer: null,
      playerUrlInfo: {
        playerUrl: null,
        playUrl: null
      },
      dragZoomDirection: ''
    }
  },
  computed: {
    /** 云台/镜头能力集（唯一真相，供播放器右侧面板显隐） */
    ptzCaps() {
      return derivePtzCaps(this.ptzType)
    },
    /** 是否有任意云台/镜头能力（决定右侧云台卡是否默认展开） */
    hasPtzCaps() {
      const c = this.ptzCaps
      return !!(c.direction || c.zoom || c.focus || c.iris)
    },
    /** 传给面板的实际能力集：识别到能力按真值；未识别到但用户手动展开时全开，方便现场兜底操作 */
    effectivePtzCaps() {
      return this.hasPtzCaps ? this.ptzCaps : { ...FULL_PTZ_CAPS }
    },
    /** 是否支持预置位：需要转台能力才能定位 */
    supportsPreset() {
      return this.showPtz && this.ptzCaps.direction
    },
    isStreaming() {
      return !!(this.streamInfo && this.streamId) && !this.isLoging && !this.playError
    },
    // 依据国标目录应答的 PTZType 取中文形态名（仅用于文案展示）
    ptzTypeText() {
      const texts = { 1: '球机', 2: '半球', 3: '固定枪机', 4: '遥控枪机', 5: '遥控半球', 6: '全景通道', 7: '分割通道' }
      return texts[Number(this.ptzType)] || '枪机'
    }
  },
  beforeDestroy() {
    this.stopClock()
    this.clearFirstFrameWatchdog()
  },
  methods: {
    tickClock() {
      const d = new Date()
      const p = n => String(n).padStart(2, '0')
      this.osdClock = `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
    },
    startClock() {
      this.stopClock()
      this.tickClock()
      this.clockTimer = setInterval(this.tickClock, 1000)
    },
    stopClock() {
      if (this.clockTimer) {
        clearInterval(this.clockTimer)
        this.clockTimer = null
      }
    },
    extraTabClick() {},
    refreshMediaInfo() {
      this.$refs.mediaInfo && this.$refs.mediaInfo.getMediaInfo()
    },
    /** media_info 轮询到编码后，交给播放器做“纠正式”切换（如发现 H265 自动切到高清播放器） */
    onMediaInfo(info) {
      if (!info) return
      if (this.streamInfo && (info.videoCodec || info.audioCodec)) {
        this.streamInfo = Object.assign({}, this.streamInfo, {
          videoCodec: info.videoCodec || this.streamInfo.videoCodec,
          audioCodec: info.audioCodec || this.streamInfo.audioCodec
        })
      }
      if (this.$refs.playerTabs && this.$refs.playerTabs.updateCodec) {
        this.$refs.playerTabs.updateCodec({ videoCodec: info.videoCodec, audioCodec: info.audioCodec })
      }
    },
    openDialog(tab, deviceId, channelId, param) {
      if (this.showVideoDialog) return
      this.deviceId = deviceId
      this.channelId = channelId
      this.resourceId = (param && param.resourceId) || ''
      this.channelName = (param && (param.name || param.channelName)) || ''
      this.ptzType = (param && param.ptzType != null) ? param.ptzType : 0
      this.showPtz = !(param && param.showPtz === false)
      // 有能力默认展开，无能力/未知默认折叠（保留手动展开兜底）。
      this.ptzExpanded = this.hasPtzCaps
      this.extraTab = (tab === 'preset' && this.showPtz) ? 'preset' : 'media'
      this.streamId = ''
      this.mediaServerId = ''
      this.app = ''
      this.videoUrl = ''
      this.playError = ''
      this.showVideoDialog = true
      this.startClock()
      if (param && param.pending) {
        this.streamInfo = null
        this.hasAudio = !!(param.hasAudio)
        this.isLoging = true
        return
      }
      if (param && param.streamInfo) {
        this.play(param.streamInfo, param.hasAudio)
      }
    },
    onStreamReady(streamInfo, hasAudio) {
      if (!this.showVideoDialog) return
      this.play(streamInfo, hasAudio)
    },
    onStreamError(error) {
      if (!this.showVideoDialog) return
      this.isLoging = false
      const msg = typeof error === 'string' ? error : (error && error.message) || '取流失败，请稍后重试'
      this.playError = msg
      this.$message({ showClose: true, message: msg, type: 'error' })
    },
    play(streamInfo, hasAudio) {
      this.streamInfo = streamInfo
      this.hasAudio = hasAudio
      this.isLoging = false
      this.playError = ''
      this.streamId = streamInfo.stream
      this.app = streamInfo.app
      this.mediaServerId = streamInfo.mediaServerId
      this.showVideoDialog = true
      this.$nextTick(() => {
        if (this.$refs.playerTabs) {
          this.$refs.playerTabs.setStreamInfo(streamInfo.transcodeStream || streamInfo)
          this.$refs.playerTabs.syncPlayerSize && this.$refs.playerTabs.syncPlayerSize()
          // loading 消失后布局会变，延迟再对齐一次，避免国标首开四边黑框
          setTimeout(() => this.$refs.playerTabs && this.$refs.playerTabs.syncPlayerSize(), 120)
          setTimeout(() => this.$refs.playerTabs && this.$refs.playerTabs.syncPlayerSize(), 400)
        }
        if (this.$refs.mediaInfo) {
          this.$refs.mediaInfo.startTask()
        }
        this.startFirstFrameWatchdog()
      })
    },
    // 首帧看门狗：地址先行后若长时间不出帧（真无信号），给超时提示而不是永远转圈。
    startFirstFrameWatchdog() {
      this.clearFirstFrameWatchdog()
      this.firstFrameTimer = setTimeout(() => {
        this.firstFrameTimer = null
        if (this.showVideoDialog && !this.playError) {
          this.playError = '连接超时，请点右下角刷新重试'
        }
      }, 12000)
    },
    clearFirstFrameWatchdog() {
      if (this.firstFrameTimer) {
        clearTimeout(this.firstFrameTimer)
        this.firstFrameTimer = null
      }
    },
    onPlayStatus(playing) {
      if (playing) {
        this.clearFirstFrameWatchdog()
        if (this.playError) this.playError = ''
      }
    },
    playerChanged(playerUrlInfo) {
      this.playerUrlInfo = playerUrlInfo
    },
    close() {
      if (this.$refs.playerTabs) {
        this.$refs.playerTabs.stop()
      }
      if (this.$refs.mediaInfo) {
        this.$refs.mediaInfo.stopTask()
      }
      this.stopClock()
      this.clearFirstFrameWatchdog()
      this.isLoging = false
      this.playError = ''
      this.streamInfo = null
      this.videoUrl = ''
      this.ptzType = 0
      this.channelName = ''
      this.resourceId = ''
      this.showPtz = true
      this.extraTab = 'media'
      this.showVideoDialog = false
    },
    toggleDragZoom(direction) {
      this.dragZoomDirection = direction
      this.$refs.playerTabs.startDragZoom((params) => {
        // 拖动变倍是国标专属能力，走保留的 /devices/gb/control/drag_zoom/*（需 GB 对码）
        params.deviceId = this.deviceId
        params.channelId = this.channelId
        const req = this.dragZoomDirection === 'in' ? dragZoomIn : dragZoomOut
        const successMsg = this.dragZoomDirection === 'in' ? '拉框放大成功' : '拉框缩小成功'
        const failMsg = this.dragZoomDirection === 'in' ? '拉框放大失败' : '拉框缩小失败'
        req(params).then(() => {
          this.$message({ showClose: true, message: successMsg, type: 'success' })
        }).catch(() => {
          this.$message({ showClose: true, message: failMsg, type: 'error' })
        })
        this.dragZoomDirection = ''
      })
    }
  }
}
</script>
