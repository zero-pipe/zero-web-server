<template>
  <div id="channelList" style="height: calc(100vh - 124px);">
    <el-form :inline="true" size="mini">
      <el-form-item style="margin-right: 2rem">
        <el-page-header content="通道列表" @back="$emit('show-device')" />
      </el-form-item>
      <el-form-item label="搜索">
        <el-input
          v-model="searchStr"
          style="margin-right: 1rem; width: auto;"
          placeholder="关键字"
          prefix-icon="el-icon-search"
          clearable
          @input="filterChannels"
        />
      </el-form-item>
      <el-form-item>
        <el-button icon="el-icon-refresh-right" @click="handleSync">同步通道</el-button>
      </el-form-item>
      <el-form-item style="float: right;">
        <el-button icon="el-icon-refresh-right" circle :loading="loading" @click="loadChannels" />
      </el-form-item>
    </el-form>

    <el-table
      v-loading="loading"
      :data="filteredList"
      size="small"
      height="calc(100% - 64px)"
      style="width: 100%; font-size: 12px;"
      header-row-class-name="table-header"
    >
      <el-table-column prop="name" label="名称" min-width="180" />
      <el-table-column label="开启音频" min-width="100">
        <template v-slot:default="scope">
          <el-switch
            v-model="scope.row.hasAudio"
            active-color="#409EFF"
            @change="() => onAudioChange(scope.row)"
          />
        </template>
      </el-table-column>
      <el-table-column label="码流类型" min-width="200">
        <template v-slot:default="scope">
          <el-select
            v-model="scope.row.selectedProfileToken"
            size="mini"
            style="width: 100%;"
            placeholder="请选择码流类型"
            @change="(val) => onStreamTypeChange(scope.row, val)"
          >
            <el-option
              v-for="sp in scope.row.streamProfiles"
              :key="sp.profileToken"
              :label="profileOptionLabel(sp)"
              :value="sp.profileToken"
            />
          </el-select>
        </template>
      </el-table-column>
      <el-table-column label="状态" min-width="90">
        <template v-slot:default="scope">
          <el-tag v-if="scope.row.status === 'ON'" size="medium">在线</el-tag>
          <el-tag v-else size="medium" type="info">离线</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" min-width="200" fixed="right">
        <template v-slot:default="scope">
          <el-button
            size="medium"
            type="text"
            icon="el-icon-video-play"
            :loading="scope.row.playLoading"
            @click="handlePlay(scope.row)"
          >播放
          </el-button>
          <template v-if="scope.row.hasPtz">
            <el-divider direction="vertical" />
            <el-dropdown @command="(cmd) => handlePtz(scope.row, cmd)">
              <el-button size="medium" type="text">
                云台<i class="el-icon-arrow-down el-icon--right" />
              </el-button>
              <el-dropdown-menu>
                <el-dropdown-item command="LEFT">左转</el-dropdown-item>
                <el-dropdown-item command="RIGHT">右转</el-dropdown-item>
                <el-dropdown-item command="STOP">停止</el-dropdown-item>
              </el-dropdown-menu>
            </el-dropdown>
          </template>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog
      v-el-drag-dialog
      custom-class="vms-player-dialog"
      top="5vh"
      width="1040px"
      :close-on-click-modal="false"
      :visible.sync="playerVisible"
      @close="handleStopCurrent"
    >
      <div slot="title" class="vms-player-header">
        <span class="vms-live-dot" :class="{ 'is-idle': !isStreaming }" />
        <span class="vms-live-label" :class="{ 'is-idle': !isStreaming }">{{ isStreaming ? 'LIVE' : 'IDLE' }}</span>
        <span v-if="playMeta && playMeta.streamType" class="onvif-header-sub">{{ playMeta.streamType }}</span>
      </div>

      <div class="dhsdk-player-body">
        <div class="player-side">
          <div class="player-stage">
            <div
              class="player-container"
              v-loading="isPlaying"
              element-loading-text="正在拉流…"
              element-loading-background="rgba(240, 244, 248, 0.85)"
              element-loading-spinner="el-icon-loading"
            >
              <playerTabs
                v-if="streamInfo"
                ref="playerTabs"
                :has-audio="hasAudio"
                :show-button="true"
                :show-tab="allowedPlayers.length > 1"
                :allowed-players="allowedPlayers"
                :url-priority="playerUrlPriority"
                :preferred-player="preferredPlayer"
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
              <el-tab-pane label="码流信息" name="meta">
                <div class="onvif-meta-grid">
                  <div class="onvif-meta-item">
                    <span class="k">码流</span>
                    <span class="v">{{ (playMeta && playMeta.streamType) || '—' }}</span>
                  </div>
                  <div class="onvif-meta-item">
                    <span class="k">RTSP 通道</span>
                    <span class="v">{{ (playMeta && playMeta.streamChannel) || '—' }}</span>
                  </div>
                  <div class="onvif-meta-item">
                    <span class="k">配置编码</span>
                    <span class="v">{{ (playMeta && playMeta.configCodec) || '—' }}</span>
                  </div>
                  <div class="onvif-meta-item">
                    <span class="k">实测编码</span>
                    <span class="v">{{ (playMeta && playMeta.videoCodec) || '—' }}</span>
                  </div>
                  <div class="onvif-meta-item">
                    <span class="k">分辨率</span>
                    <span class="v">{{ (playMeta && playMeta.mediaResolution) || '—' }}</span>
                  </div>
                  <div class="onvif-meta-item">
                    <span class="k">播放器</span>
                    <span class="v">{{ preferredPlayer || '—' }}</span>
                  </div>
                </div>
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
              v-if="app && streamId && mediaServerId"
              ref="mediaInfo"
              :app="app"
              :stream="streamId"
              :media-server-id="mediaServerId"
            />
            <div v-else class="panel-disabled-tip">等待流媒体信息…</div>
          </div>

          <div class="panel-block is-ptz">
            <div class="panel-block-title">云台控制</div>
            <ptzControls
              v-if="currentHasPtz"
              btn-layout="row"
              :show-diagonals="false"
              @ptz-move="onPtzMove"
              @ptz-stop="onPtzStop"
            />
            <div v-else class="panel-disabled-tip">当前通道不支持云台控制</div>
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import elDragDialog from '@/directive/el-drag-dialog'
import playerTabs from '@/views/common/playerTabs.vue'
import streamMediaPanel from '@/views/common/streamMediaPanel.vue'
import mediaInfo from '@/views/common/mediaInfo.vue'
import ptzControls from '@/views/common/ptzControls.vue'
import { ptzControl, queryChannels, startPlay, stopPlay, syncDevice, updateChannel } from '@/api/onvif'
import { getMediaServerList } from '@/api/server'
import { isH265Codec, resolvePlayerStrategy } from '@/utils/playerStrategy'

const PTZ_CMD_MAP = {
  left: 'LEFT',
  right: 'RIGHT',
  up: 'UP',
  down: 'DOWN'
}

export default {
  name: 'OnvifChannelList',
  directives: { elDragDialog },
  components: { playerTabs, streamMediaPanel, mediaInfo, ptzControls },
  props: {
    deviceId: {
      type: Number,
      required: true
    }
  },
  data() {
    return {
      loading: false,
      isPlaying: false,
      channelList: [],
      searchStr: '',
      playerVisible: false,
      streamInfo: null,
      playMeta: null,
      hasAudio: false,
      allowedPlayers: ['jessibuca'],
      playerUrlPriority: ['flv', 'ws_flv', 'https_flv', 'wss_flv'],
      preferredPlayer: 'jessibuca',
      currentChannelId: null,
      currentProfileToken: null,
      currentHasPtz: false,
      app: '',
      streamId: '',
      mediaServerId: '',
      extraTab: 'media',
      playerUrlInfo: {
        playerUrl: null,
        playUrl: null
      }
    }
  },
  computed: {
    isStreaming() {
      return !!(this.streamInfo && this.playerVisible)
    },
    filteredList() {
      const q = (this.searchStr || '').trim().toLowerCase()
      if (!q) return this.channelList
      return this.channelList.filter(ch => {
        const blob = `${ch.name || ''} ${ch.profileToken || ''} ${ch.resolution || ''}`.toLowerCase()
        return blob.indexOf(q) >= 0
      })
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
        const list = (res.data && res.data.list) || []
        this.channelList = list.map(ch => this.normalizeChannel(ch))
      }).finally(() => {
        this.loading = false
      })
    },
    normalizeChannel(ch) {
      const profiles = (ch.streamProfiles && ch.streamProfiles.length)
        ? ch.streamProfiles
        : [{
          profileToken: ch.profileToken,
          label: ch.streamType || ch.profileToken,
          resolution: ch.resolution,
          codec: ch.codec,
          streamType: ch.streamType,
          streamChannel: ch.streamChannel,
          hasAudio: ch.hasAudio,
          hasPtz: ch.hasPtz
        }]
      const selected = ch.profileToken || (profiles[0] && profiles[0].profileToken)
      const row = {
        ...ch,
        streamProfiles: profiles,
        selectedProfileToken: selected,
        playLoading: false
      }
      this.applyProfileFields(row, selected)
      return row
    },
    profileOptionLabel(sp) {
      if (!sp) return ''
      const label = sp.label || sp.streamType || '码流'
      // 与国标「主/子码流」语义对齐；分辨率作辅助信息
      return sp.resolution ? `${label}(${sp.resolution})` : label
    },
    applyProfileFields(row, profileToken) {
      const sp = (row.streamProfiles || []).find(p => p.profileToken === profileToken)
      if (!sp) return
      row.selectedProfileToken = sp.profileToken
      row.profileToken = sp.profileToken
      row.resolution = sp.resolution || row.resolution
      row.codec = sp.codec || row.codec
      row.configCodec = sp.codec || row.configCodec
      row.streamType = sp.streamType || sp.label || row.streamType
      row.streamChannel = sp.streamChannel || row.streamChannel
    },
    onAudioChange(row) {
      updateChannel({
        channelId: row.id,
        hasAudio: !!row.hasAudio
      }).catch(() => {
        row.hasAudio = !row.hasAudio
        this.$message.error('更新音频设置失败')
      })
    },
    onStreamTypeChange(row, profileToken) {
      this.applyProfileFields(row, profileToken)
      updateChannel({
        channelId: row.id,
        profileToken
      }).then(() => {
        this.$message.success('码流类型已更新')
      }).catch(err => {
        this.$message.error((err && err.message) || '更新码流类型失败')
        this.loadChannels()
      })
    },
    filterChannels() {
      // filteredList 计算属性已处理
    },
    handleSync() {
      syncDevice(this.deviceId).then(res => {
        const channels = Array.isArray(res.data) ? res.data : []
        let streamOptions = 0
        channels.forEach(ch => {
          const n = (ch.streamProfiles && ch.streamProfiles.length) || 1
          streamOptions += n
        })
        if (channels.length === 0) {
          this.$message.success('同步成功（无通道）')
        } else if (streamOptions <= channels.length) {
          this.$message.warning(
            `同步完成：${channels.length} 个通道，但仅发现 ${streamOptions} 路码流（通常主+子应 ≥2）。请确认摄像机 ONVIF 已开启子码流，或查看服务端日志 rawProfiles`
          )
        } else {
          this.$message.success(`同步成功：${channels.length} 个通道，${streamOptions} 路码流可选`)
        }
        this.loadChannels()
      })
    },
    buildStreamInfo(data) {
      const urls = (data && data.urls) || {}
      return {
        app: data.app,
        stream: data.stream,
        mediaServerId: data.mediaServerId,
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
      this.isPlaying = true
      this.$set(row, 'playLoading', true)
      const profileToken = row.selectedProfileToken || row.profileToken
      startPlay(row.id, profileToken).then(res => {
        const data = res.data || {}
        if (!data.urls || (!data.urls.flv && !data.urls.ws)) {
          this.$message.error('未获取到播放地址')
          return
        }
        if (!data.videoCodec) {
          this.$message.error('未获取到实测编码，请稍后重试')
          return
        }
        this.applyPlayerStrategy({
          ...data,
          hasAudio: !!(row.hasAudio && data.hasAudio)
        })
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
        this.app = data.app || ''
        this.streamId = data.stream || ''
        this.mediaServerId = data.mediaServerId || ''
        this.currentChannelId = row.id
        this.currentProfileToken = profileToken
        this.currentHasPtz = !!row.hasPtz
        this.extraTab = 'media'
        this.playerVisible = true
        const openPlayer = () => {
          this.$nextTick(() => {
            this.$nextTick(() => {
              if (this.$refs.playerTabs) {
                this.$refs.playerTabs.setStreamInfo(this.streamInfo)
              }
              if (this.$refs.mediaInfo) {
                this.$refs.mediaInfo.startTask()
              }
            })
          })
        }
        if (!this.mediaServerId) {
          getMediaServerList().then(listRes => {
            const list = (listRes && listRes.data) || []
            const online = list.find(item => item.status) || list[0]
            if (online && online.id) {
              this.mediaServerId = online.id
            }
          }).finally(openPlayer)
        } else {
          openPlayer()
        }
      }).catch(err => {
        const msg = (err && err.message) || '播放失败'
        this.$message.error(msg)
      }).finally(() => {
        this.isPlaying = false
        this.$set(row, 'playLoading', false)
      })
    },
    destroyPlayer() {
      if (this.$refs.mediaInfo) {
        this.$refs.mediaInfo.stopTask()
      }
      if (this.$refs.playerTabs) {
        this.$refs.playerTabs.destroy()
      }
    },
    closePlayer(callBackendStop) {
      this.destroyPlayer()
      this.playerVisible = false
      this.streamInfo = null
      this.playMeta = null
      this.app = ''
      this.streamId = ''
      this.mediaServerId = ''
      this.playerUrlInfo = { playerUrl: null, playUrl: null }
      if (callBackendStop !== false && this.currentChannelId) {
        stopPlay(this.currentChannelId, this.currentProfileToken)
      }
      this.currentChannelId = null
      this.currentProfileToken = null
      this.currentHasPtz = false
    },
    handleStopCurrent() {
      this.closePlayer(true)
    },
    handlePtz(row, command) {
      ptzControl({ channelId: row.id, command, speed: 0.5 })
    },
    onPtzMove(e) {
      if (!this.currentChannelId) return
      const command = PTZ_CMD_MAP[e.direction]
      if (!command) {
        // ONVIF 当前仅支持方向云台
        return
      }
      const speed = Math.min(1, Math.max(0.05, (e.speed || 50) / 100))
      ptzControl({ channelId: this.currentChannelId, command, speed })
    },
    onPtzStop() {
      if (!this.currentChannelId) return
      ptzControl({ channelId: this.currentChannelId, command: 'STOP', speed: 0 })
    },
    playerChanged(playerUrlInfo) {
      this.playerUrlInfo = playerUrlInfo || { playerUrl: null, playUrl: null }
    },
    refreshMediaInfo() {
      if (this.$refs.mediaInfo) {
        this.$refs.mediaInfo.getMediaInfo()
      }
    }
  }
}
</script>

<style scoped>
.text-muted {
  color: #c0c4cc;
}

.hint-text {
  font-size: 12px;
  color: #909399;
}

.onvif-header-sub {
  margin-left: 4px;
  padding: 2px 8px;
  border-radius: 4px;
  background: #e8f1fb;
  color: #1565c0;
  font-size: 12px;
  font-weight: 500;
}

.onvif-meta-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px 16px;
  padding: 4px 2px 8px;
}

.onvif-meta-item {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  font-size: 12px;
  line-height: 1.6;
}

.onvif-meta-item .k {
  color: #94a3b8;
  flex-shrink: 0;
}

.onvif-meta-item .v {
  color: #334155;
  font-weight: 500;
  text-align: right;
  word-break: break-all;
}
</style>
