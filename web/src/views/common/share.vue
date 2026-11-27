<template>
  <div style="width: 100vw; height: 100vh; background-color: #000; overflow: hidden;">
    <jessibuca-player
      ref="player"
      :show-button="true"
      style="width: 100%; height: 100%"
    />
  </div>
</template>

<script>
import jessibucaPlayer from './jessibuca.vue'

export default {
  name: 'SharePlayer',
  components: { jessibucaPlayer },
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
