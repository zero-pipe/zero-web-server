<template>
  <div class="ptz-section">
    <ptzControls
      btn-layout="row"
      :caps="caps"
      @ptz-move="onPtzMove"
      @ptz-stop="onPtzStop"
      @focus-move="onFocusMove"
      @focus-stop="onFocusStop"
      @iris-move="onIrisMove"
      @iris-stop="onIrisStop"
      @toggle-drag-zoom="$emit('drag-zoom-start', 'in')"
      @toggle-drag-zoom-out="$emit('drag-zoom-start', 'out')"
    />
  </div>
</template>

<script>
import ptzControls from '../../common/ptzControls.vue'
import { FULL_PTZ_CAPS } from '@/utils/playerStrategy'
import { resourcePtz, resourceCommand } from '@/api/asset'

export default {
  name: 'DevicePtzPanel',
  components: { ptzControls },
  props: {
    // 通道资源 id（=通道表 id=树通道叶 id），统一云台/镜头族的路径主键
    resourceId: { type: String, default: null },
    // 云台/镜头能力集（由父级按通道事实推导后传入；缺省=不过滤全显示）
    caps: { type: Object, default: () => ({ ...FULL_PTZ_CAPS }) }
  },
  methods: {
    // 面板速度量为 1~100，统一云台族口径为 水平/垂直 0~255、变倍 0~16。
    moveSpeed(speed) {
      return parseInt(speed * 255 / 100)
    },
    zoomSpeed(speed) {
      return parseInt(speed * 16 / 100)
    },
    onPtzMove(e) {
      if (!this.resourceId) return
      resourcePtz(this.resourceId, {
        command: e.direction,
        horizonSpeed: this.moveSpeed(e.speed),
        verticalSpeed: this.moveSpeed(e.speed),
        zoomSpeed: this.zoomSpeed(e.speed)
      })
    },
    onPtzStop() {
      if (!this.resourceId) return
      resourcePtz(this.resourceId, { command: 'stop', horizonSpeed: 0, verticalSpeed: 0, zoomSpeed: 0 })
    },
    onFocusMove(e) {
      if (!this.resourceId) return
      resourceCommand(this.resourceId, 'focus', { command: e.command, speed: this.moveSpeed(e.speed) })
    },
    onFocusStop() {
      if (!this.resourceId) return
      resourceCommand(this.resourceId, 'focus', { command: 'stop', speed: 0 })
    },
    onIrisMove(e) {
      if (!this.resourceId) return
      resourceCommand(this.resourceId, 'iris', { command: e.command, speed: this.moveSpeed(e.speed) })
    },
    onIrisStop() {
      if (!this.resourceId) return
      resourceCommand(this.resourceId, 'iris', { command: 'stop', speed: 0 })
    }
  }
}
</script>

<style scoped>
.ptz-section {
  flex-shrink: 0;
  margin-bottom: 8px;
}
</style>
