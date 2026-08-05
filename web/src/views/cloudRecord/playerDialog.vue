<template>
  <div id="playerDialog">
    <el-dialog
      v-el-drag-dialog
      custom-class="cloud-record-play-dialog"
      top="2rem"
      width="880px"
      :append-to-body="true"
      :modal-append-to-body="true"
      :modal="true"
      :show-close="false"
      :close-on-click-modal="false"
      :visible.sync="showDialog"
      :destroy-on-close="true"
      @opened="onDialogOpened"
      @close="close()"
    >
      <div slot="title" class="cloud-record-play-chrome">
        <div class="cloud-record-play-chrome-title">云端录像</div>
        <div class="cloud-record-play-chrome-actions">
          <button
            type="button"
            class="cloud-record-play-chrome-btn"
            :title="maximized ? '还原' : '最大化'"
            @click.stop="toggleMaximize"
          >
            <i :class="maximized ? 'el-icon-copy-document' : 'el-icon-full-screen'" />
          </button>
          <button
            type="button"
            class="cloud-record-play-chrome-btn is-close"
            title="关闭"
            @click.stop="close()"
          >
            <i class="el-icon-close" />
          </button>
        </div>
      </div>
      <div class="cloud-record-play-frame" :class="{ 'is-maximized': maximized }">
        <cloudRecordPlayer
          class="cloud-record-play-stage"
          ref="cloudRecordPlayer"
          :hide-player-switch="true"
          :hide-fullscreen="true"
        />
      </div>
    </el-dialog>
  </div>
</template>

<script>

import elDragDialog from '@/directive/el-drag-dialog'
import cloudRecordPlayer from './player.vue'

export default {
  name: 'PlayerDialog',
  components: { cloudRecordPlayer },
  directives: { elDragDialog },
  data() {
    return {
      showDialog: false,
      streamInfo: null,
      pendingPlay: null,
      maximized: false
    }
  },
  methods: {
    openDialog(streamInfo, timeLen, startTime) {
      this.maximized = false
      this.pendingPlay = { streamInfo, timeLen, startTime }
      this.streamInfo = streamInfo
      this.showDialog = true
      this.$nextTick(() => {
        this.resetDialogStyle()
        this.applyPendingPlay()
      })
    },
    onDialogOpened() {
      this.resetDialogStyle()
      this.applyPendingPlay()
    },
    applyPendingPlay(retries) {
      const pending = this.pendingPlay
      if (!pending) return
      const left = retries == null ? 20 : retries
      const player = this.$refs.cloudRecordPlayer
      if (!player) {
        if (left <= 0) return
        this.$nextTick(() => this.applyPendingPlay(left - 1))
        return
      }
      this.pendingPlay = null
      player.setStreamInfo(pending.streamInfo, pending.timeLen, pending.startTime)
    },
    stopPlay() {
      if (this.$refs.cloudRecordPlayer) {
        this.$refs.cloudRecordPlayer.stopPLay()
      }
    },
    close() {
      if (this.$refs.cloudRecordPlayer) {
        this.$refs.cloudRecordPlayer.stopPLay()
      }
      this.pendingPlay = null
      this.maximized = false
      this.showDialog = false
    },
    resetDialogStyle() {
      // append-to-body 后弹窗不在 this.$el 内
      const dialog = document.querySelector('.cloud-record-play-dialog')
      if (!dialog) return
      dialog.style.width = ''
      dialog.style.marginTop = ''
      dialog.style.top = ''
      dialog.style.left = ''
      dialog.style.maxWidth = ''
      dialog.style.height = ''
      dialog.style.borderRadius = ''
    },
    toggleMaximize() {
      this.maximized = !this.maximized
      this.$nextTick(() => {
        const dialog = document.querySelector('.cloud-record-play-dialog')
        if (!dialog) return
        if (this.maximized) {
          dialog.style.width = '100vw'
          dialog.style.marginTop = '0'
          dialog.style.top = '0'
          dialog.style.left = '0'
          dialog.style.maxWidth = '100vw'
          dialog.style.height = '100vh'
          dialog.style.borderRadius = '0'
        } else {
          this.resetDialogStyle()
        }
      })
    }
  }
}
</script>

<style>
.cloud-record-play-dialog {
  background: #1a1a1a !important;
  border: 1px solid #2f2f2f !important;
  border-radius: 8px !important;
  overflow: hidden;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.5) !important;
}

/* 顶栏即 Element header：与播放区无缝衔接，可拖拽 */
.cloud-record-play-dialog .el-dialog__header {
  padding: 0 !important;
  margin: 0 !important;
  border-bottom: none !important;
  background: #252525;
}

.cloud-record-play-dialog .el-dialog__body {
  padding: 0 !important;
  background: #1a1a1a;
  color: #fff;
}

.cloud-record-play-chrome {
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 6px 0 14px;
  background: #252525;
  border-bottom: 1px solid #333;
  user-select: none;
  cursor: move;
}

.cloud-record-play-chrome-title {
  font-size: 13px;
  font-weight: 600;
  color: #e8e8e8;
}

.cloud-record-play-chrome-actions {
  display: flex;
  align-items: center;
  gap: 2px;
}

.cloud-record-play-chrome-btn {
  width: 40px;
  height: 30px;
  margin: 0;
  padding: 0;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: #c8c8c8;
  font-size: 15px;
  line-height: 30px;
  cursor: pointer;
  outline: none;
}

.cloud-record-play-chrome-btn:hover {
  background: #3a3a3a;
  color: #fff;
}

.cloud-record-play-chrome-btn.is-close:hover {
  background: #e81123;
  color: #fff;
}

.cloud-record-play-frame {
  display: flex;
  flex-direction: column;
  background: #1a1a1a;
}

.cloud-record-play-frame.is-maximized {
  height: calc(100vh - 40px);
}

.cloud-record-play-stage {
  flex: 1;
  min-height: 0;
  height: auto !important;
}

.cloud-record-play-frame.is-maximized .cloud-record-play-stage {
  height: 100% !important;
  display: flex;
  flex-direction: column;
}

.cloud-record-play-frame.is-maximized .cloud-record-play-stage .cloud-record-player-root {
  height: 100%;
}

.cloud-record-play-frame.is-maximized .cloud-record-play-stage .cloud-record-playBox {
  flex: 1;
  aspect-ratio: auto;
  min-height: 0;
}
</style>
