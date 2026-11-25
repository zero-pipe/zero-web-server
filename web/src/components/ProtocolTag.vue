<template>
  <span class="proto-text"><i class="proto-dot" :class="dotClass" />{{ label }}</span>
</template>

<script>
// 协议列统一表现：彩色圆点 + 文字（GB28181 蓝 / ONVIF 紫 / SDK 橙）。
// 协议值中 plugin 对外统一显示为 SDK。
export default {
  name: 'ProtocolTag',
  props: {
    protocol: { type: String, default: '' }
  },
  computed: {
    label() {
      return { gb28181: 'GB28181', onvif: 'ONVIF', plugin: 'SDK' }[this.protocol] || (this.protocol || '—')
    },
    dotClass() {
      return this.protocol === 'gb28181' ? 'gb' : (this.protocol === 'onvif' ? 'onvif' : 'sdk')
    }
  }
}
</script>

<style scoped>
.proto-text {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.proto-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex: 0 0 auto;
}
.proto-dot.gb { background: #1A73E8; }
.proto-dot.onvif { background: #7c3aed; }
.proto-dot.sdk { background: #e6a23c; }
</style>
