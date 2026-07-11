<template>
  <div id="gbConfig" class="app-container gb-config-page">
    <el-card shadow="never" class="gb-config-card">
      <div slot="header" class="gb-config-header">
        <span>国标配置</span>
        <span class="gb-config-sub">本平台唯一 SIP 配置，设备接入与国标级联共用</span>
      </div>
      <el-form
        ref="form"
        v-loading="loading"
        :model="form"
        :rules="rules"
        label-width="120px"
        style="max-width: 640px"
        @submit.native.prevent
      >
        <el-form-item label="SIP IP" prop="ip">
          <el-input v-model="form.ip" placeholder="摄像机可达的本机网卡地址" clearable />
        </el-form-item>
        <el-form-item label="SIP 端口" prop="port">
          <el-input-number v-model="form.port" :min="1" :max="65535" controls-position="right" style="width: 100%" />
        </el-form-item>
        <el-form-item label="SIP 域" prop="domain">
          <el-input v-model="form.domain" placeholder="例如 3402000000" clearable />
        </el-form-item>
        <el-form-item label="国标编号" prop="deviceId">
          <el-input v-model="form.deviceId" maxlength="20" show-word-limit placeholder="20位平台国标编号" clearable />
        </el-form-item>
        <el-form-item label="SIP 密码" prop="password">
          <el-input v-model="form.password" show-password placeholder="设备注册鉴权密码" clearable />
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
        alarm: false
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
            alarm: !!data.alarm
          }
        })
        .catch(err => {
          this.$message.error(err || '加载失败')
        })
        .finally(() => {
          this.loading = false
        })
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
.gb-config-page {
  max-width: 960px;
}
.gb-config-card {
  border-radius: 10px;
  border: 1px solid #e8eef6;
}
.gb-config-header {
  display: flex;
  align-items: baseline;
  gap: 12px;
}
.gb-config-sub {
  font-size: 12px;
  color: #909399;
  font-weight: 400;
}
</style>
