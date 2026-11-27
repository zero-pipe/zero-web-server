<template>
    <div id="devicePosition" style="height: calc(100vh - 84px);width: 100%;">
      <div style="height: 100%; display: grid; grid-template-columns: 360px auto">
        <DeviceTree ref="deviceTree" @clickEvent="treeChannelClickEvent" :showPosition="true" :contextmenu="getContextmenu()"/>
        <MapComponent ref="mapComponent" @loaded="initChannelLayer" @coordinateSystemChange="initChannelLayer" @zoomChange="zoomChange"></MapComponent>
      </div>
      <div class="map-tool-box-bottom-right">
        <div class="map-tool-btn-group" v-if="mapTileList.length > 0">
          <el-dropdown placement="top"  @command="changeLayerType">
            <div class="el-dropdown-link map-tool-btn">
              <i class="iconfont icon-mti-jutai"></i>
            </div>
            <el-dropdown-menu slot="dropdown">
              <el-dropdown-item :command="0" >
                <span v-if="layerType !== 0">图层关闭</span>
                <span v-if="layerType === 0" style="color: rgb(26, 115, 232);">图层关闭</span>
              </el-dropdown-item>
              <el-dropdown-item :command="1" >
                <span v-if="layerType !== 1">直接展示</span>
                <span v-if="layerType === 1" style="color: rgb(26, 115, 232);">直接展示</span>
              </el-dropdown-item>
              <el-dropdown-item :command="2">
                <span v-if="layerType !== 2">抽稀图层</span>
                <span v-if="layerType === 2" style="color: rgb(26, 115, 232);">抽稀图层</span>
              </el-dropdown-item>
            </el-dropdown-menu>
          </el-dropdown>
        </div>
        <div class="map-tool-btn-group" v-if="mapTileList && mapTileList.length > 1">
          <el-dropdown placement="top"  @command="changeMapTile">
            <div class="el-dropdown-link map-tool-btn">
              <i class="iconfont icon-tuceng"></i>
            </div>
            <el-dropdown-menu slot="dropdown">
              <el-dropdown-item  v-for="(item,index) in mapTileList" :key="index" :command="index">{{item.name}}</el-dropdown-item>
            </el-dropdown-menu>
          </el-dropdown>
        </div>
        <div class="map-tool-btn-group">
          <div class="map-tool-btn" @click="refreshLayer">
            <i class="iconfont icon-shuaxin3"></i>
          </div>
        </div>
        <div class="map-tool-btn-group">
          <div class="map-tool-btn" @click="zoomIn">
            <i class="iconfont icon-plus1"></i>
          </div>
          <div class="map-tool-btn" @click="zoomOut">
            <i class="iconfont icon-minus1"></i>
          </div>
        </div>
      </div>
      <div class="map-tool-box-top-left">
        <div class="map-tool-btn-group">
          <div class="map-tool-btn" title="图层抽稀" @click="showDrawThinBox(true)">
            <i class="iconfont icon-mti-sandian"></i> <span>图层抽稀</span>
          </div>
        </div>
      </div>
      <transition name="el-zoom-in-top">
        <div v-show="showDrawThin"  class="map-tool-draw-thin">
          <div class="map-tool-draw-thin-density">
            <span style="line-height: 36px; font-size: 15px">间隔： </span>
            <el-slider v-model="diffPixels" show-input :min="1" :max="200" input-size="mini" ></el-slider>
            <div style="margin-left: 10px; line-height: 38px;">
              <el-button :loading="quicklyDrawThinLoading" @click="quicklyDrawThin" size="mini">快速抽稀</el-button>
              <el-button :loading="boxDrawThinLoading" size="mini" @click="boxDrawThin" >局部抽稀</el-button>
              <el-button size="mini" @click="resetDrawThinData()">数据还原</el-button>
              <el-button :loading="saveDrawThinLoading" type="primary" :disabled="drawThinId === null" size="mini" @click="saveDrawThin()">保存</el-button>
              <el-button type="warning" size="mini" @click="showDrawThinBox(false)">取消</el-button>
            </div>
          </div>
        </div>
      </transition>

<!--      <div class="map-tool-box-top-right">-->
<!--        <div class="map-tool-btn-group">-->
<!--          <div class="map-tool-btn" title="抽稀">-->
<!--            <i class="iconfont icon-mti-sandian"></i>-->
<!--          </div>-->
<!--          <div class="map-tool-btn" title="聚合">-->
<!--            <i class="iconfont icon-mti-jutai"></i>-->
<!--          </div>-->

<!--        </div>-->
<!--      </div>-->
      <div ref="deviceInfoBox">
        <div class="popup-light" v-if="deviceInfo">
          <div class="popup-header">
            <div class="popup-title-wrap">
              <div class="popup-title">{{ deviceInfo.name }}</div>
              <div class="popup-meta">
                <span class="status-dot" :class="{ off: !deviceOnline }">{{ deviceOnline ? '在线' : '离线' }}</span>
                <span>·</span>
                <span>{{ protoText(deviceInfo.protocol) }}</span>
              </div>
            </div>
            <button class="close-icon" title="关闭" @click="closeInfoBox">×</button>
          </div>

          <div class="kv-list">
            <div class="kv-row">
              <span class="kv-label">编号</span>
              <span class="kv-val mono" :class="{ empty: !deviceInfo.code }">{{ deviceInfo.code || '-' }}</span>
            </div>
            <div class="kv-row">
              <span class="kv-label">厂商</span>
              <span class="kv-val" :class="{ empty: !deviceInfo.vendor }">{{ deviceInfo.vendor || '-' }}</span>
            </div>
            <div class="kv-row">
              <span class="kv-label">位置</span>
              <span class="kv-val" :class="{ empty: !deviceInfo.address }">{{ deviceInfo.address || '未配置' }}</span>
            </div>
          </div>

          <div class="chan-chips" v-if="deviceInfo.channels.length > 1">
            <button
              v-for="ch in deviceInfo.channels"
              :key="ch.gbId"
              class="chan-chip"
              :class="{ active: playingChannelId === ch.gbId }"
              :disabled="ch.gbStatus !== 'ON'"
              @click="playChannelOfDevice(ch)"
            >{{ ch.gbName }}</button>
          </div>

          <div class="popup-actions">
            <button class="act-btn act-primary" :disabled="!deviceOnline" @click="playFirstChannel">▶ 播放</button>
            <button class="act-btn act-secondary" @click="editDevice">✎ 编辑</button>
            <button class="act-btn act-secondary" @click="locateDevice">◎ 定位</button>
          </div>
        </div>
      </div>

      <player ref="player" @closed="onPlayerClosed"></player>
      <queryTrace ref="queryTrace" ></queryTrace>
      <DeviceEditDrawer :visible.sync="deviceEditVisible" :row="deviceEditRow" @success="initChannelLayer" />
      <DrawThinProgress ref="drawThinProgress" ></DrawThinProgress>
    </div>
</template>

<script>
import DeviceTree from '../common/DeviceTree.vue'
import queryTrace from './queryTrace.vue'
import MapComponent from '../common/MapComponent.vue'
import player from '../channel/player.vue'
import DeviceEditDrawer from '@/views/devices/DeviceEditDrawer.vue'
import DrawThinProgress from './dialog/drawThinProgress.vue'
import { queryChannels, queryAssets, getAssetDetail, startResourceLive, stopResourceLive } from '@/api/asset'

let channelLayer, channelTileLayer = null
export default {
  name: 'Map',
  components: {
    DrawThinProgress,
    DeviceEditDrawer,
    DeviceTree,
    player,
    queryTrace,
    MapComponent
  },
  data() {
    return {
      deviceInfo: null,
      deviceInfoBoxId: null,
      deviceEditVisible: false,
      deviceEditRow: null,
      deviceMetaMap: {},
      deviceGroups: {},
      playingChannelId: null,
      playLoading: false,
      playSeq: 0,
      closeStopTimer: null,
      isLoging: false,
      longitudeStr: 'longitude',
      latitudeStr: 'latitude',
      mapTileList: [],
      diffPixels: 120,
      zoomValue: 10,
      showDrawThin: false,
      quicklyDrawThinLoading: false,
      boxDrawThinLoading: false,
      drawThinId: null,
      drawThinLayer: null,
      saveDrawThinLoading: false,
      layerType: 0
    }
  },
  created() {

  },
  destroyed() {
    this.cancelCloseStop()
    // 离开地图时停掉保留的热流，避免后台一直拉设备。
    if (this.playingChannelId != null) {
      stopResourceLive(this.playingChannelId).catch(() => {})
      this.playingChannelId = null
    }
  },
  computed: {
    deviceOnline() {
      return !!(this.deviceInfo && this.deviceInfo.channels.some(c => (c.gbStatus || c.status) === 'ON'))
    }
  },
  methods: {
    initChannelLayer: function () {
      this.mapTileList = this.$refs.mapComponent.mapTileList
      this.closeInfoBox()

      if (channelLayer) {
        this.$refs.mapComponent.removeLayer(channelLayer)
        channelLayer = null
      }

      this.loadDeviceMeta()

      queryChannels({ page: 1, count: 5000 }).then(res => {
        // 只显示已纳入目录（行政区划 civilCode 或业务分组 parentId）的通道：
        // 仅仅配置了经纬度、未加入任何目录的设备不上图。
        const raw = (res.data && res.data.list) || []
        const list = raw.filter(c => {
          const gb = c.gb || {}
          return !!String(gb.civilCode || '').trim() || !!String(gb.parentId || '').trim()
        })
        // 一个设备一个图标：按 deviceId 归并同设备通道；通道统一成 {gbId, gbName, gbStatus, channelType}。
        const groups = {}
        list.forEach(c => {
          const key = c.deviceId || ('ch' + c.resourceId)
          if (!groups[key]) groups[key] = { key, name: c.deviceName || '', channels: [], position: null }
          groups[key].channels.push({
            gbId: c.resourceId,
            gbName: c.name,
            gbStatus: c.status || (c.online ? 'ON' : 'OFF'),
            channelType: c.channelType || '',
            protocol: c.protocol || '',
            gbCode: c.gbCode || '',
            externalKey: c.externalKey || '',
            ptzType: c.ptzType,
            hasAudio: c.hasAudio
          })
          if (!groups[key].position && c.longitude > 0 && c.latitude > 0) {
            groups[key].position = [c.longitude, c.latitude]
          }
        })
        this.deviceGroups = groups
        const points = []
        Object.keys(groups).forEach(k => {
          const g = groups[k]
          if (!g.position) return
          const meta = this.deviceMetaMap[k] || {}
          const online = g.channels.some(c => c.gbStatus === 'ON')
          const form = this.formIconBase(meta.deviceType)
          points.push({
            id: 'dev-' + k,
            position: g.position,
            status: online ? 'ON' : 'OFF',
            image: { src: this.deviceIconSrc(form, online), anchor: [0.5, 1] },
            data: {
              deviceId: k,
              name: meta.name || g.name || this.guessDeviceName(g.channels),
              deviceType: meta.deviceType || '',
              channels: g.channels,
              position: g.position
            }
          })
        })
        channelLayer = this.$refs.mapComponent.addPointLayerImages(points, this.deviceClickEvent, null)
        if (points.length === 1) {
          this.$refs.mapComponent.panTo(points[0].position, window.mapParam.zoom || 15)
        } else if (points.length > 1) {
          const lons = points.map(p => p.position[0])
          const lats = points.map(p => p.position[1])
          const mid = [
            (Math.min(...lons) + Math.max(...lons)) / 2,
            (Math.min(...lats) + Math.max(...lats)) / 2
          ]
          this.$refs.mapComponent.panTo(mid, window.mapParam.zoom || 15)
        }
      }).catch(() => {
        channelLayer = this.$refs.mapComponent.addPointLayerImages([], this.deviceClickEvent, null)
      })
    },
    // 设备元数据（名称 + 设备类型）→ { 设备dbid: {name, deviceType} }，用于图标形态与标题。
    loadDeviceMeta() {
      queryAssets({ page: 1, count: 1000 }).then(res => {
        const map = {}
        ;((res.data && res.data.list) || []).forEach(a => {
          if (a.device) map[a.device.id] = { name: a.device.name, deviceType: a.device.deviceType || '' }
        })
        this.deviceMetaMap = map
      }).catch(() => {})
    },
    // 设备类型 → 图标底图：枪机→camera3，球机→camera2，半球→camera1，其余→camera（通用）。
    formIconBase(deviceType) {
      const t = (deviceType || '').trim()
      if (t === 'bullet' || t === 'remote_ptz') return 'camera3'
      if (t === 'ptz') return 'camera2'
      if (t === 'dome' || t === 'remote_dome') return 'camera1'
      return 'camera'
    },
    deviceIconSrc(base, online) {
      return `static/images/gis/${base}${online ? '' : '-offline'}.png`
    },
    // 兜底设备名：通道名去掉尾部 _序号（ipc_onvif_01 → ipc_onvif）。
    guessDeviceName(channels) {
      const n = (channels && channels[0] && channels[0].gbName) || ''
      return n.replace(/_\d+$/, '') || n || '设备'
    },
    // 通道构成标签：双光/热成像/可见光/测温 + 单/双/多通道。
    compositionTags(channels) {
      const types = new Set((channels || []).map(c => c.channelType).filter(Boolean))
      const hasVisible = types.has('visible')
      const hasThermal = types.has('thermal') || types.has('infrared')
      const tags = []
      if (hasVisible && hasThermal) tags.push('双光(可见光+热成像)')
      else if (hasThermal) tags.push(types.has('thermal') ? '热成像' : '红外')
      else if (hasVisible) tags.push('可见光')
      if (types.has('temperature')) tags.push('测温')
      if (tags.length === 0) tags.push('未知类型')
      const n = (channels || []).length
      tags.push(n <= 1 ? '单通道' : (n === 2 ? '双通道' : `多通道(${n})`))
      return tags
    },
    // 点击设备图标：打开设备卡片（图二样式），默认自动播放第一个在线通道。
    deviceClickEvent(items) {
      const dev = (items && items[0]) || null
      if (!dev || !dev.deviceId) return
      this.openDeviceCard(dev.deviceId, dev.position)
    },
    // 打开某设备的卡片：先按已有分组数据填，再拉一次详情补厂商/安装位置/协议/编号。
    openDeviceCard(deviceId, position) {
      const g = this.deviceGroups[deviceId]
      if (!g) return
      const meta = this.deviceMetaMap[deviceId] || {}
      const c0 = g.channels[0] || {}
      this.closeInfoBox()
      this.deviceInfo = {
        deviceId: String(deviceId),
        name: meta.name || g.name || this.guessDeviceName(g.channels),
        deviceType: meta.deviceType || '',
        protocol: c0.protocol || '',
        code: c0.gbCode || c0.externalKey || '',
        vendor: '',
        address: '',
        channels: g.channels,
        position: position || g.position
      }
      const first = g.channels.find(c => c.gbStatus === 'ON') || g.channels[0]
      if (first) this.playChannelOfDevice(first)
      getAssetDetail(Number(deviceId)).then(res => {
        if (!this.deviceInfo || String(this.deviceInfo.deviceId) !== String(deviceId)) return
        const d = res.data || {}
        const dev = d.device || {}
        const ext = d.ext || {}
        const b = d.binding || {}
        this.deviceInfo.vendor = dev.vendor || ''
        this.deviceInfo.address = ext.address || ''
        if (b.protocol) this.deviceInfo.protocol = b.protocol
        if (b.protocol === 'gb28181' && b.externalId) this.deviceInfo.code = b.externalId
        else if (dev.internalCode) this.deviceInfo.code = dev.internalCode
      }).catch(() => {})
      this.$nextTick(() => {
        this.deviceInfoBoxId = this.$refs.mapComponent.openInfoBox(this.deviceInfo.position, this.$refs.deviceInfoBox, [0, -50])
      })
    },
    protoText(p) {
      return { gb28181: 'GB28181', onvif: 'ONVIF', plugin: 'SDK' }[p] || (p || '—')
    },
    playFirstChannel() {
      if (!this.deviceInfo) return
      const first = this.deviceInfo.channels.find(c => c.gbStatus === 'ON') || this.deviceInfo.channels[0]
      if (first) this.playChannelOfDevice(first)
    },
    editDevice() {
      if (this.deviceInfo) this.editDeviceById(this.deviceInfo.deviceId)
    },
    editDeviceById(deviceId) {
      this.deviceEditRow = { dbId: Number(deviceId), protocol: (this.deviceInfo && this.deviceInfo.protocol) || '' }
      this.deviceEditVisible = true
    },
    locateDevice() {
      if (this.deviceInfo && this.deviceInfo.position) {
        this.$refs.mapComponent.panTo(this.deviceInfo.position, 16)
      }
    },
    // 按通道 id 反查其所属设备 id（deviceGroups 的 key）。
    findDeviceIdByChannel(channelId) {
      return Object.keys(this.deviceGroups).find(k =>
        this.deviceGroups[k].channels.some(c => String(c.gbId) === String(channelId))
      )
    },
    // 切换设备内通道：统一交给 play()（内部会停掉上一个不同通道），避免重复起流。
    playChannelOfDevice(ch) {
      if (!ch) return
      this.play(ch)
    },
    refreshLayer(){
      this.closeInfoBox()

      // 刷新瓦片图层
      if (channelLayer) {
        this.$refs.mapComponent.refreshLayer(channelLayer)
      }
      if (channelTileLayer) {
        this.$refs.mapComponent.refreshLayer(channelTileLayer)
      }
    },
    treeChannelClickEvent: function (id) {
      const deviceId = this.findDeviceIdByChannel(id)
      if (!deviceId) {
        this.$message.warning({ showClose: true, message: '该设备无位置信息' })
        return
      }
      const pos = this.deviceGroups[deviceId].position
      const zoomExtent = this.$refs.mapComponent.getZoomExtent()
      this.$refs.mapComponent.panTo(pos, zoomExtent[1], () => { this.openDeviceCard(deviceId, pos) })
    },
    zoomIn: function() {
      this.$refs.mapComponent.zoomIn()
    },
    zoomOut: function() {
      this.$refs.mapComponent.zoomOut()
    },
    getContextmenu: function (event) {
        return [
          {
            label: '播放',
            icon: 'el-icon-video-play',
            type: 1,
            onClick: (event, data, node) => {
              this.$store.dispatch('commonChanel/queryOne', data.id)
                .then(d => { this.play(d) })
                .catch((error) => {
                  this.$message({ showClose: true, message: error, type: 'error' })
                })
            }
          },
          {
            label: '编辑设备',
            icon: 'el-icon-edit',
            type: 1,
            onClick: (event, data, node) => {
              const deviceId = this.findDeviceIdByChannel(data.id)
              if (deviceId) this.editDeviceById(deviceId)
            }
          }
        ]
    },
    zoomChange: function(zoom) {},

    changeMapTile: function (index) {
      if (this.showDrawThin) {
        this.$message.warning({
          showClose: true,
          message: '抽稀操作进行中，禁止切换图层'
        })
        return
      }
      this.$refs.mapComponent.changeMapTile(index)
      this.changeLayerType(this.layerType)
    },
    clientEvent(data){
      const ch = data && data[0]
      if (!ch) return
      const deviceId = this.findDeviceIdByChannel(ch.gbId != null ? ch.gbId : ch.resourceId)
      if (deviceId) this.openDeviceCard(deviceId, this.deviceGroups[deviceId].position)
    },
    changeLayerType: function (index) {
      this.layerType = index
      if (index === 0) {
        this.$refs.mapComponent.removeLayer(channelTileLayer)
        return
      }
      if (channelTileLayer) {
        this.$refs.mapComponent.removeLayer(channelTileLayer)
      }

      const baseUrl = window.baseUrl ? window.baseUrl : ''
      let baseApi = ((process.env.NODE_ENV === 'development') ? process.env.VUE_APP_BASE_API : baseUrl)
      let tileUrl = null
      if (index === 1) {
        tileUrl = baseApi + '/api/v1/common/channel/map/tile/{z}/{x}/{y}'
      }else if (index === 2) {
        tileUrl = baseApi + '/api/v1/common/channel/map/thin/tile/{z}/{x}/{y}'
      }
      channelTileLayer = this.$refs.mapComponent.addVectorTileLayer(tileUrl, this.clientEvent)
    },
    closeInfoBox: function () {
      if (this.deviceInfoBoxId !== null) {
        this.$refs.mapComponent.closeInfoBox(this.deviceInfoBoxId)
        this.deviceInfoBoxId = null
      }
      this.deviceInfo = null
    },
    play: function (channel) {
      // gbId 即通道 resourceId，走统一资源取流（GB/ONVIF/插件通用），不再走旧的国标专用接口。
      if (!channel || channel.gbId == null) return
      // 重新点播放：撤销关闭时排定的“延时停流”，让热流继续复用（避免每次重开都重建拉流）。
      this.cancelCloseStop()
      const targetId = channel.gbId
      // 同一通道正在起流中：忽略重复点击，避免并发 StartLive 在 ZMS 互相踢流。
      if (this.playLoading && String(this.playingChannelId) === String(targetId)) return
      // 切到别的通道/设备：先把上一路停掉，避免后台残留拉流。
      const prev = this.playingChannelId
      if (prev != null && String(prev) !== String(targetId)) {
        stopResourceLive(prev).catch(() => {})
      }
      // 请求序号：只有最新一次请求的结果允许落到播放器，过期结果只做清理。
      const seq = ++this.playSeq
      this.playingChannelId = targetId
      this.playLoading = true
      this.$refs.player.openDialog('codec', targetId, {
        pending: true,
        name: channel.gbName,
        hasAudio: channel.hasAudio,
        ptzType: channel.ptzType
      })
      startResourceLive(targetId).then(res => {
        if (seq !== this.playSeq) {
          // 请求已过期：只有在当前目标已不是这个通道（切走/关闭）时才清理，
          // 避免误杀“关掉后立刻又点开同一通道”时新请求刚建好的流。
          if (this.playingChannelId == null || String(this.playingChannelId) !== String(targetId)) {
            stopResourceLive(targetId).catch(() => {})
          }
          return
        }
        this.playLoading = false
        const content = (res && res.data) || null
        if (!content) {
          this.$refs.player.onStreamError('未返回播放地址')
          return
        }
        this.$refs.player.onStreamReady(content, channel.hasAudio)
      }).catch(error => {
        if (seq !== this.playSeq) return
        this.playLoading = false
        this.$refs.player.onStreamError((error && error.message) || '取流失败')
      })
    },
    // 撤销“关闭后延时停流”的定时器（重新点播放时用）。
    cancelCloseStop() {
      if (this.closeStopTimer) {
        clearTimeout(this.closeStopTimer)
        this.closeStopTimer = null
      }
    },
    // 关闭弹窗不停流：保留这一路热流（地图同一时刻至多 1 路），再次打开秒开。
    // 切到别的通道时 play() 会停掉上一路；离开地图（destroyed）时统一停掉。
    // 只在途请求失效，避免回落进已关闭的弹窗。
    onPlayerClosed() {
      this.cancelCloseStop()
      this.playSeq++
      this.playLoading = false
    },
    showDrawThinBox: function(show){
      this.showDrawThin = show
      if (!show) {
        setTimeout(() => {
          // 关闭抽稀预览
          if (this.drawThinId !== null) {
            // 发送消息 清空抽稀结果
            this.$store.dispatch('commonChanel/clearThin', this.drawThinId)
              .catch((error) => {
                this.$message({
                  showClose: true,
                  message: error,
                  type: 'error'
                })
              })
            this.drawThinId = null
          }
          if (this.drawThinLayer !== null) {
            this.$refs.mapComponent.removeLayer(this.drawThinLayer)
            this.drawThinLayer = null
          }
          // 展示图层
          if (this.layerType > 0) {
            this.changeLayerType(this.layerType)
          }
        }, 1)
      }

    },
    quicklyDrawThin: function (){
      if (channelLayer) {
        this.$refs.mapComponent.removeLayer(channelLayer)
      }
      if (channelTileLayer) {
        this.$refs.mapComponent.removeLayer(channelTileLayer)
      }
      if (this.drawThinLayer !== null) {
        this.$refs.mapComponent.removeLayer(this.drawThinLayer)
        this.drawThinLayer = null
      }
      this.quicklyDrawThinLoading = true
      // 获取每一个图层的抽稀参数
      this.$store.dispatch('commonChanel/drawThin', {
        zoomParam: this.getDrawThinParam()
      })
        .then(drawThinId => {
          // 显示抽稀进度
          this.drawThinId = drawThinId
          this.$refs.drawThinProgress.openDialog(drawThinId, () => {
            this.closeInfoBox()
            this.$message.success({
              showClose: true,
              message: '抽稀完成，请预览无误后保存抽稀结果'
            })
            // 展示抽稀结果
            this.showDrawThinLayer(drawThinId)
          })
        })
        .catch((error) => {
          this.$message({
            showClose: true,
            message: error,
            type: 'error'
          })
        })
        .finally(() => {
          this.quicklyDrawThinLoading = false
        })
    },
    showDrawThinLayer(thinId) {
      if (this.drawThinLayer) {
        this.$refs.mapComponent.removeLayer(this.drawThinLayer)
        this.drawThinLayer = null
      }
      // 展示抽稀结果
      let geoCoordSys = this.$refs.mapComponent.getCoordSys()
      const baseUrl = window.baseUrl ? window.baseUrl : ''
      let baseApi = ((process.env.NODE_ENV === 'development') ? process.env.VUE_APP_BASE_API : baseUrl)
      let tileUrl = baseApi + `/api/v1/common/channel/map/thin/tile/{z}/{x}/{y}?geoCoordSys=${geoCoordSys}&thinId=${thinId}&accessToken=${this.$store.getters.token}`
      this.drawThinLayer = this.$refs.mapComponent.addVectorTileLayer(tileUrl, this.clientEvent)
    },
    boxDrawThin: function (){
      this.$message.warning({
        showClose: true,
        message: '点击地图进行框选'
      })
      // 绘制框
      this.$refs.mapComponent.startDrawBox((extent) => {

        // 清理默认的摄像头图层
        if (channelLayer) {
          this.$refs.mapComponent.removeLayer(channelLayer)
        }
        if (channelTileLayer) {
          this.$refs.mapComponent.removeLayer(channelTileLayer)
        }
        if (this.drawThinLayer !== null) {
          this.$refs.mapComponent.removeLayer(this.drawThinLayer)
          this.drawThinLayer = null
        }
        this.boxDrawThinLoading = true
        // 获取每一个图层的抽稀参数
        this.$store.dispatch('commonChanel/drawThin', {
          zoomParam: this.getDrawThinParam(),
          extent: {
            minLng: extent[0],
            minLat: extent[1],
            maxLng: extent[2],
            maxLat: extent[3]
          },
          geoCoordSys: 'GCJ02'
        })
          .then(drawThinId => {
            // 显示抽稀进度
            this.drawThinId = drawThinId
            this.$refs.drawThinProgress.openDialog(drawThinId, () => {
              this.closeInfoBox()
              this.$message.success({
                showClose: true,
                message: '抽稀完成，请预览无误后保存抽稀结果'
              })
              // 展示抽稀结果
              this.showDrawThinLayer(drawThinId)
            })
          })
          .catch((error) => {
            this.$message({
              showClose: true,
              message: error,
              type: 'error'
            })
          })
          .finally(() => {
            this.boxDrawThinLoading = false
          })
      })
    },
    getDrawThinParam() {
      // 获取全部层级
      let zoomExtent = this.$refs.mapComponent.getZoomExtent()
      let zoomMap = {}
      let zoom = zoomExtent[0]
      while (zoom <= zoomExtent[1]) {
        // 计算经纬度差值
        let diff = this.$refs.mapComponent.computeDiff(this.diffPixels, zoom)
        if (diff && diff > 0) {
          zoomMap[zoom] = diff
        }
        zoom += 1
      }
      return zoomMap
    },

    saveDrawThin: function(){
      if (!this.drawThinId) {
        return
      }
      this.saveDrawThinLoading = true
      this.$store.dispatch('commonChanel/saveThin', this.drawThinId)
        .then((data) => {
          this.$message.success({
            showClose: true,
            message: '保存成功'
          })
          this.showDrawThinBox(false)
        })
        .catch((error) => {
          this.$message({
            showClose: true,
            message: error,
            type: 'error'
          })
        })
        .finally(() => {
          this.saveDrawThinLoading = false
        })
    },
    resetDrawThinData(){
      this.$confirm('确定移除抽稀结果?', '操作提示', {
        confirmButtonText: '确认',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(() => {
        this.$store.dispatch('commonChanel/resetLevel')
          .then(() => {
            this.$message.success({
              showClose: true,
              message: '数据还原成功'
            })
          })
          .catch((error) => {
            this.$message({
              showClose: true,
              message: error,
              type: 'error'
            })
          })
      })
    }
  }

}
</script>

<style>
.map-tool-box-bottom-right {
  position: absolute;
  right: 20px;
  bottom: 20px;
}
.map-tool-box-top-right {
  position: absolute;
  right: 20px;
  top: 20px;
}
.map-tool-box-top-left {
  position: absolute;
  left: 380px;
  top: 20px;
}
.map-tool-btn-group {
  background-color: #FFFFFF;
  border-radius: 3px;
  user-select: none;
  box-shadow: 0 2px 2px rgba(0, 0, 0, .15);
  margin-bottom: 10px;
}
.map-tool-box-top-left .map-tool-btn-group {
  display: flex;
}
.map-tool-box-top-right .map-tool-btn-group {
  display: flex;
}
.map-tool-box-top-left .map-tool-btn {
  padding: 0 10px;
}
.map-tool-box-top-right .map-tool-btn {
  padding: 0 10px;
}
.map-tool-btn {
  border-bottom: 1px #dfdfdf solid;
  border-right: 1px #dfdfdf solid;
  width: fit-content;
  min-width: 33px;
  height: 36px;
  cursor: pointer;
  text-align: center;
  line-height: 36px;
  font-size: 14px;
}
.map-tool-btn i {
  font-size: 14px;
}
.map-tool-btn-group:last-child {
  border-bottom: none;
  border-right: none;
}
.map-tool-draw-thin {
  position: absolute;
  top: 63px;
  left: 380px;
  border: 1px solid #dfdfdf;
  background-color: #fff;
  border-radius: 4px;
  padding: 0 10px;
}
.map-tool-draw-thin-density {
  display: grid;
  grid-template-columns: 50px 400px auto;
  padding: 0;
  margin: 0;
}

/* 设备气泡卡片（现代轻量白底） */
.popup-light {
  position: relative;
  width: 260px;
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  box-shadow: 0 10px 30px rgba(15, 23, 42, .14);
  padding: 12px 14px 14px;
  box-sizing: border-box;
  font-family: inherit;
}
.popup-light::after {
  content: "";
  position: absolute;
  bottom: -7px;
  left: calc(50% - 7px);
  width: 12px;
  height: 12px;
  background: #fff;
  border-right: 1px solid #e2e8f0;
  border-bottom: 1px solid #e2e8f0;
  transform: rotate(45deg);
}
.popup-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  padding-bottom: 10px;
  border-bottom: 1px solid #eef2f7;
}
.popup-title-wrap {
  min-width: 0;
}
.popup-title {
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.popup-meta {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 4px;
  font-size: 12px;
  color: #94a3b8;
}
.status-dot {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: #16a34a;
  font-weight: 500;
}
.status-dot::before {
  content: "";
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #22c55e;
}
.status-dot.off {
  color: #94a3b8;
}
.status-dot.off::before {
  background: #cbd5e1;
}
.close-icon {
  flex: 0 0 auto;
  width: 20px;
  height: 20px;
  padding: 0;
  border: none;
  background: transparent;
  color: #94a3b8;
  font-size: 18px;
  line-height: 18px;
  cursor: pointer;
  border-radius: 4px;
}
.close-icon:hover {
  background: #f1f5f9;
  color: #475569;
}
.kv-list {
  padding: 10px 0 4px;
}
.kv-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 3px 0;
  font-size: 12.5px;
  line-height: 1.5;
}
.kv-label {
  flex: 0 0 40px;
  color: #94a3b8;
}
.kv-val {
  flex: 1;
  min-width: 0;
  color: #334155;
  word-break: break-all;
}
.kv-val.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  letter-spacing: .2px;
}
.kv-val.empty {
  color: #cbd5e1;
}
.chan-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 4px 0 8px;
}
.chan-chip {
  padding: 2px 10px;
  font-size: 12px;
  color: #475569;
  background: #f1f5f9;
  border: 1px solid transparent;
  border-radius: 999px;
  cursor: pointer;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.chan-chip:hover:not(:disabled) {
  background: #e2e8f0;
}
.chan-chip.active {
  color: #1a73e8;
  background: #e8f1fe;
  border-color: #bcd7fb;
}
.chan-chip:disabled {
  color: #cbd5e1;
  cursor: not-allowed;
}
.popup-actions {
  display: grid;
  grid-template-columns: 1.2fr 1fr 1fr;
  gap: 8px;
  margin-top: 8px;
}
.act-btn {
  height: 32px;
  border-radius: 7px;
  font-size: 13px;
  cursor: pointer;
  border: 1px solid transparent;
  transition: background .15s, border-color .15s;
}
.act-primary {
  background: #1a73e8;
  color: #fff;
}
.act-primary:hover:not(:disabled) {
  background: #1667d1;
}
.act-primary:disabled {
  background: #b9d3f7;
  cursor: not-allowed;
}
.act-secondary {
  background: #fff;
  color: #475569;
  border-color: #e2e8f0;
}
.act-secondary:hover {
  background: #f8fafc;
  border-color: #cbd5e1;
}
</style>
