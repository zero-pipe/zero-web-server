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
        <span class="on">1. 选择接入模式</span>
        <span>2. 填写协议参数</span>
      </div>
      <el-alert
        title="先添加，再接入。被动设备入库后为「待上线」，仅当设备用正确编号向平台注册后才变在线。"
        type="warning"
        :closable="false"
        style="margin-bottom: 14px;"
      />
      <div class="mode-cards">
        <div
          class="mode-card"
          :class="{ selected: accessMode === 'passive' }"
          @click="pickMode('passive')"
        >
          <h4><el-tag size="mini" type="success">被动</el-tag> 设备找平台</h4>
          <p>设备配置平台地址与自身 ID，主动向平台注册。比如：GB28181。</p>
        </div>
        <div
          class="mode-card"
          :class="{ selected: accessMode === 'active' }"
          @click="pickMode('active')"
        >
          <h4><el-tag size="mini" type="warning">主动</el-tag> 平台找设备</h4>
          <p>平台持有设备固定 IP，主动连接并拉取通道。比如：ONVIF。</p>
        </div>
      </div>
      <el-form ref="step1Form" :model="step1" :rules="step1Rules" label-width="90px" size="small" style="margin-top: 12px;">
        <el-form-item label="协议" required>
          <el-select v-model="protocol" style="width: 100%;">
            <el-option
              v-for="p in protocolOptions"
              :key="p.value"
              :label="p.label"
              :value="p.value"
            />
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
    </div>

    <div v-else-if="accessMode === 'passive'">
      <div class="steps">
        <span>1. 选择接入模式</span>
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
        <el-form-item label="设备编号" prop="deviceId">
          <el-input v-model="gb.deviceId" placeholder="20 位 GB28181 编码" />
        </el-form-item>
        <el-form-item label="注册密码">
          <el-input v-model="gb.password" type="password" show-password placeholder="可空则用平台默认" />
        </el-form-item>
        <el-form-item label="收流 IP">
          <el-input v-model="gb.sdpIp" placeholder="可选" />
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

    <div v-else>
      <div class="steps">
        <span>1. 选择接入模式</span>
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
      accessMode: 'passive',
      protocol: 'gb28181',
      vendor: '',
      deviceType: '',
      accessTip: '',
      step1: { name: '' },
      gb: {
        deviceId: '',
        password: '',
        sdpIp: '',
        mediaServerId: 'auto',
        charset: 'GB2312'
      },
      onvif: {
        ip: '',
        port: 80,
        username: 'admin',
        password: ''
      },
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
      return this.accessMode === 'passive' ? '添加设备 · GB28181（被动）' : '添加设备 · ONVIF（主动）'
    },
    protocolOptions() {
      if (this.accessMode === 'passive') {
        return [{ label: 'GB28181', value: 'gb28181' }]
      }
      return [{ label: 'ONVIF', value: 'onvif' }]
    }
  },
  mounted() {
    this.$root.$on('devices-prefill-onvif', this.prefillOnvif)
  },
  beforeDestroy() {
    this.$root.$off('devices-prefill-onvif', this.prefillOnvif)
  },
  methods: {
    pickMode(mode) {
      this.accessMode = mode
      this.protocol = mode === 'passive' ? 'gb28181' : 'onvif'
    },
    goNext() {
      this.$refs.step1Form.validate(valid => {
        if (!valid) return
        this.step = 2
        if (this.accessMode === 'passive') {
          this.loadAccessTip()
        }
      })
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
      this.pickMode('active')
      this.step = 1
      this.onvif.ip = data.ip || ''
      this.onvif.port = data.port || 80
      if (data.name && !this.step1.name) {
        this.step1.name = data.name
      }
    },
    reset() {
      this.step = 1
      this.accessMode = 'passive'
      this.protocol = 'gb28181'
      this.vendor = ''
      this.deviceType = ''
      this.accessTip = ''
      this.step1 = { name: '' }
      this.gb = { deviceId: '', password: '', sdpIp: '', mediaServerId: 'auto', charset: 'GB2312' }
      this.onvif = { ip: '', port: 80, username: 'admin', password: '' }
    },
    submit() {
      const formRef = this.accessMode === 'passive' ? 'gbForm' : 'onvifForm'
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
          accessMode: this.accessMode,
          protocol: this.protocol,
          name,
          vendor: this.vendor,
          deviceType: this.deviceType
        }
        if (this.accessMode === 'passive') {
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
  color: #64748b;
}
.steps span {
  padding: 4px 10px;
  border-radius: 999px;
  background: #f1f5f9;
}
.steps span.on {
  background: #e8f1fb;
  color: #1565c0;
  font-weight: 600;
}
.mode-cards {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}
.mode-card {
  border: 2px solid #e8eef6;
  border-radius: 8px;
  padding: 14px;
  cursor: pointer;
  transition: .15s;
}
.mode-card:hover { border-color: #93c5fd; }
.mode-card.selected {
  border-color: #1565c0;
  background: #e8f1fb;
}
.mode-card h4 {
  font-size: 14px;
  margin: 0 0 6px;
  display: flex;
  align-items: center;
  gap: 8px;
  color: #1e293b;
}
.mode-card p {
  margin: 0;
  font-size: 12px;
  color: #64748b;
  line-height: 1.5;
}
</style>
