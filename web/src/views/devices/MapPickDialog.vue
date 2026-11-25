<template>
  <el-dialog
    v-el-drag-dialog
    title="地图选点"
    width="70%"
    top="5vh"
    :close-on-click-modal="false"
    :visible.sync="showDialog"
    :destroy-on-close="true"
    append-to-body
    @opened="onOpened"
    @closed="onClosed"
  >
    <div style="position: relative; height: 62vh; border: 1px solid #ebeef5;">
      <MapComponent ref="map" @loaded="onMapLoaded" />
      <div class="pick-tip">在地图上点击选择位置（坐标按 WGS84 回填）</div>
    </div>
    <div style="margin-top: 10px; font-size: 13px; color: #606266;">
      当前坐标：
      <template v-if="picked">
        经度 <b>{{ picked[0].toFixed(6) }}</b> · 纬度 <b>{{ picked[1].toFixed(6) }}</b>
      </template>
      <span v-else>尚未选择</span>
    </div>
    <div slot="footer">
      <el-button size="small" @click="showDialog = false">取消</el-button>
      <el-button size="small" type="primary" :disabled="!picked" @click="confirm">确定</el-button>
    </div>
  </el-dialog>
</template>

<script>
import MapComponent from '@/views/common/MapComponent.vue'

export default {
  name: 'MapPickDialog',
  components: { MapComponent },
  data() {
    return {
      showDialog: false,
      picked: null,
      center: null,
      callback: null,
      mapReady: false
    }
  },
  methods: {
    // center: 可选 [lon, lat] 初始定位（WGS84）
    openDialog(center, callback) {
      this.center = Array.isArray(center) && center.length === 2 && center[0] && center[1] ? center : null
      this.callback = typeof callback === 'function' ? callback : null
      this.picked = null
      this.mapReady = false
      this.showDialog = true
    },
    onOpened() {
      // 组件在 destroy-on-close 下每次打开都会重建，等其 MapComponent 初始化完成
    },
    onMapLoaded() {
      this.mapReady = true
      this.startPick()
    },
    startPick() {
      const map = this.$refs.map
      if (!map) return
      if (this.center) {
        map.panTo(this.center, 16)
      }
      map.pickPoint(lonlat => {
        this.picked = lonlat
      })
    },
    onClosed() {
      const map = this.$refs.map
      if (map && map.cancelPick) map.cancelPick()
      this.picked = null
      this.callback = null
    },
    confirm() {
      if (this.picked && this.callback) {
        this.callback(this.picked.slice())
      }
      this.showDialog = false
    }
  }
}
</script>

<style scoped>
.pick-tip {
  position: absolute;
  left: 12px;
  top: 12px;
  z-index: 10;
  background: rgba(0, 0, 0, 0.6);
  color: #fff;
  font-size: 12px;
  padding: 4px 10px;
  border-radius: 4px;
}
</style>
