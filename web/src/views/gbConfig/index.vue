<template>
  <div id="gbConfig" class="app-container gb-config-page">
    <el-card shadow="never" class="gb-config-card">
      <div slot="header" class="gb-config-header">
        <span>本级国标身份</span>
        <span class="gb-config-sub">摄像机 / 下级平台把本页信息填为「上级」；改编码/域/密码/IP/预登记保存即生效；改端口需重启</span>
      </div>
      <el-form
        ref="form"
        v-loading="loading"
        :model="form"
        :rules="rules"
        label-width="140px"
        style="max-width: 720px"
        @submit.native.prevent
      >
        <el-form-item label="SIP IP" prop="ip">
          <el-input v-model="form.ip" placeholder="对端可达的本机网卡地址" clearable />
        </el-form-item>
        <el-form-item label="SIP 端口" prop="port">
          <el-input-number v-model="form.port" :min="1" :max="65535" controls-position="right" style="width: 100%" />
        </el-form-item>
        <el-form-item label="SIP 域" prop="domain">
          <el-input v-model="form.domain" placeholder="例如 3402000000" clearable />
        </el-form-item>
        <el-form-item label="本级国标编号" prop="deviceId">
          <el-input v-model="form.deviceId" maxlength="20" show-word-limit placeholder="20位平台国标编号" clearable />
        </el-form-item>
        <el-form-item label="默认注册密码" prop="password">
          <el-input v-model="form.password" show-password placeholder="设备未单独设密时使用" clearable />
        </el-form-item>
        <el-form-item label="信令传输">
          <el-radio-group v-model="form.transport">
            <el-radio label="UDP">UDP</el-radio>
            <el-radio label="TCP">TCP</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="强制预登记">
          <el-switch v-model="form.requirePreRegister" />
          <span class="hint">开启后：未在「设备列表」或「下级平台」预登记的编号，REGISTER 返回 403</span>
        </el-form-item>
        <el-form-item label="报警订阅">
          <el-switch v-model="form.alarm" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="saving" @click="onSubmit">保存</el-button>
          <el-button @click="load">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never" class="gb-config-card" style="margin-top: 16px;">
      <div slot="header" class="gb-config-header">
        <span>接入信息（填到摄像机 / 下级）</span>
        <el-button type="text" icon="el-icon-document-copy" @click="copyAccess">一键复制</el-button>
      </div>
      <el-descriptions :column="1" border size="small" style="max-width: 720px;">
        <el-descriptions-item label="上级平台 IP">{{ form.ip }}</el-descriptions-item>
        <el-descriptions-item label="上级平台端口">{{ form.port }}</el-descriptions-item>
        <el-descriptions-item label="上级平台编号">{{ form.deviceId }}</el-descriptions-item>
        <el-descriptions-item label="上级平台域">{{ form.domain }}</el-descriptions-item>
        <el-descriptions-item label="注册密码">{{ form.password }}</el-descriptions-item>
        <el-descriptions-item label="传输">{{ form.transport || 'UDP' }}</el-descriptions-item>
      </el-descriptions>
      <p class="hint" style="margin-top: 12px;">
        说明：摄像机或下级平台向上注册时，请填写本页「上级平台 IP / 端口 / 编号 / 域 / 密码」。下级平台需先在「国标级联 → 下级平台」中预登记其注册编号。
      </p>
    </el-card>
  </div>
</template>

<script>
import { getGbSipConfig, saveGbSipConfig } from '@/api/server'

export default {
  name: 'GbConfig',
  data() {
    return {
      loading: false,
      saving: false,
      form: {
        id: 1,
        ip: '',
        port: 5060,
        domain: '',
        deviceId: '',
        password: '',
        alarm: false,
        requirePreRegister: true,
        transport: 'UDP'
      },
      rules: {
        ip: [{ required: true, message: '请填写 SIP IP', trigger: 'blur' }],
        port: [{ required: true, message: '请填写 SIP 端口', trigger: 'change' }],
        domain: [{ required: true, message: '请填写 SIP 域', trigger: 'blur' }],
        deviceId: [
          { required: true, message: '请填写国标编号', trigger: 'blur' },
          { len: 20, message: '国标编号须为20位', trigger: 'blur' }
        ],
        password: [{ required: true, message: '请填写 SIP 密码', trigger: 'blur' }]
      }
    }
  },
  created() {
    this.load()
  },
  methods: {
    load() {
      this.loading = true
      getGbSipConfig()
        .then(res => {
          const data = (res && res.data) || res || {}
          this.form = {
            id: data.id || 1,
            ip: data.ip || '',
            port: data.port || 5060,
            domain: data.domain || '',
            deviceId: data.deviceId || '',
            password: data.password || '',
            alarm: !!data.alarm,
            requirePreRegister: data.requirePreRegister !== false,
            transport: data.transport || 'UDP'
          }
        })
        .catch(err => {
          this.$message.error(err || '加载失败')
        })
        .finally(() => {
          this.loading = false
        })
    },
    accessText() {
      return [
        `上级平台IP: ${this.form.ip}`,
        `上级平台端口: ${this.form.port}`,
        `上级平台编号: ${this.form.deviceId}`,
        `上级平台域: ${this.form.domain}`,
        `注册密码: ${this.form.password}`,
        `传输: ${this.form.transport || 'UDP'}`
      ].join('\n')
    },
    copyAccess() {
      const text = this.accessText()
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(() => {
          this.$message.success('已复制接入信息')
        }).catch(() => this.fallbackCopy(text))
      } else {
        this.fallbackCopy(text)
      }
    },
    fallbackCopy(text) {
      const ta = document.createElement('textarea')
      ta.value = text
      document.body.appendChild(ta)
      ta.select()
      try {
        document.execCommand('copy')
        this.$message.success('已复制接入信息')
      } catch (e) {
        this.$message.error('复制失败，请手动选择')
      }
      document.body.removeChild(ta)
    },
    onSubmit() {
      this.$refs.form.validate(valid => {
        if (!valid) return
        this.saving = true
        saveGbSipConfig(this.form)
          .then(res => {
            const data = (res && res.data) || {}
            this.$message.success(data.message || '保存成功')
            this.load()
          })
          .catch(err => {
            this.$message.error(err || '保存失败')
          })
          .finally(() => {
            this.saving = false
          })
      })
    }
  }
}
</script>

<style scoped>
.gb-config-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.gb-config-sub {
  font-size: 12px;
  color: #909399;
  font-weight: normal;
}
.hint {
  margin-left: 12px;
  font-size: 12px;
  color: #909399;
}
</style>
