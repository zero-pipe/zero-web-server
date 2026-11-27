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
import { resourcePresetQuery, resourcePresetCall } from '@/api/asset'

export default {
  name: 'LivePtzPreset',
  props: {
    // 通道资源 id（=通道表 id=树通道叶 id；实时页 currentChannelId 即它）
    resourceId: { type: String, default: null }
  },
  data() {
    return {
      presetList: [],
      loading: false,
      emptyTip: '暂无预置位'
    }
  },
  watch: {
    resourceId: {
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
      if (!this.resourceId) return
      this.loading = true
      this.emptyTip = '暂无预置位'
      resourcePresetQuery(this.resourceId)
        .then(res => {
          this.presetList = Array.isArray(res && res.data) ? res.data : []
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
      resourcePresetCall(this.resourceId, preset.presetId).then(() => {
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
  background: #E8F0FE;
}
.preset-item:active {
  background: #D2E3FC;
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
  color: #80868B;
  font-size: 13px;
  padding: 20px 0;
}
</style>
