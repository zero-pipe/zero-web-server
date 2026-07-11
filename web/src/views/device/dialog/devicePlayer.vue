<template>
  <div id="devicePlayer">
    <el-dialog
      v-if="showVideoDialog"
      v-el-drag-dialog
      custom-class="vms-player-dialog"
      top="5vh"
      width="1040px"
      :close-on-click-modal="false"
      :visible.sync="showVideoDialog"
      @close="close()"
    >
      <div slot="title" class="vms-player-header">
        <span class="vms-live-dot" :class="{ 'is-idle': !isStreaming }" />
        <span class="vms-live-label" :class="{ 'is-idle': !isStreaming }">{{ isStreaming ? 'LIVE' : 'IDLE' }}</span>
        <span class="vms-player-select">
          播放器:
          <el-select v-model="selectedPlayer" size="mini" style="width: 120px" @change="onPlayerChange">
            <el-option label="Jessibuca" value="jessibuca" />
            <el-option label="WebRTC" value="webRTC" />
            <el-option label="H265web" value="h265web" />
          </el-select>
        </span>
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
                :show-button="true"
                :show-tab="false"
                :preferred-player="selectedPlayer"
                @playerChanged="playerChanged"
              />
            </div>
          </div>

          <div class="player-under panel-block control-extra">
            <el-tabs v-model="extraTab" @tab-click="extraTabClick">
              <el-tab-pane label="实时视频" name="media">
                <streamMediaPanel
                  v-if="extraTab === 'media'"
                  :player-url="playerUrlInfo.playerUrl"
                  :play-url="playerUrlInfo.playUrl"
                  :stream-info="streamInfo"
                />
              </el-tab-pane>
              <el-tab-pane label="预置位" name="preset">
                <ptzPreset
                  v-if="extraTab === 'preset'"
                  :device-id="deviceId"
                  :channel-device-id="channelId"
                />
              </el-tab-pane>
            </el-tabs>
          </div>
        </div>

        <div class="control-side">
          <div class="panel-block">
            <div class="panel-block-title">
              <span>编码信息</span>
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

          <div class="panel-block is-ptz">
            <div class="panel-block-title">云台控制</div>
            <devicePtzPanel
              :device-id="deviceId"
              :channel-id="channelId"
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
import streamMediaPanel from '../../common/streamMediaPanel.vue'

function isDomeCamera(ptzType) {
  const t = Number(ptzType)
  return t === 1 || t === 2
}

export default {
  name: 'DevicePlayer',
  directives: { elDragDialog },
  components: { playerTabs, devicePtzPanel, PtzPreset, mediaInfo, streamMediaPanel },
  data() {
    return {
      videoUrl: '',
      streamId: '',
      app: '',
      mediaServerId: '',
      deviceId: '',
      channelId: '',
      ptzType: 0,
      extraTab: 'media',
      hasAudio: false,
      isLoging: false,
      playError: '',
      showVideoDialog: false,
      streamInfo: null,
      selectedPlayer: 'jessibuca',
      playerUrlInfo: {
        playerUrl: null,
        playUrl: null
      },
      dragZoomDirection: ''
    }
  },
  created() {
    const saved = window.localStorage.getItem('globalPlayer')
    if (saved && ['jessibuca', 'webRTC', 'h265web'].includes(saved)) {
      this.selectedPlayer = saved
    }
  },
  computed: {
    ptzEnabled() {
      return isDomeCamera(this.ptzType)
    },
    isStreaming() {
      return !!(this.streamInfo && this.streamId) && !this.isLoging && !this.playError
    }
  },
  methods: {
    onPlayerChange(key) {
      this.selectedPlayer = key
      window.localStorage.setItem('globalPlayer', key)
      if (this.$refs.playerTabs) {
        this.$refs.playerTabs.switchPlayer(key)
      }
    },
    extraTabClick() {},
    refreshMediaInfo() {
      this.$refs.mediaInfo && this.$refs.mediaInfo.getMediaInfo()
    },
    openDialog(tab, deviceId, channelId, param) {
      if (this.showVideoDialog) return
      this.deviceId = deviceId
      this.channelId = channelId
      this.ptzType = (param && param.ptzType != null) ? param.ptzType : 0
      this.extraTab = tab === 'preset' ? 'preset' : 'media'
      this.streamId = ''
      this.mediaServerId = ''
      this.app = ''
      this.videoUrl = ''
      this.playError = ''
      this.showVideoDialog = true
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
      this.isLoging = false
      this.playError = ''
      this.streamInfo = null
      this.videoUrl = ''
      this.ptzType = 0
      this.extraTab = 'media'
      this.showVideoDialog = false
    },
    toggleDragZoom(direction) {
      this.dragZoomDirection = direction
      this.$refs.playerTabs.startDragZoom((params) => {
        params.deviceId = this.deviceId
        params.channelId = this.channelId
        const action = this.dragZoomDirection === 'in' ? 'frontEnd/dragZoomIn' : 'frontEnd/dragZoomOut'
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
