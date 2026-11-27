<template>
  <div style="width: 100vw; height: 100vh; background-color: #000; overflow: hidden;">
    <mpegts-player
      v-if="playerType === 0"
      ref="player"
      :show-button="true"
      style="width: 100%; height: 100%"
    />
    <rtc-player
      v-if="playerType === 1"
      ref="player"
      :show-controls="true"
      style="width: 100%; height: 100%"
    />
  </div>
</template>

<script>
import mpegtsPlayer from './mpegts.vue'
import rtcPlayer from './rtcPlayer.vue'

export default {
  name: 'SharePlayer',
  components: { mpegtsPlayer, rtcPlayer },
  data() {
    return {
      playerType: 0
    }
  },
  created() {
    const type = parseInt(this.$route.query.type)
    if (!isNaN(type) && type >= 0 && type <= 1) {
      this.playerType = type
    }
  },
  mounted() {
    const url = this.$route.query.url
    if (url) {
      this.$nextTick(() => {
        if (this.$refs.player) {
          this.$refs.player.play(decodeURIComponent(url))
        }
      })
    }
  }
}
</script>
