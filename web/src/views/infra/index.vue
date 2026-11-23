<template>
  <div class="app-container infra-page">
    <el-alert
      v-if="restartRequired"
      type="warning"
      :closable="false"
      show-icon
      title="MySQL / Redis 配置已保存，请重启服务后生效"
      style="margin-bottom: 12px;"
    />
    <el-alert
      :type="pipeline.ready ? 'success' : 'warning'"
      :closable="false"
      show-icon
      :title="pipeline.ready ? '事件管道已就绪：对象存储与 MQTT 均可用' : '事件管道未就绪：请先完成对象存储与 MQTT 配置'"
      style="margin-bottom: 12px;"
    />

    <div class="status-row">
      <div class="status-card" v-for="item in statusCards" :key="item.key">
        <div class="status-head">
          <span class="status-name">{{ item.name }}</span>
          <el-tag :type="item.online ? 'success' : 'info'" size="mini">{{ item.online ? '在线' : '离线' }}</el-tag>
        </div>
        <div class="status-sub">{{ item.desc }}</div>
      </div>
    </div>

    <el-tabs v-model="tab" type="border-card">
      <el-tab-pane label="MySQL" name="mysql">
        <el-form v-loading="loading" :model="mysql" label-width="140px" style="max-width: 680px;" @submit.native.prevent>
          <el-form-item label="Host"><el-input v-model="mysql.host" /></el-form-item>
          <el-form-item label="Port"><el-input-number v-model="mysql.port" :min="1" :max="65535" /></el-form-item>
          <el-form-item label="User"><el-input v-model="mysql.user" /></el-form-item>
          <el-form-item label="Password"><el-input v-model="mysql.password" show-password placeholder="****** 表示不修改" /></el-form-item>
          <el-form-item label="Database"><el-input v-model="mysql.database" /></el-form-item>
          <el-form-item label="Charset"><el-input v-model="mysql.charset" /></el-form-item>
          <el-form-item label="MaxIdleConns"><el-input-number v-model="mysql.maxIdleConns" :min="1" /></el-form-item>
          <el-form-item label="MaxOpenConns"><el-input-number v-model="mysql.maxOpenConns" :min="1" /></el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="saving" @click="saveMySQL">保存</el-button>
            <span class="hint">修改后需重启服务生效</span>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="Redis" name="redis">
        <el-form v-loading="loading" :model="redis" label-width="140px" style="max-width: 680px;" @submit.native.prevent>
          <el-form-item label="Host"><el-input v-model="redis.host" /></el-form-item>
          <el-form-item label="Port"><el-input-number v-model="redis.port" :min="1" :max="65535" /></el-form-item>
          <el-form-item label="Password"><el-input v-model="redis.password" show-password placeholder="****** 表示不修改" /></el-form-item>
          <el-form-item label="Database"><el-input-number v-model="redis.database" :min="0" /></el-form-item>
          <el-form-item label="PoolSize"><el-input-number v-model="redis.poolSize" :min="1" /></el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="saving" @click="saveRedis">保存</el-button>
            <span class="hint">修改后需重启服务生效</span>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="消息队列（MQTT）" name="mqtt">
        <el-form v-loading="loading" :model="mqtt" label-width="150px" style="max-width: 720px;" @submit.native.prevent>
          <el-form-item label="启用"><el-switch v-model="mqtt.enabled" /></el-form-item>
          <el-form-item label="Broker"><el-input v-model="mqtt.broker" placeholder="tcp://192.168.1.10:1883 或 ssl://..." /></el-form-item>
          <el-form-item label="ClientID 前缀"><el-input v-model="mqtt.clientIdPrefix" /></el-form-item>
          <el-form-item label="用户名"><el-input v-model="mqtt.username" /></el-form-item>
          <el-form-item label="密码"><el-input v-model="mqtt.password" show-password placeholder="****** 表示不修改" /></el-form-item>
          <el-form-item label="Topic 前缀"><el-input v-model="mqtt.topicPrefix" placeholder="zws/events" /></el-form-item>
          <el-form-item label="QoS">
            <el-select v-model="mqtt.qos" style="width: 120px;">
              <el-option :value="0" label="0" />
              <el-option :value="1" label="1（推荐）" />
              <el-option :value="2" label="2" />
            </el-select>
          </el-form-item>
          <el-form-item label="KeepAlive(秒)"><el-input-number v-model="mqtt.keepAliveSec" :min="5" /></el-form-item>
          <el-form-item label="TLS"><el-switch v-model="mqtt.tlsEnabled" /></el-form-item>
          <template v-if="mqtt.tlsEnabled">
            <el-form-item label="CA 文件路径"><el-input v-model="mqtt.caCert" /></el-form-item>
            <el-form-item label="客户端证书"><el-input v-model="mqtt.clientCert" /></el-form-item>
            <el-form-item label="客户端私钥"><el-input v-model="mqtt.clientKey" /></el-form-item>
          </template>
          <el-form-item>
            <el-button type="primary" :loading="saving" @click="saveMQTT">保存并重载</el-button>
            <el-button :loading="checking" @click="checkMQTT">连通性检查</el-button>
            <span class="hint">MQTT 支持热重载，无需重启</span>
          </el-form-item>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="对象存储" name="objectStore">
        <el-form v-loading="loading" :model="objectStoreForm" label-width="120px" style="max-width: 640px;" @submit.native.prevent>
          <el-form-item label="启用"><el-switch v-model="objectStoreForm.enabled" /></el-form-item>
          <el-form-item label="驱动">
            <el-select v-model="objectStoreForm.provider" style="width: 100%;">
              <el-option label="noop（未对接）" value="noop" />
              <el-option label="MinIO" value="minio" />
              <el-option label="S3 兼容" value="s3" />
            </el-select>
          </el-form-item>
          <el-form-item label="Endpoint"><el-input v-model="objectStoreForm.endpoint" placeholder="如 192.168.1.10:9000" /></el-form-item>
          <el-form-item label="Region"><el-input v-model="objectStoreForm.region" placeholder="S3 区域，MinIO 可空" /></el-form-item>
          <el-form-item label="Bucket"><el-input v-model="objectStoreForm.bucket" /></el-form-item>
          <el-form-item label="AccessKey"><el-input v-model="objectStoreForm.accessKey" /></el-form-item>
          <el-form-item label="SecretKey"><el-input v-model="objectStoreForm.secretKey" show-password /></el-form-item>
          <el-form-item label="HTTPS"><el-switch v-model="objectStoreForm.useSSL" /></el-form-item>
          <el-form-item label="PathStyle">
            <el-switch v-model="objectStoreForm.pathStyle" />
            <span class="hint">MinIO 建议开启</span>
          </el-form-item>
          <el-form-item label="公网前缀"><el-input v-model="objectStoreForm.publicBase" placeholder="可选，用于拼直链" /></el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="osSaving" @click="saveObjectStore">保存</el-button>
            <el-button :loading="osChecking" @click="checkObjectStore">连通性检查</el-button>
          </el-form-item>
        </el-form>
        <el-alert v-if="osHealthMsg" :type="osHealthOk ? 'success' : 'warning'" :closable="false" :title="osHealthMsg" />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script>
import { getInfraOverview, saveMySQLConfig, saveRedisConfig, saveMQTTConfig, checkMQTTHealth, getPipelineStatus } from '@/api/infra'
import { getObjectStoreConfig, saveObjectStoreConfig, checkObjectStoreHealth } from '@/api/server'

export default {
  name: 'Infrastructure',
  data() {
    return {
      loading: false,
      saving: false,
      checking: false,
      tab: 'mysql',
      mysql: { host: '', port: 3306, user: '', password: '', database: '', charset: 'utf8mb4', maxIdleConns: 10, maxOpenConns: 100 },
      redis: { host: '', port: 6379, password: '', database: 0, poolSize: 10 },
      mqtt: { enabled: false, broker: '', clientIdPrefix: 'zws', username: '', password: '', topicPrefix: 'zws/events', qos: 1, keepAliveSec: 30, tlsEnabled: false, caCert: '', clientCert: '', clientKey: '' },
      objectStore: {},
      objectStoreForm: {
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
      },
      osSaving: false,
      osChecking: false,
      osHealthOk: false,
      osHealthMsg: '',
      pipeline: { ready: false },
      restartRequired: false,
      statusCards: []
    }
  },
  created() {
    this.load()
  },
  methods: {
    load() {
      this.loading = true
      getInfraOverview().then(res => {
        const d = (res && res.data) || {}
        this.restartRequired = !!d.restartRequired
        if (d.mysql) this.mysql = { ...this.mysql, ...d.mysql, password: d.mysql.password || '' }
        if (d.redis) this.redis = { ...this.redis, ...d.redis, password: d.redis.password || '' }
        if (d.mqtt) this.mqtt = { ...this.mqtt, ...d.mqtt, password: d.mqtt.password || '' }
        this.objectStore = d.objectStore || {}
        this.buildStatusCards(d)
      }).catch(e => this.$message.error(e || '加载失败'))
        .finally(() => { this.loading = false })
      getPipelineStatus().then(res => {
        this.pipeline = (res && res.data) || { ready: false }
      }).catch(() => {})
      getObjectStoreConfig().then(res => {
        const d = (res && res.data) || res || {}
        this.objectStoreForm = {
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
      }).catch(() => {})
    },
    buildStatusCards(d) {
      this.statusCards = [
        { key: 'mysql', name: 'MySQL', online: !!(d.mysql && d.mysql.status && d.mysql.status.online), desc: d.mysql ? `${d.mysql.host}:${d.mysql.port}/${d.mysql.database}` : '' },
        { key: 'redis', name: 'Redis', online: !!(d.redis && d.redis.status && d.redis.status.online), desc: d.redis ? `${d.redis.host}:${d.redis.port}` : '' },
        { key: 'mqtt', name: 'MQTT', online: !!(d.mqtt && d.mqtt.status && d.mqtt.status.online), desc: d.mqtt ? (d.mqtt.broker || '未配置') : '' },
        { key: 'objectStore', name: '对象存储', online: !!(d.objectStore && d.objectStore.status && d.objectStore.status.online), desc: d.objectStore ? (d.objectStore.provider || 'noop') : '' }
      ]
    },
    saveMySQL() {
      this.saving = true
      saveMySQLConfig(this.mysql).then(res => {
        const d = (res && res.data) || {}
        this.$alert(d.warning || '配置已保存，请重启服务后生效', '提示', { type: 'warning' })
        this.load()
      }).catch(e => this.$message.error(e || '保存失败'))
        .finally(() => { this.saving = false })
    },
    saveRedis() {
      this.saving = true
      saveRedisConfig(this.redis).then(res => {
        const d = (res && res.data) || {}
        this.$alert(d.warning || '配置已保存，请重启服务后生效', '提示', { type: 'warning' })
        this.load()
      }).catch(e => this.$message.error(e || '保存失败'))
        .finally(() => { this.saving = false })
    },
    saveMQTT() {
      this.saving = true
      saveMQTTConfig(this.mqtt).then(res => {
        const d = (res && res.data) || {}
        if (d.connected) this.$message.success('MQTT 已保存并连接成功')
        else this.$message.warning('MQTT 已保存，但未连接：' + (d.error || '未知错误'))
        this.load()
      }).catch(e => this.$message.error(e || '保存失败'))
        .finally(() => { this.saving = false })
    },
    checkMQTT() {
      this.checking = true
      checkMQTTHealth().then(res => {
        const d = (res && res.data) || {}
        if (d.ok) this.$message.success('MQTT 连接正常')
        else this.$message.warning('MQTT 未连接：' + (d.error || '未知错误'))
      }).catch(e => this.$message.error(e || '检查失败'))
        .finally(() => { this.checking = false })
    },
    saveObjectStore() {
      this.osSaving = true
      saveObjectStoreConfig(this.objectStoreForm).then(res => {
        const d = (res && res.data) || {}
        this.$message.success(d.message || '保存成功')
        this.load()
      }).catch(e => this.$message.error(e || '保存失败'))
        .finally(() => { this.osSaving = false })
    },
    checkObjectStore() {
      this.osChecking = true
      checkObjectStoreHealth().then(res => {
        const d = (res && res.data) || {}
        this.osHealthOk = !!d.ok
        this.osHealthMsg = d.ok
          ? `连通正常（provider=${d.provider}）`
          : `未连通：${d.error || 'unknown'}（provider=${d.provider}）`
      }).catch(e => {
        this.osHealthOk = false
        this.osHealthMsg = String(e || '检查失败')
      }).finally(() => { this.osChecking = false })
    }
  }
}
</script>

<style scoped>
.infra-page { padding: 12px; }
.status-row { display: flex; gap: 12px; margin-bottom: 12px; flex-wrap: wrap; }
.status-card { flex: 1 1 200px; background: #fff; border: 1px solid #e4e7ed; border-radius: 8px; padding: 12px 14px; }
.status-head { display: flex; align-items: center; justify-content: space-between; }
.status-name { font-weight: 600; color: #303133; }
.status-sub { margin-top: 6px; color: #80868B; font-size: 12px; word-break: break-all; }
.hint { margin-left: 10px; color: #80868B; font-size: 12px; }
</style>
