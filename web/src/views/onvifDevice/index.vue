<template>
  <div id="onvif-device" class="app-container">
    <deviceList v-show="deviceId === null" @show-channel="showChannelList" />
    <channelList v-if="deviceId !== null" :device-id="deviceId" @show-device="showDevice" />
  </div>
</template>

<script>
import deviceList from './list.vue'
import channelList from './channel/index.vue'

export default {
  name: 'OnvifDevice',
  components: {
    deviceList,
    channelList
  },
  data() {
    return {
      deviceId: null
    }
  },
  methods: {
    showChannelList(deviceId) {
      this.deviceId = deviceId
    },
    showDevice() {
      this.deviceId = null
    }
  }
}
</script>
