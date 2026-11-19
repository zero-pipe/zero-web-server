<template>
  <span class="player-mode-select">
    <span v-if="showLabel" class="pms-label">{{ label }}</span>
    <el-select
      :value="mode"
      size="mini"
      class="pms-select"
      @change="onChange"
    >
      <el-option
        v-for="opt in options"
        :key="opt.key"
        :label="opt.label"
        :value="opt.key"
      />
    </el-select>
  </span>
</template>

<script>
import { PLAYER_MODE_OPTIONS } from '@/utils/playerStrategy'

/**
 * 全局播放模式选择器：所有播放页面共用同一份全局状态（store + localStorage），
 * 文案只暴露「自动 / 标准 / 超低延迟」，不出现引擎名称，便于统一维护。
 */
export default {
  name: 'PlayerModeSelect',
  props: {
    showLabel: { type: Boolean, default: true },
    label: { type: String, default: '播放器:' }
  },
  data() {
    return {
      options: PLAYER_MODE_OPTIONS
    }
  },
  computed: {
    mode() {
      return this.$store.getters.playerMode || 'auto'
    }
  },
  methods: {
    onChange(mode) {
      this.$store.dispatch('setPlayerMode', mode).then(() => {
        this.$emit('change', mode)
      })
    }
  }
}
</script>

<style scoped>
.player-mode-select {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.pms-label {
  white-space: nowrap;
}
.pms-select {
  width: 120px;
}
</style>
