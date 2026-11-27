<template>
  <div id="ptzPreset" style="width: 100%">
    <el-tag
      v-for="item in presetList"
      :key="item.presetId"
      size="mini"
      style="margin-right: 1rem; cursor: pointer; margin-bottom: 0.6rem"
      @click="gotoPreset(item)"
    >
      {{ item.presetName || item.presetId }}
    </el-tag>
  </div>
</template>

<script>
import { resourcePresetQuery, resourcePresetCall } from '@/api/asset'

export default {
  name: 'PtzPreset',
  props: {
    // 通道资源 id（=通道表 id=树通道叶 id）
    resourceId: { type: String, default: null }
  },
  data() {
    return {
      presetList: []
    }
  },
  created() {
    this.getPresetList()
  },
  methods: {
    getPresetList: function() {
      if (!this.resourceId) return
      resourcePresetQuery(this.resourceId)
        .then(res => {
          this.presetList = (res && res.data) || []
        })
    },
    gotoPreset: function(preset) {
      resourcePresetCall(this.resourceId, preset.presetId)
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
