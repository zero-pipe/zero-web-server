<template>
  <div id="live" class="live-container">
    <div v-loading="loading" class="live-content" :class="{ 'sidebar-collapsed': !sidebarVisible }" element-loading-text="拼命加载中">
      <div class="device-tree-container-box" :class="{ 'device-tree-hidden': !sidebarVisible }">
        <DeviceTree @clickEvent="clickEvent" :context-menu-event="contextMenuEvent" />
      </div>
      <div class="video-container">
        <div class="control-bar">
          <div class="split-controls">
            <i :class="['btn', 'sidebar-toggle', sidebarVisible ? 'el-icon-s-fold' : 'el-icon-s-unfold']" title="切换侧边栏" @click="toggleSidebar" />
            <span class="divider" />
            分屏:
            <i class="iconfont icon-a-mti-1fenpingshi btn" :class="{active:spiltIndex === 0}" @click="spiltIndex=0" />
            <i class="iconfont icon-a-mti-4fenpingshi btn" :class="{active: spiltIndex === 1}" @click="spiltIndex=1" />
            <i class="iconfont icon-a-mti-6fenpingshi btn" :class="{active: spiltIndex === 2}" @click="spiltIndex=2" />
            <i class="iconfont icon-a-mti-9fenpingshi btn" :class="{active: spiltIndex === 3}" @click="spiltIndex=3" />
          </div>
          <div class="global-player-control">
            播放器:
            <el-select v-model="globalPlayer" size="mini" style="width: 120px">
              <el-option label="Jessibuca" value="jessibuca" />
              <el-option label="WebRTC" value="webRTC" />
              <el-option label="H265web" value="h265web" />
            </el-select>
          </div>
          <div class="fullscreen-control">
            <i class="el-icon-full-screen btn" @click="fullScreen()" />
            <i class="iconfont icon-PTZ btn" title="云台控制" @click="togglePtzPanel" />
          </div>
        </div>
        <div class="player-container">
          <div
            ref="playBox"
            class="play-grid"
            :style="liveStyle"
          >
            <div
              v-for="i in layout[spiltIndex].spilt"
              :key="i"
              class="play-box"
              :class="getPlayerClass(spiltIndex, i)"
              @click="playerIdx = (i-1)"
            >
              <PlayerTabs
                v-if="streamInfo[i-1]"
                :ref="'playerTabs' + i"
                :show-tab="false"
                :show-button="true"
                :preferred-player="globalPlayer"
                @playStatusChange="onPlayerPlayStatus(i - 1, $event)"
              />
              <div
                v-if="isPulling(i - 1)"
                class="pull-countdown"
                :class="{ 'is-fading': pullFading[i - 1] }"
                aria-live="polite"
              >
                <div
                  class="pull-countdown-num"
                  :key="'cd-' + (i - 1) + '-' + pullCountdown[i - 1] + '-' + pullCycle[i - 1]"
                >
                  {{ pullCountdown[i - 1] }}
                </div>
              </div>
              <div v-else-if="!streamInfo[i-1]" class="no-signal">
                <div class="no-signal-icon" aria-hidden="true">
                  <svg viewBox="0 0 48 48" width="36" height="36">
                    <rect x="8" y="12" width="32" height="24" rx="4" fill="none" stroke="currentColor" stroke-width="2"/>
                    <path d="M20 20l10 6-10 6V20z" fill="currentColor" opacity="0.55"/>
                    <path d="M14 38h20" stroke="currentColor" stroke-width="2" stroke-linecap="round" opacity="0.35"/>
                  </svg>
                </div>
                <div class="no-signal-title">{{ videoTip[i-1] ? videoTip[i-1] : '无信号' }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div v-if="ptzVisible" class="ptz-panel">
        <div class="ptz-panel-header">
          <span>云台控制</span>
          <i class="el-icon-close" @click="ptzVisible = false" />
        </div>
        <div class="ptz-panel-body">
          <template v-if="currentChannelId">
            <div class="ptz-preset-section">
              <div class="section-title">预置位</div>
              <LivePtzPreset :channel-id="String(currentChannelId)" />
            </div>
            <div class="ptz-control-section">
              <div class="section-title">方向控制</div>
              <channelPtzPanel :channel-id="currentChannelId" @drag-zoom-start="handleDragZoom" />
            </div>
          </template>
          <div v-else class="ptz-empty-tip">请先在左侧选择通道</div>
        </div>
      </div>
    </div>
  </div>
</template>
<script>

import PlayerTabs from '../common/playerTabs.vue'
import DeviceTree from '../common/DeviceTree.vue'
import channelPtzPanel from '../channel/common/channelPtzPanel.vue'
import LivePtzPreset from './LivePtzPreset.vue'
import screenFull from 'screenfull'

export default {
  name: 'Live',
  components: {
    PlayerTabs, DeviceTree, channelPtzPanel, LivePtzPreset
  },

  data() {
    return {
      streamInfo: [null],
      videoTip: [''],
      pullLoading: [],
      pullCountdown: [],
      pullCycle: [],
      pullFading: [],
      globalPlayer: 'jessibuca',
      sidebarVisible: true, // 侧边栏
      ptzVisible: false, // 云台面板
      currentChannelId: null, // 当前选中通道
      spiltIndex: 2, // 分屏
      playerIdx: 0, // 激活播放器

      updateLooper: 0, // 数据刷新轮训标志
      count: 15,
      total: 0,

      // channel
      loading: false,
      layout: [
        {
          spilt: 1,
          columns: '1fr',
          rows: '1fr',
          style: function() {}
        },
        {
          spilt: 4,
          columns: '1fr 1fr',
          rows: '1fr 1fr',
          style: function() {}
        },
        {
          spilt: 6,
          columns: '1fr 1fr 1fr',
          rows: '1fr 1fr 1fr',
          style: function(index) {
            console.log(index)
            if (index === 0) {
              return {
                gridColumn: ' 1 / span 2',
                gridRow: ' 1 / span 2'
              }
            }
          }

        },
        {
          spilt: 9,
          columns: '1fr 1fr 1fr',
          rows: '1fr 1fr 1fr',
          style: function() {}
        }
      ]
    }
  },

  computed: {
    liveStyle() {
      return {
        display: 'grid',
        gridTemplateColumns: this.layout[this.spiltIndex].columns,
        gridTemplateRows: this.layout[this.spiltIndex].rows,
        gap: '10px',
        backgroundColor: 'transparent'
      }
    }
  },
  watch: {
    spiltIndex(newValue) {
      console.log('切换画幅;' + newValue)
      const that = this
      for (let i = 1; i <= this.layout[newValue].spilt; i++) {
        if (!that.$refs['playerTabs' + i]) {
          continue
        }
        this.$nextTick(() => {
          const ref = that.$refs['playerTabs' + i]
          const instance = ref instanceof Array ? ref[0] : ref
          if (instance && instance.resize) {
            instance.resize()
          }
        })
      }
      window.localStorage.setItem('split', newValue)
    },
    globalPlayer(newKey) {
      for (let i = 1; i <= this.layout[this.spiltIndex].spilt; i++) {
        const ref = this.$refs['playerTabs' + i]
        if (ref) {
          const instance = ref instanceof Array ? ref[0] : ref
          instance.switchPlayer(newKey)
        }
      }
      window.localStorage.setItem('globalPlayer', newKey)
    },
    '$route.fullPath': 'checkPlayByParam'
  },
  mounted() {
    // Add window resize event listener to handle responsive behavior
    window.addEventListener('resize', this.handleResize)
    this.handleResize()
  },
  created() {
    const savedPlayer = window.localStorage.getItem('globalPlayer')
    if (savedPlayer && ['jessibuca', 'webRTC', 'h265web'].includes(savedPlayer)) {
      this.globalPlayer = savedPlayer
    }
    const savedSplit = window.localStorage.getItem('split')
    if (savedSplit !== null && savedSplit !== '') {
      const n = parseInt(savedSplit, 10)
      if (!Number.isNaN(n) && n >= 0 && n < this.layout.length) {
        this.spiltIndex = n
      }
    }
    this.checkPlayByParam()
  },
  destroyed() {
    clearTimeout(this.updateLooper)
    this.clearAllPullCountdowns()
    // Remove event listener when component is destroyed
    window.removeEventListener('resize', this.handleResize)
  },
  methods: {
    startPullCountdown(idx) {
      this.stopPullCountdown(idx)
      this.$set(this.pullLoading, idx, true)
      this.$set(this.pullFading, idx, false)
      this.$set(this.pullCountdown, idx, 3)
      this.$set(this.pullCycle, idx, 0)
      if (!this._countdownTimers) this._countdownTimers = {}
      this._countdownTimers[idx] = setInterval(() => {
        if (!this.pullLoading[idx] || this.pullFading[idx]) return
        const cur = this.pullCountdown[idx]
        if (cur <= 1) {
          this.$set(this.pullCycle, idx, (this.pullCycle[idx] || 0) + 1)
          this.$set(this.pullCountdown, idx, 3)
        } else {
          this.$set(this.pullCountdown, idx, cur - 1)
        }
      }, 780)
    },
    stopPullCountdown(idx) {
      if (this._countdownTimers && this._countdownTimers[idx]) {
        clearInterval(this._countdownTimers[idx])
        this._countdownTimers[idx] = null
      }
      if (this._countdownWatchdogs && this._countdownWatchdogs[idx]) {
        clearTimeout(this._countdownWatchdogs[idx])
        this._countdownWatchdogs[idx] = null
      }
      if (this._fadeTimers && this._fadeTimers[idx]) {
        clearTimeout(this._fadeTimers[idx])
        this._fadeTimers[idx] = null
      }
      this.$set(this.pullLoading, idx, false)
      this.$set(this.pullFading, idx, false)
      this.$set(this.pullCountdown, idx, null)
    },
    /** 出画后淡出遮罩，避免瞬删露出播放器黑底一闪 */
    fadeOutPullCountdown(idx) {
      if (!this.pullLoading[idx] || this.pullFading[idx]) return
      if (this._countdownTimers && this._countdownTimers[idx]) {
        clearInterval(this._countdownTimers[idx])
        this._countdownTimers[idx] = null
      }
      if (this._countdownWatchdogs && this._countdownWatchdogs[idx]) {
        clearTimeout(this._countdownWatchdogs[idx])
        this._countdownWatchdogs[idx] = null
      }
      this.$set(this.pullFading, idx, true)
      if (!this._fadeTimers) this._fadeTimers = {}
      clearTimeout(this._fadeTimers[idx])
      this._fadeTimers[idx] = setTimeout(() => {
        this.stopPullCountdown(idx)
      }, 320)
    },
    isPulling(idx) {
      return !!this.pullLoading[idx]
    },
    onPlayerPlayStatus(idx, playing) {
      if (playing && this.pullLoading[idx]) {
        this.fadeOutPullCountdown(idx)
      }
    },
    clearAllPullCountdowns() {
      if (this._countdownTimers) {
        Object.keys(this._countdownTimers).forEach(k => {
          clearInterval(this._countdownTimers[k])
        })
      }
      if (this._countdownWatchdogs) {
        Object.keys(this._countdownWatchdogs).forEach(k => {
          clearTimeout(this._countdownWatchdogs[k])
        })
      }
      if (this._fadeTimers) {
        Object.keys(this._fadeTimers).forEach(k => {
          clearTimeout(this._fadeTimers[k])
        })
      }
      this._countdownTimers = {}
      this._countdownWatchdogs = {}
      this._fadeTimers = {}
      this.pullLoading = []
      this.pullCountdown = []
      this.pullCycle = []
      this.pullFading = []
    },
    toggleSidebar() {
      this.sidebarVisible = !this.sidebarVisible
      if (this.sidebarVisible) {
        this.ptzVisible = false
      }
    },
    handleDragZoom(direction) {
      const refName = 'playerTabs' + (this.playerIdx + 1)
      const ref = this.$refs[refName]
      if (!ref) return
      const instance = ref instanceof Array ? ref[0] : ref
      if (!instance || !instance.startDragZoom) return
      console.log('[live] handleDragZoom playerTabs:', refName, 'playerIdx:', this.playerIdx, 'direction:', direction)
      instance.startDragZoom((params) => {
        console.log('[live] dragZoom before channelId:', JSON.stringify(params))
        params.channelId = this.currentChannelId
        console.log('[live] dragZoom after channelId:', JSON.stringify(params))
        const action = direction === 'in' ? 'commonChanel/dragZoomIn' : 'commonChanel/dragZoomOut'
        const successMsg = direction === 'in' ? '拉框放大成功' : '拉框缩小成功'
        const failMsg = direction === 'in' ? '拉框放大失败' : '拉框缩小失败'
        this.$store.dispatch(action, params).then(() => {
          this.$message({ showClose: true, message: successMsg, type: 'success' })
        }).catch(() => {
          this.$message({ showClose: true, message: failMsg, type: 'error' })
        })
      })
    },
    togglePtzPanel() {
      this.ptzVisible = !this.ptzVisible
      if (this.ptzVisible) {
        this.sidebarVisible = false
      }
    },
    handleResize() {
      this.$forceUpdate()

      this.$nextTick(() => {
        for (let i = 0; i < this.layout[this.spiltIndex].spilt; i++) {
          const ref = this.$refs[`playerTabs${i + 1}`]
          if (ref) {
            const instance = ref instanceof Array ? ref[0] : ref
            instance.resize && instance.resize()
          }
        }
      })
    },
    clickEvent: function(channelId) {
      this.currentChannelId = channelId
      this.sendDevicePush(channelId)
    },
    getPlayerClass: function(splitIndex, i) {
      let classStr = 'play-box-' + splitIndex + '-' + i
      if (this.playerIdx === (i - 1)) {
        classStr += ' redborder'
      }
      return classStr
    },
    contextMenuEvent: function(device, event, data, isCatalog) {

    },
    // 通知设备上传媒体流
    sendDevicePush: function(channelId) {
      if (!channelId) return
      // 树节点双击会触发两次 click，防抖避免重复 openRtpServer / INVITE 打崩 ZMS
      const now = Date.now()
      if (this._playBusy || (this._lastPlayChannelId === channelId && now - (this._lastPlayAt || 0) < 800)) {
        console.warn('[live] ignore duplicate play', channelId)
        return
      }
      this._playBusy = true
      this._lastPlayChannelId = channelId
      this._lastPlayAt = now
      this.save(channelId)
      const idxTmp = this.playerIdx
      this.$set(this.streamInfo, idxTmp, null)
      this.$set(this.videoTip, idxTmp, '')
      this.startPullCountdown(idxTmp)
      this.$store.dispatch('commonChanel/playChannel', channelId)
        .then(data => {
          // 接口很快返回，真正出画才停倒计时；先挂播放器在底层拉流
          this.setPlayStream(data.transcodeStream || data, idxTmp)
          // 兜底：部分播放器不抛 play 事件时，最多等 12s 后仍停表，避免永远挡画面
          if (!this._countdownWatchdogs) this._countdownWatchdogs = {}
          clearTimeout(this._countdownWatchdogs[idxTmp])
          this._countdownWatchdogs[idxTmp] = setTimeout(() => {
            if (this.pullLoading[idxTmp] && this.streamInfo[idxTmp]) {
              this.fadeOutPullCountdown(idxTmp)
            }
          }, 12000)
        })
        .catch(err => {
          this.stopPullCountdown(idxTmp)
          this.$set(this.videoTip, idxTmp, '播放失败: ' + err)
        })
        .finally(() => {
          this.loading = false
          this._playBusy = false
        })
    },
    setPlayStream(streamInfo, idx) {
      this.$set(this.streamInfo, idx, streamInfo)
      // 等 PlayerTabs 挂载且格子有宽高后再播（Jessibuca 首播对 0 尺寸敏感）
      this.$nextTick(() => {
        this.$nextTick(() => {
          const refName = 'playerTabs' + (idx + 1)
          const ref = this.$refs[refName]
          if (!ref) return
          const instance = ref instanceof Array ? ref[0] : ref
          if (instance && instance.setStreamInfo) {
            instance.setStreamInfo(streamInfo)
          }
        })
      })
    },
    checkPlayByParam() {
      const query = this.$route.query
      if (query.channelId) {
        this.sendDevicePush(query.channelId)
      }
    },

    save(item) {
      const dataStr = window.localStorage.getItem('playData') || '[]'
      const data = JSON.parse(dataStr)
      data[this.playerIdx] = item
      window.localStorage.setItem('playData', JSON.stringify(data))
    },
    clear(idx) {
      const dataStr = window.localStorage.getItem('playData') || '[]'
      const data = JSON.parse(dataStr)
      data[idx - 1] = null
      console.log(data)
      window.localStorage.setItem('playData', JSON.stringify(data))
    },
    fullScreen: function() {
      if (screenFull.isEnabled) {
        screenFull.toggle(this.$refs.playBox)
      }
    }
  }
}
</script>
<style>
.live-container {
  height: calc(100vh - 124px);
  width: 100%;
}

.live-content {
  height: 100%;
  display: flex;
  flex-direction: row;
}

.device-tree-container-box {
  width: 406px;
  min-width: 250px;
  max-width: 400px;
  background-color: #ffffff;
  overflow: auto;
  resize: horizontal;
  transition: width 0.3s ease, min-width 0.3s ease;
}

.device-tree-hidden {
  width: 0 !important;
  min-width: 0 !important;
  overflow: hidden;
  resize: none;
}

@media (max-width: 768px) {
  .live-content {
    flex-direction: column;
  }

  .device-tree-container-box {
    width: 100%;
    max-width: 100%;
    height: 200px;
    min-height: 150px;
    max-height: 300px;
    resize: vertical;
  }
}

.video-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #f0f4f8;
}

.control-bar {
  height: 5vh;
  min-height: 44px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 15px;
  color: #334155;
  padding: 0 4px;
  background: #ffffff;
  border-bottom: 1px solid #e3ebf5;
}

.split-controls {
  text-align: left;
  padding-left: 10px;
}

.fullscreen-control {
  text-align: right;
  padding-right: 10px;
}

.ptz-toggle-control {
  text-align: right;
  padding-right: 10px;
}

.ptz-toggle-control .btn.active {
  color: #1565c0;
}

.ptz-panel {
  width: 406px;
  min-width: 340px;
  background-color: #ffffff;
  border-left: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.ptz-panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #e4e7ed;
  font-size: 15px;
  font-weight: bold;
}

.ptz-panel-header .el-icon-close {
  cursor: pointer;
  font-size: 18px;
  color: #909399;
}

.ptz-panel-header .el-icon-close:hover {
  color: #409EFF;
}

.ptz-panel-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 12px 16px;
}

.ptz-preset-section {
  flex: 1;
  overflow-y: auto;
  margin-bottom: 8px;
}

.ptz-control-section {
  flex-shrink: 0;
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 10px;
  padding-bottom: 6px;
  border-bottom: 1px solid #ebeef5;
}

.ptz-divider {
  height: 1px;
  background-color: #e4e7ed;
  margin: 12px 0;
}

.ptz-empty-tip {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #909399;
  font-size: 14px;
}

.global-player-control {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
}

.player-container {
  flex: 1;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 14px 16px 16px;
  overflow: hidden;
  background: #f0f4f8;
}

.play-grid {
  width: 100%;
  height: 100%;
  max-height: calc(100vh - 180px);
  aspect-ratio: 16/9;
  border: none;
  border-radius: 0;
  overflow: visible;
  box-shadow: none;
}

.btn {
  margin: 0 10px;
  cursor: pointer;
  color: #64748b;
  transition: color 0.15s ease;
}

.btn:hover {
  color: #1565c0;
}

.btn.active {
  color: #1565c0;
}

.sidebar-toggle {
  margin: 0 2px;
  font-size: 18px;
  vertical-align: middle;
}

.divider {
  display: inline-block;
  width: 1px;
  height: 16px;
  background-color: #d8e2ee;
  margin: 0 8px;
  vertical-align: middle;
}

.play-box {
  /* 独立圆角卡片 + 中性灰蓝，减少「黑窗贴浅蓝缝」的割裂感 */
  background: linear-gradient(165deg, #4a5568 0%, #3a4556 46%, #323c4b 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  position: relative;
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04), 0 8px 20px rgba(15, 23, 42, 0.06);
  cursor: pointer;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.play-box::before {
  content: '';
  position: absolute;
  inset: 0;
  background:
    radial-gradient(80% 60% at 50% 38%, rgba(255, 255, 255, 0.06) 0%, rgba(255, 255, 255, 0) 70%);
  pointer-events: none;
  z-index: 0;
}

/* 内侧选中环：不受 overflow:hidden 裁切，空窗/有画面都可见 */
.play-box.redborder {
  border-color: rgba(21, 101, 192, 0.55);
  z-index: 2;
  box-shadow:
    0 1px 2px rgba(15, 23, 42, 0.04),
    0 8px 20px rgba(21, 101, 192, 0.14);
}

.play-box.redborder::after {
  content: '';
  position: absolute;
  inset: 3px;
  border-radius: 9px;
  border: 2.5px solid #1565c0;
  box-shadow:
    0 0 0 1.5px rgba(255, 255, 255, 0.92),
    inset 0 0 0 1px rgba(255, 255, 255, 0.35);
  pointer-events: none;
  z-index: 12;
}

.no-signal {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  color: #b7c2d0;
  user-select: none;
  padding: 16px;
  text-align: center;
}

.no-signal-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  color: rgba(226, 232, 240, 0.72);
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.08);
  margin-bottom: 4px;
}

.no-signal-title {
  color: rgba(241, 245, 249, 0.88);
  font-size: 13px;
  font-weight: 560;
  letter-spacing: 0.06em;
}

.pull-countdown {
  position: absolute;
  inset: 0;
  z-index: 5;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;
  /* 与窗格灰蓝底同色系，略提亮半透，避免「黑罩」突兀 */
  background:
    linear-gradient(165deg, rgba(74, 85, 104, 0.72) 0%, rgba(58, 69, 86, 0.78) 50%, rgba(50, 60, 75, 0.82) 100%);
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
  border-radius: inherit;
  opacity: 1;
  transition: opacity 0.3s ease;
}

.pull-countdown.is-fading {
  opacity: 0;
}

.pull-countdown.is-fading .pull-countdown-num {
  animation: none;
  opacity: 0;
  transition: opacity 0.2s ease;
}

.pull-countdown-num {
  font-size: clamp(56px, 12vw, 92px);
  font-weight: 620;
  letter-spacing: -0.04em;
  color: #ffffff;
  text-shadow: 0 8px 28px rgba(15, 23, 42, 0.28);
  font-variant-numeric: tabular-nums;
  font-family: -apple-system, BlinkMacSystemFont, "SF Pro Display", "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
  animation: pull-count-pop 0.72s cubic-bezier(0.22, 1, 0.36, 1);
  will-change: transform, opacity;
}

@keyframes pull-count-pop {
  0% {
    opacity: 0;
    transform: scale(1.28);
    filter: blur(2px);
  }
  28% {
    opacity: 1;
    filter: blur(0);
  }
  100% {
    opacity: 0.92;
    transform: scale(1);
  }
}

@media (prefers-reduced-motion: reduce) {
  .pull-countdown-num {
    animation: none;
  }
}

.play-box-2-1 {
  grid-column: 1 / span 2;
  grid-row: 1 / span 2;
}

/* Responsive adjustments for smaller screens */
@media (max-width: 576px) {
  .control-bar {
    flex-direction: column;
    height: auto;
    padding: 5px 0;
  }

  .split-controls, .fullscreen-control {
    width: 100%;
    text-align: center;
    padding: 5px 0;
  }

  .btn {
    margin: 0 5px;
  }
}


</style>
