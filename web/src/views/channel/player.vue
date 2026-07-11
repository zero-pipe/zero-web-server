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
        <span>视频播放</span>
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
                @playerChanged="playerChanged"
              />
            </div>
          </div>

          <div class="player-under panel-block control-extra">
            <el-tabs v-model="extraTab">
              <el-tab-pane label="实时视频" name="media">
                <streamMediaPanel
                  v-if="extraTab === 'media'"
                  :player-url="playerUrlInfo.playerUrl"
                  :play-url="playerUrlInfo.playUrl"
                  :stream-info="streamInfo"
                />
              </el-tab-pane>
              <el-tab-pane label="预置位" name="preset">
                <channelPreset
                  v-if="extraTab === 'preset'"
                  :channel-id="channelId"
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
            <channelPtzPanel
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
import playerTabs from '../common/playerTabs.vue'
import channelPtzPanel from './common/channelPtzPanel.vue'
import channelPreset from './common/ptzPreset.vue'
import mediaInfo from '../common/mediaInfo.vue'
import streamMediaPanel from '../common/streamMediaPanel.vue'

function isDomeCamera(ptzType) {
  const t = Number(ptzType)
  return t === 1 || t === 2
}

export default {
  name: 'ChannelPlayer',
  directives: { elDragDialog },
  components: { playerTabs, channelPtzPanel, channelPreset, mediaInfo, streamMediaPanel },
  data() {
    return {
      videoUrl: '',
      streamId: '',
      app: '',
      mediaServerId: '',
      channelId: '',
      ptzType: 0,
      extraTab: 'media',
      hasAudio: false,
      isLoging: false,
      playError: '',
      showVideoDialog: false,
      streamInfo: null,
      playerUrlInfo: {
        playerUrl: null,
        playUrl: null
      },
      dragZoomDirection: ''
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
    refreshMediaInfo() {
      this.$refs.mediaInfo && this.$refs.mediaInfo.getMediaInfo()
    },
    openDialog(tab, channelId, param) {
      if (this.showVideoDialog) return
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
