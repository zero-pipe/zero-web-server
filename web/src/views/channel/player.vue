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
      @closed="handleClosed"
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
              />
            </div>
          </div>

          <div class="player-under panel-block control-extra">
            <div class="panel-block-title">
              <span class="pbt-title"><i class="pbt-bar" />预置位</span>
            </div>
            <div v-if="!ptzCaps.direction" class="preset-muted">
              该通道为{{ ptzTypeText }}，不支持预置位
            </div>
            <channelPreset v-else-if="channelId" :channel-id="channelId" />
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
            />
          </div>

          <!-- 云台卡：按可折叠 accordion 渲染；无能力时折叠成一行但保留「展开」兜底。 -->
          <div class="panel-block is-ptz" :class="{ 'is-collapsed': !ptzExpanded }">
            <div class="panel-block-title ptz-card-head" @click="ptzExpanded = !ptzExpanded">
              <span class="pbt-title"><i class="pbt-bar" />云台控制</span>
              <span class="ptz-card-toggle">
                <span v-if="!hasPtzCaps" class="ptz-card-hint">该通道未识别到云台</span>
                <i :class="ptzExpanded ? 'el-icon-arrow-up' : 'el-icon-arrow-down'" />
              </span>
            </div>
            <channelPtzPanel
              v-show="ptzExpanded"
              :channel-id="channelId"
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
import playerTabs from '../common/playerTabs.vue'
import { derivePtzCaps, FULL_PTZ_CAPS } from '@/utils/playerStrategy'
import channelPtzPanel from './common/channelPtzPanel.vue'
import channelPreset from './common/ptzPreset.vue'
import mediaInfo from '../common/mediaInfo.vue'

// 云台能力判定统一收敛到 playerStrategy.derivePtzCaps（唯一真相）。

export default {
  name: 'ChannelPlayer',
  directives: { elDragDialog },
  components: { playerTabs, channelPtzPanel, channelPreset, mediaInfo },
  data() {
    return {
      videoUrl: '',
      streamId: '',
      app: '',
      mediaServerId: '',
      channelId: '',
      channelName: '',
      ptzType: 0,
      ptzExpanded: true,
      extraTab: 'media',
      hasAudio: false,
      isLoging: false,
      playError: '',
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
    ptzCaps() {
      return derivePtzCaps(this.ptzType)
    },
    hasPtzCaps() {
      const c = this.ptzCaps
      return !!(c.direction || c.zoom || c.focus || c.iris)
    },
    /** 传给面板的实际能力集：识别到能力按真值；未识别到但用户手动展开时全开，方便现场兜底操作 */
    effectivePtzCaps() {
      return this.hasPtzCaps ? this.ptzCaps : { ...FULL_PTZ_CAPS }
    },
    // 依据国标目录应答的 PTZType 取中文形态名（仅用于文案展示）
    ptzTypeText() {
      const texts = { 1: '球机', 2: '半球', 3: '固定枪机', 4: '遥控枪机', 5: '遥控半球', 6: '全景通道', 7: '分割通道' }
      return texts[Number(this.ptzType)] || '枪机'
    },
    isStreaming() {
      return !!(this.streamInfo && this.streamId) && !this.isLoging && !this.playError
    }
  },
  beforeDestroy() {
    this.stopClock()
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
    refreshMediaInfo() {
      this.$refs.mediaInfo && this.$refs.mediaInfo.getMediaInfo(false)
    },
    openDialog(tab, channelId, param) {
      if (this.showVideoDialog) return
      this.channelId = channelId
      this.channelName = (param && (param.name || param.channelName)) || ''
      this.ptzType = (param && param.ptzType != null) ? param.ptzType : 0
      // 有能力默认展开，无能力/未知默认折叠（保留手动展开兜底）。
      this.ptzExpanded = this.hasPtzCaps
      this.extraTab = tab === 'preset' ? 'preset' : 'media'
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
          setTimeout(() => this.$refs.playerTabs && this.$refs.playerTabs.syncPlayerSize(), 120)
          setTimeout(() => this.$refs.playerTabs && this.$refs.playerTabs.syncPlayerSize(), 400)
        }
        if (this.$refs.mediaInfo) {
          this.$refs.mediaInfo.startTask()
        }
      })
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
      this.isLoging = false
      this.playError = ''
      this.streamInfo = null
      this.videoUrl = ''
      this.ptzType = 0
      this.channelName = ''
      this.ptzExpanded = true
      this.extraTab = 'media'
      this.showVideoDialog = false
    },
    handleClosed() {
      this.$emit('closed')
    },
    toggleDragZoom(direction) {
      this.dragZoomDirection = direction
      this.$refs.playerTabs.startDragZoom((params) => {
        params.channelId = this.channelId
        const action = this.dragZoomDirection === 'in' ? 'commonChanel/dragZoomIn' : 'commonChanel/dragZoomOut'
        const successMsg = this.dragZoomDirection === 'in' ? '拉框放大成功' : '拉框缩小成功'
        const failMsg = this.dragZoomDirection === 'in' ? '拉框放大失败' : '拉框缩小失败'
        this.$store.dispatch(action, params).then(() => {
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

<style scoped>
.preset-muted {
  padding: 8px 4px;
  color: #80868b;
  font-size: 13px;
}
</style>
