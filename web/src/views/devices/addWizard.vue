<template>
  <el-dialog
    :title="dialogTitle"
    :visible.sync="innerVisible"
    width="640px"
    top="4vh"
    :close-on-click-modal="false"
    @close="reset"
  >
    <div v-if="step === 1">
      <div class="steps">
        <span class="on">1. 选择协议</span>
        <span>2. 填写协议参数</span>
      </div>
      <!-- 协议类别：标准协议（GB28181/ONVIF，后续可扩展）/ 插件协议 -->
      <div class="mode-cards">
        <div
          class="mode-card"
          :class="{ selected: protocolKind === 'standard' }"
          @click="pickKind('standard')"
        >
          <h4>标准协议</h4>
        </div>
        <div
          class="mode-card"
          :class="{ selected: protocolKind === 'plugin' }"
          @click="pickKind('plugin')"
        >
          <h4>厂商私有 SDK</h4>
        </div>
      </div>

      <!-- 标准协议：选 GB28181 / ONVIF -->
      <el-form v-if="protocolKind === 'standard'" ref="step1Form" :model="step1" :rules="step1Rules" label-width="90px" size="small" style="margin-top: 12px;">
        <el-form-item label="标准协议" required>
          <el-select v-model="protocol" style="width: 100%;">
            <el-option label="GB28181" value="gb28181" />
            <el-option label="ONVIF" value="onvif" />
          </el-select>
        </el-form-item>
        <el-form-item label="设备名称" prop="name">
          <el-input v-model="step1.name" placeholder="平台内唯一名称，通道将命名为 名称_channel_1" maxlength="64" show-word-limit />
        </el-form-item>
        <el-form-item label="厂商">
          <el-select v-model="vendor" clearable placeholder="不限 / 未知" style="width: 100%;">
            <el-option label="海康" value="海康" />
            <el-option label="大华" value="大华" />
            <el-option label="宇视" value="宇视" />
            <el-option label="其他" value="其他" />
          </el-select>
        </el-form-item>
        <el-form-item label="设备类型">
          <el-select v-model="deviceType" clearable placeholder="不强制" style="width: 100%;">
            <el-option label="IPC" value="IPC" />
            <el-option label="NVR" value="NVR" />
            <el-option label="球机" value="球机" />
            <el-option label="其他" value="其他" />
          </el-select>
        </el-form-item>
      </el-form>

      <!-- 插件协议：选插件 + 名称 -->
      <el-form v-else ref="step1Form" :model="step1" :rules="step1Rules" label-width="90px" size="small" style="margin-top: 12px;">
        <el-form-item label="插件" required>
          <el-select v-model="pluginForm.pluginId" style="width: 100%;" placeholder="选择在线插件" @change="onPluginChange">
            <el-option v-for="p in onlinePlugins" :key="p.id" :label="p.id + ' (' + p.protocols.join(',') + ')'" :value="p.id" />
          </el-select>
          <div v-if="!onlinePlugins.length" class="field-desc" style="color:#FBBC05">暂无在线插件，请先在「插件管理」安装并启用插件。</div>
        </el-form-item>
        <el-form-item label="插件协议">
          <el-select v-model="pluginForm.protocol" style="width: 100%;">
            <el-option v-for="p in pluginProtocols" :key="p" :label="p" :value="p" />
          </el-select>
        </el-form-item>
        <el-form-item label="设备名称" prop="name">
          <el-input v-model="step1.name" placeholder="留空则使用 IP:端口" maxlength="64" show-word-limit />
        </el-form-item>
      </el-form>
    </div>

    <!-- 标准协议参数 -->
    <div v-else-if="protocolKind === 'standard' && protocol === 'gb28181'">
      <div class="steps">
        <span>1. 选择协议</span>
        <span class="on">2. GB28181 参数</span>
      </div>
      <el-alert
        type="info"
        :closable="false"
        style="margin-bottom: 12px;"
        :title="'设备名称：' + step1.name + '；设备编号须与摄像机 GB28181 ID 一致。'"
      />
      <el-alert
        v-if="accessTip"
        type="success"
        :closable="false"
        style="margin-bottom: 12px;"
        :title="accessTip"
      />
      <el-form ref="gbForm" :model="gb" :rules="gbRules" label-width="100px" size="small">
        <el-form-item label="国标编号" prop="deviceId">
          <el-input v-model="gb.deviceId" placeholder="20 位 GB28181 编码" />
        </el-form-item>
        <el-form-item label="注册密码">
          <el-input v-model="gb.password" type="password" show-password placeholder="可空则用平台默认" />
        </el-form-item>
        <el-form-item label="流媒体节点">
          <el-select v-model="gb.mediaServerId" style="width: 100%;">
            <el-option label="自动负载最小" value="auto" />
          </el-select>
        </el-form-item>
        <el-form-item label="字符集">
          <el-select v-model="gb.charset" style="width: 100%;">
            <el-option label="GB2312" value="GB2312" />
            <el-option label="UTF-8" value="UTF-8" />
          </el-select>
        </el-form-item>
      </el-form>
    </div>

    <!-- ONVIF 参数 -->
    <div v-else-if="protocolKind === 'standard' && protocol === 'onvif'">
      <div class="steps">
        <span>1. 选择协议</span>
        <span class="on">2. ONVIF 参数</span>
      </div>
      <el-alert
        type="info"
        :closable="false"
        style="margin-bottom: 12px;"
        :title="'设备名称：' + step1.name"
      />
      <el-form ref="onvifForm" :model="onvif" :rules="onvifRules" label-width="90px" size="small">
        <el-form-item label="IP" prop="ip">
          <el-input v-model="onvif.ip" placeholder="192.168.1.100" />
        </el-form-item>
        <el-form-item label="端口" prop="port">
          <el-input v-model.number="onvif.port" />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="onvif.username" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="onvif.password" type="password" show-password />
        </el-form-item>
      </el-form>
    </div>

    <!-- 插件参数（按插件 configSchema 动态渲染） -->
    <div v-else>
      <div class="steps">
        <span>1. 选择协议</span>
        <span class="on">2. 插件参数</span>
      </div>
      <el-alert
        type="info"
        :closable="false"
        style="margin-bottom: 12px;"
        :title="'插件：' + pluginForm.pluginId + (step1.name ? '；设备名称：' + step1.name : '')"
      />
      <el-form label-width="110px" size="small">
        <el-form-item v-for="(f, key) in schemaProps" :key="key" :label="f.title || key">
          <el-input v-if="isNumber(f)" v-model.number="pluginForm.params[key]" />
          <el-input v-else-if="f.format === 'password'" v-model="pluginForm.params[key]" type="password" show-password />
          <el-input v-else v-model="pluginForm.params[key]" />
          <div v-if="f.description" class="field-desc">{{ f.description }}</div>
        </el-form-item>
      </el-form>
    </div>

    <div slot="footer">
      <el-button v-if="step > 1" size="small" @click="step = 1">上一步</el-button>
      <el-button size="small" @click="innerVisible = false">取消</el-button>
      <el-button v-if="step === 1" type="primary" size="small" @click="goNext">下一步</el-button>
      <el-button v-else type="primary" size="small" :loading="submitting" @click="submit">确认添加</el-button>
    </div>
  </el-dialog>
</template>

<script>
import { createDevice } from '@/api/devices'
import { addIpcDevice, queryPlugins } from '@/api/plugin'
import { queryAgents, queryAgentPlugins } from '@/api/agent'
import { getGbSipConfig } from '@/api/server'

export default {
  name: 'AddWizard',
  props: {
    visible: { type: Boolean, default: false }
  },
  data() {
    return {
      step: 1,
      submitting: false,
      protocolKind: 'standard',
      protocol: 'gb28181',
      vendor: '',
      deviceType: '',
      accessTip: '',
      step1: { name: '' },
      gb: {
        deviceId: '',
        password: '',
        mediaServerId: 'auto',
        charset: 'GB2312'
      },
      onvif: {
        ip: '',
        port: 80,
        username: 'admin',
        password: ''
      },
      // 插件添加
      onlinePlugins: [],
      pluginForm: { pluginId: '', protocol: '', params: {} },
      pluginProtocols: [],
      schemaProps: {},
      step1Rules: {
        name: [
          { required: true, message: '请输入设备名称', trigger: 'blur' },
          { min: 1, max: 64, message: '长度 1~64', trigger: 'blur' }
        ]
      },
      gbRules: {
        deviceId: [{ required: true, message: '请输入设备编号', trigger: 'blur' }]
      },
      onvifRules: {
        ip: [{ required: true, message: '请输入 IP', trigger: 'blur' }],
        port: [{ required: true, message: '请输入端口', trigger: 'blur' }]
      }
    }
  },
  computed: {
    innerVisible: {
      get() { return this.visible },
      set(v) { this.$emit('update:visible', v) }
    },
    dialogTitle() {
      if (this.step === 1) return '添加设备'
      if (this.protocolKind === 'plugin') return '添加设备 · 插件'
      return this.protocol === 'gb28181' ? '添加设备 · GB28181' : '添加设备 · ONVIF'
    }
  },
  mounted() {
    this.$root.$on('devices-prefill-onvif', this.prefillOnvif)
  },
  beforeDestroy() {
    this.$root.$off('devices-prefill-onvif', this.prefillOnvif)
  },
  methods: {
    pickKind(kind) {
      this.protocolKind = kind
      if (kind === 'standard') {
        this.protocol = 'gb28181'
      } else {
        this.loadOnlinePlugins()
      }
    },
    loadOnlinePlugins() {
      // 插件来自 Agent 托管，在线插件才能添加设备。
      queryAgents().then(res => {
        const agents = (res.data && res.data.list) || []
        return Promise.all(agents.filter(a => a.status === 'online').map(a =>
          queryAgentPlugins(a.id).then(r => (r.data && r.data.list) || []).catch(() => [])
        ))
      }).then(lists => {
        const all = lists.reduce((acc, cur) => acc.concat(cur), [])
        // 兼容 queryPlugins 的平台插件表（含 configSchema）
        return queryPlugins().then(r => {
          const platform = (r.data && r.data.list) || []
          const byId = {}
          platform.forEach(p => { byId[p.id] = p })
          this.onlinePlugins = all.map(p => ({ ...p, ...(byId[p.pluginId] || byId[p.id] || {}) }))
            .filter(p => p.online !== false)
        }).catch(() => { this.onlinePlugins = all })
      }).catch(() => { this.onlinePlugins = [] })
    },
    onPluginChange(id) {
      const plugin = this.onlinePlugins.find(p => p.id === id || p.pluginId === id)
      this.pluginProtocols = (plugin && plugin.protocols) || []
      this.pluginForm.protocol = this.pluginProtocols[0] || ''
      this.schemaProps = {}
      this.pluginForm.params = {}
      if (plugin && plugin.configSchema) {
        try {
          const schema = JSON.parse(plugin.configSchema)
          this.schemaProps = schema.properties || {}
          Object.keys(this.schemaProps).forEach(key => {
            const def = this.schemaProps[key].default
            this.$set(this.pluginForm.params, key, def !== undefined ? def : '')
          })
        } catch (e) {
          console.warn('invalid config schema', e)
        }
      }
    },
    isNumber(f) {
      return f.type === 'integer' || f.type === 'number'
    },
    goNext() {
      if (this.protocolKind === 'standard') {
        this.$refs.step1Form.validate(valid => {
          if (!valid) return
          this.step = 2
          if (this.protocol === 'gb28181') this.loadAccessTip()
        })
        return
      }
      // 插件：只需选插件即可进入下一步
      if (!this.pluginForm.pluginId) {
        this.$message.warning('请选择插件')
        return
      }
      this.step = 2
    },
    loadAccessTip() {
      getGbSipConfig().then(res => {
        const d = (res && res.data) || res || {}
        this.accessTip = `请在摄像机上配置上级：IP ${d.ip || '-'} 端口 ${d.port || '-'} 编号 ${d.deviceId || '-'} 域 ${d.domain || '-'} 密码（设备密码或平台默认）。可在「系统管理→国标配置」一键复制。`
      }).catch(() => {
        this.accessTip = '请先在「系统管理→国标配置」填写本级平台信息，再配置摄像机。'
      })
    },
    prefillOnvif(data) {
      this.protocolKind = 'standard'
      this.protocol = 'onvif'
      this.step = 1
      this.onvif.ip = data.ip || ''
      this.onvif.port = data.port || 80
      if (data.name && !this.step1.name) {
        this.step1.name = data.name
      }
    },
    reset() {
      this.step = 1
      this.protocolKind = 'standard'
      this.protocol = 'gb28181'
      this.vendor = ''
      this.deviceType = ''
      this.accessTip = ''
      this.step1 = { name: '' }
      this.gb = { deviceId: '', password: '', mediaServerId: 'auto', charset: 'GB2312' }
      this.onvif = { ip: '', port: 80, username: 'admin', password: '' }
      this.onlinePlugins = []
      this.pluginForm = { pluginId: '', protocol: '', params: {} }
      this.pluginProtocols = []
      this.schemaProps = {}
    },
    submit() {
      if (this.protocolKind === 'plugin') {
        this.submitPlugin()
        return
      }
      const formRef = this.protocol === 'gb28181' ? 'gbForm' : 'onvifForm'
      this.$refs[formRef].validate(valid => {
        if (!valid) return
        const name = (this.step1.name || '').trim()
        if (!name) {
          this.$message.warning('请填写设备名称')
          this.step = 1
          return
        }
        this.submitting = true
        const payload = {
          // 接入模式对用户不可见：GB 为被动注册，ONVIF 为主动连接。
          accessMode: this.protocol === 'gb28181' ? 'passive' : 'active',
          protocol: this.protocol,
          name,
          vendor: this.vendor,
          deviceType: this.deviceType
        }
        if (this.protocol === 'gb28181') {
          payload.gb = { ...this.gb }
        } else {
          payload.onvif = { ...this.onvif }
        }
        createDevice(payload).then(() => {
          this.$message.success('设备已添加')
          this.innerVisible = false
          this.$emit('success')
        }).catch(err => {
          this.$message.error(err.message || err || '添加失败')
        }).finally(() => {
          this.submitting = false
        })
      })
    },
    submitPlugin() {
      this.submitting = true
      addIpcDevice({
        pluginId: this.pluginForm.pluginId,
        protocol: this.pluginForm.protocol,
        name: (this.step1.name || '').trim(),
        params: this.pluginForm.params
      }).then(() => {
        this.$message.success('设备已添加')
        this.innerVisible = false
        this.$emit('success')
      }).catch(err => {
        this.$message.error(err.message || err || '添加失败')
      }).finally(() => {
        this.submitting = false
      })
    }
  }
}
</script>

<style scoped>
.steps {
  display: flex;
  gap: 8px;
  margin-bottom: 14px;
  font-size: 12px;
  color: #5F6368;
}
.steps span {
  padding: 4px 10px;
  border-radius: 999px;
  background: #f1f5f9;
}
.steps span.on {
  background: #E8F0FE;
  color: #1A73E8;
  font-weight: 600;
}
.mode-cards {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.mode-card {
  border: 2px solid #E8EAED;
  border-radius: 8px;
  padding: 22px 14px;
  cursor: pointer;
  transition: .15s;
  text-align: center;
}
.mode-card:hover { border-color: #93c5fd; }
.mode-card.selected {
  border-color: #1A73E8;
  background: #E8F0FE;
}
.mode-card h4 {
  font-size: 18px;
  font-weight: 600;
  margin: 0;
  color: #202124;
}
.field-desc { color: #80868B; font-size: 12px; line-height: 18px; margin-top: 4px; }
</style>
