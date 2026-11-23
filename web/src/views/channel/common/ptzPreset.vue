<template>
  <div id="ptzPreset" style="width: 100%">
    <template v-if="presetList.length">
      <el-tag
        v-for="item in presetList"
        :key="item.presetId"
        size="mini"
        style="margin-right: 1rem; cursor: pointer; margin-bottom: 0.6rem"
        @click="gotoPreset(item)"
      >
        {{ item.presetName || item.presetId }}
      </el-tag>
    </template>
    <div v-else-if="loaded && !loadError" class="preset-hint">暂无预置位</div>
    <div v-else-if="loadError" class="preset-hint">设备未返回预置位（可能不支持预置位）</div>
  </div>
</template>

<script>
export default {
  name: 'ChPtzPreset',
  props: {
    channelId: { type: String, default: null }
  },
  data() {
    return {
      presetList: [],
      loaded: false,
      loadError: false
    }
  },
  created() {
    this.getPresetList()
  },
  methods: {
    getPresetList() {
      this.loaded = false
      this.loadError = false
      this.$store.dispatch('commonChanel/queryPreset', this.channelId)
        .then(data => {
          this.presetList = data || []
          this.loaded = true
        })
        .catch(() => {
          // 查询失败（枪机/设备不应答）降级为内联提示，不弹全局错误
          this.loadError = true
          this.loaded = true
        })
    },
    gotoPreset(preset) {
      this.$store.dispatch('commonChanel/callPreset', { channelId: this.channelId, presetId: preset.presetId })
        .then(() => {
          this.$message({
            showClose: true,
            message: '调用成功',
            type: 'success'
          })
        }).catch((error) => {
          this.$message({
            showClose: true,
            message: error,
            type: 'error'
          })
        })
    }
  }
}
</script>

<style scoped>
.preset-hint {
  padding: 6px 4px;
  color: #80868b;
  font-size: 13px;
}
</style>
