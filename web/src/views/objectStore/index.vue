<template>
  <div class="app-container">
    <el-card shadow="never">
      <div slot="header" class="hdr">
        <span>对象存储对接</span>
        <span class="sub">平台不自建存储，对接 MinIO / S3；未启用时抓拍归档等能力返回「未配置」</span>
      </div>
      <el-form ref="form" v-loading="loading" :model="form" label-width="120px" style="max-width: 640px;" @submit.native.prevent>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
        <el-form-item label="驱动">
          <el-select v-model="form.provider" style="width: 100%;">
            <el-option label="noop（未对接）" value="noop" />
            <el-option label="MinIO" value="minio" />
            <el-option label="S3 兼容" value="s3" />
          </el-select>
        </el-form-item>
        <el-form-item label="Endpoint">
          <el-input v-model="form.endpoint" placeholder="如 192.168.1.10:9000" />
        </el-form-item>
        <el-form-item label="Region">
          <el-input v-model="form.region" placeholder="S3 区域，MinIO 可空" />
        </el-form-item>
        <el-form-item label="Bucket">
          <el-input v-model="form.bucket" />
        </el-form-item>
        <el-form-item label="AccessKey">
          <el-input v-model="form.accessKey" />
        </el-form-item>
        <el-form-item label="SecretKey">
          <el-input v-model="form.secretKey" show-password />
        </el-form-item>
        <el-form-item label="HTTPS">
          <el-switch v-model="form.useSSL" />
        </el-form-item>
        <el-form-item label="PathStyle">
          <el-switch v-model="form.pathStyle" />
          <span class="hint">MinIO 建议开启</span>
        </el-form-item>
        <el-form-item label="公网前缀">
          <el-input v-model="form.publicBase" placeholder="可选，用于拼直链" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="saving" @click="save">保存</el-button>
          <el-button :loading="checking" @click="check">连通性检查</el-button>
        </el-form-item>
      </el-form>
      <el-alert v-if="healthMsg" :type="healthOk ? 'success' : 'warning'" :closable="false" :title="healthMsg" />
    </el-card>
  </div>
</template>

<script>
import { getObjectStoreConfig, saveObjectStoreConfig, checkObjectStoreHealth } from '@/api/server'

export default {
  name: 'ObjectStore',
  data() {
    return {
      loading: false,
      saving: false,
      checking: false,
      healthOk: false,
      healthMsg: '',
      form: {
        id: 1,
        enabled: false,
        provider: 'noop',
        endpoint: '',
        region: '',
        bucket: '',
        accessKey: '',
        secretKey: '',
        useSSL: false,
        pathStyle: true,
        publicBase: ''
      }
    }
  },
  created() {
    this.load()
  },
  methods: {
    load() {
      this.loading = true
      getObjectStoreConfig().then(res => {
        const d = (res && res.data) || res || {}
        this.form = {
          id: d.id || 1,
          enabled: !!d.enabled,
          provider: d.provider || 'noop',
          endpoint: d.endpoint || '',
          region: d.region || '',
          bucket: d.bucket || '',
          accessKey: d.accessKey || '',
          secretKey: d.secretKey || '',
          useSSL: !!d.useSSL,
          pathStyle: d.pathStyle !== false,
          publicBase: d.publicBase || ''
        }
      }).catch(e => this.$message.error(e || '加载失败'))
        .finally(() => { this.loading = false })
    },
    save() {
      this.saving = true
      saveObjectStoreConfig(this.form).then(res => {
        const d = (res && res.data) || {}
        this.$message.success(d.message || '保存成功')
        this.load()
      }).catch(e => this.$message.error(e || '保存失败'))
        .finally(() => { this.saving = false })
    },
    check() {
      this.checking = true
      checkObjectStoreHealth().then(res => {
        const d = (res && res.data) || {}
        this.healthOk = !!d.ok
        this.healthMsg = d.ok
          ? `连通正常（provider=${d.provider}）`
          : `未连通：${d.error || 'unknown'}（provider=${d.provider}）`
      }).catch(e => {
        this.healthOk = false
        this.healthMsg = String(e || '检查失败')
      }).finally(() => { this.checking = false })
    }
  }
}
</script>

<style scoped>
.hdr { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.sub { font-size: 12px; color: #909399; font-weight: normal; }
.hint { margin-left: 8px; font-size: 12px; color: #909399; }
</style>
