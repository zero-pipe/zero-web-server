<template>
  <div id="devices" class="app-container">
    <device-list
      v-show="!channelCtx"
      @show-channel="showChannel"
    />
    <gb-channel
      v-if="channelCtx && channelCtx.protocol === 'gb28181'"
      :device-id="channelCtx.rawId"
      @show-device="showDevice"
    />
    <onvif-channel
      v-if="channelCtx && channelCtx.protocol === 'onvif'"
      :device-id="Number(channelCtx.rawId)"
      @show-device="showDevice"
    />
  </div>
</template>

<script>
import deviceList from './list.vue'
import gbChannel from '@/views/device/channel/index.vue'
import onvifChannel from '@/views/onvifDevice/channel/index.vue'

export default {
  name: 'Devices',
  components: { deviceList, gbChannel, onvifChannel },
  data() {
    return {
      channelCtx: null
    }
  },
  methods: {
    showChannel(row) {
      this.channelCtx = {
        protocol: row.protocol,
        rawId: row.rawId,
        id: row.id
      }
    },
    showDevice() {
      this.channelCtx = null
    }
  }
}
</script>
