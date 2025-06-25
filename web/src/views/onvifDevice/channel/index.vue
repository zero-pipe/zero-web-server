<template>
  <div style="height: calc(100vh - 124px);">
    <el-page-header content="ONVIF 通道" @back="$emit('show-device')" />

    <el-form :inline="true" size="mini" style="margin-top: 12px;">
      <el-form-item>
        <el-button type="primary" icon="el-icon-refresh" @click="loadChannels">刷新</el-button>
        <el-button icon="el-icon-refresh-right" @click="handleSync">同步通道</el-button>
      </el-form-item>
      <el-form-item>
        <span class="hint-text">播放策略：H264→Jessibuca，H265→H265web；以 ZMS 实测编码为准</span>
      </el-form-item>
    </el-form>

    <el-table v-loading="loading" :data="channelList" size="small" height="calc(100% - 100px)">
      <el-table-column prop="name" label="码流" min-width="140" />
      <el-table-column label="RTSP" width="72">
        <template v-slot:default="scope">
          {{ scope.row.streamChannel || '-' }}
        </template>
      </el-table-column>
      <el-table-column prop="profileToken" label="Profile" min-width="120" />
      <el-table-column prop="resolution" label="配置分辨率" width="110" />
      <el-table-column label="配置编码" width="100">
        <template v-slot:default="scope">
          {{ scope.row.configCodec || scope.row.codec || '-' }}
        </template>
      </el-table-column>
      <el-table-column label="PTZ" width="72">
        <template v-slot:default="scope">
          <el-tag v-if="scope.row.hasPtz" size="mini" type="success">支持</el-tag>
          <el-tag v-else size="mini" type="info">否</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="320" fixed="right">
        <template v-slot:default="scope">
          <el-button size="mini" type="primary" @click="handlePlay(scope.row)">播放</el-button>
          <el-button size="mini" @click="handleStop(scope.row)">停止</el-button>
          <el-button v-if="scope.row.hasPtz" size="mini" @click="handlePtz(scope.row, 'LEFT')">左</el-button>
          <el-button v-if="scope.row.hasPtz" size="mini" @click="handlePtz(scope.row, 'RIGHT')">右</el-button>
          <el-button v-if="scope.row.hasPtz" size="mini" @click="handlePtz(scope.row, 'STOP')">停</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog
      v-el-drag-dialog
      title="视频播放"
      top="8vh"
      width="960px"
      :close-on-click-modal="false"
      :visible.sync="playerVisible"
      @close="handleStopCurrent"
    >
      <div v-if="playMeta" class="play-meta">
        <span>{{ playMeta.streamType || '码流' }}</span>
        <span v-if="playMeta.streamChannel">RTSP {{ playMeta.streamChannel }}</span>
        <span>配置 {{ playMeta.configCodec || '-' }}</span>
        <span>实测 {{ playMeta.videoCodec || '-' }}</span>
        <span v-if="playMeta.mediaResolution">分辨率 {{ playMeta.mediaResolution }}</span>
        <span v-if="playMeta.preferredPlayer">播放器 {{ playMeta.preferredPlayer }}</span>
      </div>
      <div v-if="streamInfo" class="onvif-player-wrap">
        <playerTabs
          ref="playerTabs"
          :has-audio="hasAudio"
          :show-button="true"
          :show-tab="allowedPlayers.length > 1"
          :allowed-players="allowedPlayers"
          :url-priority="playerUrlPriority"
          :preferred-player="preferredPlayer"
        />
      </div>
    </el-dialog>
  </div>
</template>

<script>
import elDragDialog from '@/directive/el-drag-dialog'
import playerTabs from '@/views/common/playerTabs.vue'
import { ptzControl, queryChannels, startPlay, stopPlay, syncDevice } from '@/api/onvif'
import { isH265Codec, resolvePlayerStrategy } from '@/utils/playerStrategy'

export default {
  name: 'OnvifChannelList',
  directives: { elDragDialog },
  components: { playerTabs },
  props: {
    deviceId: {
      type: Number,
      required: true
    }
  },
  data() {
    return {
      loading: false,
      channelList: [],
      playerVisible: false,
      streamInfo: null,
      playMeta: null,
      hasAudio: false,
      allowedPlayers: ['jessibuca'],
      playerUrlPriority: ['flv', 'ws_flv', 'https_flv', 'wss_flv'],
      preferredPlayer: 'jessibuca',
      currentChannelId: null
    }
  },
  watch: {
    deviceId: {
      immediate: true,
      handler() {
        this.loadChannels()
      }
    }
  },
  methods: {
    loadChannels() {
      this.loading = true
      queryChannels({ page: 1, count: 100, deviceId: this.deviceId }).then(res => {
        this.channelList = (res.data && res.data.list) || []
      }).finally(() => {
        this.loading = false
      })
    },
    handleSync() {
      syncDevice(this.deviceId).then(() => {
        this.$message.success('同步成功')
        this.loadChannels()
      })
    },
    buildStreamInfo(data) {
      const urls = (data && data.urls) || {}
      return {
        app: data.app,
        stream: data.stream,
        flv: urls.flv,
        ws_flv: urls.ws,
        hls: urls.hls,
        rtsp: urls.rtsp,
        rtmp: urls.rtmp,
        rtc: urls.rtc,
        rtcs: urls.rtcs,
        videoCodec: data.videoCodec,
        audioCodec: data.audioCodec
      }
    },
    applyPlayerStrategy(data) {
      const strategy = resolvePlayerStrategy({
        videoCodec: data.videoCodec,
        configCodec: data.configCodec,
        audioCodec: data.audioCodec,
        hasAudio: data.hasAudio
      })
      this.allowedPlayers = strategy.allowedPlayers
      this.playerUrlPriority = strategy.urlPriority
      this.preferredPlayer = data.preferredPlayer || strategy.preferredPlayer
      this.hasAudio = strategy.hasAudio
      return strategy
    },
    handlePlay(row) {
      if (this.playerVisible) {
        this.destroyPlayer()
      }
      startPlay(row.id).then(res => {
        const data = res.data || {}
        if (!data.urls || (!data.urls.flv && !data.urls.ws)) {
          this.$message.error('未获取到播放地址')
          return
        }
        if (!data.videoCodec) {
          this.$message.error('未获取到实测编码，请稍后重试')
          return
        }
        const strategy = this.applyPlayerStrategy(data)
        const streamInfo = this.buildStreamInfo(data)
        this.playMeta = {
          streamType: data.streamType,
          streamChannel: data.streamChannel,
          configCodec: data.configCodec,
          videoCodec: data.videoCodec,
          mediaResolution: data.mediaResolution,
          preferredPlayer: this.preferredPlayer
        }
        if (data.configCodec && data.videoCodec &&
            String(data.configCodec).toUpperCase() !== String(data.videoCodec).toUpperCase()) {
          this.$message.warning(
            `ONVIF 配置为 ${data.configCodec}，ZMS 实测为 ${data.videoCodec}，已按实测选择 ${this.preferredPlayer}`
          )
        } else if (isH265Codec(data.videoCodec)) {
          this.$message.info('实测 H265，使用 H265web 播放')
        }
        this.streamInfo = streamInfo
        this.currentChannelId = row.id
        this.playerVisible = true
        this.$nextTick(() => {
          this.$nextTick(() => {
            if (this.$refs.playerTabs) {
              this.$refs.playerTabs.setStreamInfo(this.streamInfo)
            }
          })
        })
      }).catch(err => {
        const msg = (err && err.message) || '播放失败'
        this.$message.error(msg)
      })
    },
    handleStop(row) {
      stopPlay(row.id).then(() => {
        if (this.currentChannelId === row.id) {
          this.closePlayer(false)
        }
        this.$message.success('已停止')
      })
    },
    destroyPlayer() {
      if (this.$refs.playerTabs) {
        this.$refs.playerTabs.destroy()
      }
    },
    closePlayer(callBackendStop) {
      this.destroyPlayer()
      this.playerVisible = false
      this.streamInfo = null
      this.playMeta = null
      if (callBackendStop !== false && this.currentChannelId) {
        stopPlay(this.currentChannelId)
      }
      this.currentChannelId = null
    },
    handleStopCurrent() {
      this.closePlayer(true)
    },
    handlePtz(row, command) {
      ptzControl({ channelId: row.id, command, speed: 0.5 })
    }
  }
}
</script>

<style scoped>
.onvif-player-wrap {
  width: 100%;
  height: 480px;
}
.play-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 8px;
  font-size: 12px;
  color: #606266;
}
.hint-text {
  font-size: 12px;
  color: #909399;
}
</style>
