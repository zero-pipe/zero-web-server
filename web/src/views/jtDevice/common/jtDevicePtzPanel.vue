<template>
  <div class="ptz-section">
    <ptzControls
      btn-layout="row"
      :show-diagonals="false"
      :caps="caps"
      @ptz-move="onPtzMove"
      @ptz-stop="onPtzStop"
      @focus-move="onFocusMove"
      @focus-stop="onFocusStop"
      @iris-move="onIrisMove"
      @iris-stop="onIrisStop"
    />
  </div>
</template>

<script>
import ptzControls from '../../common/ptzControls.vue'
import { FULL_PTZ_CAPS } from '@/utils/playerStrategy'

export default {
  name: 'JtDevicePtzPanel',
  components: { ptzControls },
  props: {
    deviceId: { type: String, default: null },
    channelId: { type: String, default: null },
    // 云台/镜头能力集（由父级按通道事实推导后传入；缺省=不过滤全显示）
    caps: { type: Object, default: () => ({ ...FULL_PTZ_CAPS }) }
  },
  methods: {
    onPtzMove(e) {
      this.$store.dispatch('jtDevice/ptz', {
        phoneNumber: this.deviceId,
        channelId: this.channelId,
        command: e.direction,
        speed: e.speed
      })
    },
    onPtzStop() {
      this.$store.dispatch('jtDevice/ptz', {
        phoneNumber: this.deviceId,
        channelId: this.channelId,
        command: 'stop',
        speed: 0
      })
    },
    onFocusMove(e) {
      const command = e.command === 'near' ? 'focusnear' : 'focusfar'
      this.$store.dispatch('jtDevice/ptz', {
        phoneNumber: this.deviceId,
        channelId: this.channelId,
        command: command,
        speed: e.speed
      })
    },
    onFocusStop() {
      this.$store.dispatch('jtDevice/ptz', {
        phoneNumber: this.deviceId,
        channelId: this.channelId,
        command: 'stop',
        speed: 0
      })
    },
    onIrisMove(e) {
      const command = e.command === 'in' ? 'irisin' : 'irisout'
      this.$store.dispatch('jtDevice/ptz', {
        phoneNumber: this.deviceId,
        channelId: this.channelId,
        command: command,
        speed: e.speed
      })
    },
    onIrisStop() {
      this.$store.dispatch('jtDevice/ptz', {
        phoneNumber: this.deviceId,
        channelId: this.channelId,
        command: 'stop',
        speed: 0
      })
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
