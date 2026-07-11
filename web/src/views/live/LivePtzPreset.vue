<template>
  <div class="live-ptz-preset" v-loading="loading">
    <div v-for="item in presetList" :key="item.presetId" class="preset-item" @click="gotoPreset(item)">
      <span class="preset-idx">{{ item.presetId }}</span>
      <span class="preset-name">{{ item.presetName || '预置位 ' + item.presetId }}</span>
    </div>
    <div v-if="!loading && !presetList.length" class="preset-empty">{{ emptyTip }}</div>
  </div>
</template>

<script>
export default {
  name: 'LivePtzPreset',
  props: {
    channelId: { type: String, default: null }
  },
  data() {
    return {
      presetList: [],
      loading: false,
      emptyTip: '暂无预置位'
    }
  },
  watch: {
    channelId: {
      immediate: true,
      handler(val) {
        if (val) {
          this.getPresetList()
        } else {
          this.presetList = []
        }
      }
    }
  },
  methods: {
    getPresetList() {
      if (!this.channelId) return
      this.loading = true
      this.emptyTip = '暂无预置位'
      this.$store.dispatch('commonChanel/queryPreset', this.channelId)
        .then(data => {
          this.presetList = Array.isArray(data) ? data : []
        })
        .catch(err => {
          this.presetList = []
          this.emptyTip = typeof err === 'string' ? err : ((err && (err.msg || err.message)) || '预置位查询失败')
        })
        .finally(() => {
          this.loading = false
        })
    },
    gotoPreset(preset) {
      this.$store.dispatch('commonChanel/callPreset', {
        channelId: this.channelId,
        presetId: preset.presetId
      }).then(() => {
        this.$message({ showClose: true, message: '已调用预置位 ' + preset.presetId, type: 'success' })
      }).catch(() => {
        this.$message({ showClose: true, message: '调用预置位失败', type: 'error' })
      })
    }
  }
}
</script>

<style scoped>
.live-ptz-preset {
  width: 100%;
  min-height: 80px;
}
.preset-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  cursor: pointer;
  border-radius: 4px;
  border-bottom: 1px solid #f0f0f0;
  transition: background 0.15s;
}
.preset-item:hover {
  background: #ecf5ff;
}
.preset-item:active {
  background: #d9ecff;
}
.preset-idx {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: #e6f7ff;
  color: #1890ff;
  font-size: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.preset-name {
  font-size: 13px;
  color: #303133;
}
.preset-empty {
  text-align: center;
  color: #909399;
  font-size: 13px;
  padding: 20px 0;
}
</style>
