<template>
  <div id="devices" class="app-container">
    <device-list
      v-show="!channelCtx"
      @show-channel="showChannel"
    />
    <!-- 通道列表对所有协议统一：不再按协议分支到不同组件，新增协议零改动 -->
    <unified-channel
      v-if="channelCtx"
      :device-internal-code="channelCtx.internalCode"
      :device-name="channelCtx.name"
      :protocol="channelCtx.protocol"
      embedded
      device-locked
      @back="showDevice"
    />
  </div>
</template>

<script>
import deviceList from './list.vue'
import unifiedChannel from '@/views/channel/unified-panel.vue'

export default {
  name: 'Devices',
  components: { deviceList, unifiedChannel },
  data() {
    return {
      channelCtx: null
    }
  },
  methods: {
    showChannel(row) {
      this.channelCtx = {
        protocol: row.protocol,
        internalCode: row.internalCode,
        name: row.name
      }
    },
    showDevice() {
      this.channelCtx = null
    }
  }
}
</script>
