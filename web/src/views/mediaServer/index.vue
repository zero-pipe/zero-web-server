<template>
  <div id="mediaServerManger" class="app-container media-server-page">
    <el-form :inline="true" size="mini" class="media-server-toolbar">
      <el-button icon="el-icon-plus" size="mini" type="primary" @click="add">添加节点</el-button>
    </el-form>
    <el-row :gutter="16">
      <el-col v-for="item in mediaServerList" :key="item.id" :span="getNumberByWidth()">
        <el-card shadow="hover" :body-style="{ padding: '0' }" class="server-card">
          <div class="server-card__banner">
            <div
              class="server-card__logo"
              :class="isZms(item) ? 'is-zms' : 'is-abl'"
            />
            <div class="server-card__badges">
              <el-tag size="mini" :type="item.status ? 'success' : 'info'" effect="dark">
                {{ item.status ? '在线' : '离线' }}
              </el-tag>
              <el-tag size="mini" type="primary" effect="plain">{{ typeLabel(item) }}</el-tag>
            </div>
          </div>

          <div class="server-card__body">
            <div class="server-card__field">
              <span class="server-card__label">节点 ID</span>
              <span class="server-card__value server-card__value--id" :title="item.id">{{ item.id }}</span>
            </div>

            <div class="server-card__meta">
              <div class="server-card__field">
                <span class="server-card__label">服务地址</span>
                <span class="server-card__value">{{ item.ip }}:{{ item.httpPort }}</span>
              </div>
              <div class="server-card__field server-card__field--load">
                <span class="server-card__label">活跃流</span>
                <span class="server-card__value">{{ item.load == null ? 0 : item.load }} <small>路</small></span>
              </div>
            </div>
          </div>

          <div class="server-card__footer">
            <span class="server-card__time">{{ item.createTime || '—' }}</span>
            <div class="server-card__actions">
              <el-button icon="el-icon-edit" circle size="mini" title="编辑" @click="edit(item)" />
              <el-button icon="el-icon-delete" circle size="mini" type="danger" plain title="删除" @click="del(item)" />
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>
    <edit ref="edit" />
  </div>
</template>

<script>
import edit from './edit.vue'

export default {
  name: 'MediaServer',
  components: { edit },
  data() {
    return {
      mediaServerList: [],
      currentPage: 1,
      count: 15,
      total: 0
    }
  },
  mounted() {
    this.initData()
  },
  methods: {
    initData() {
      this.getServerList()
    },
    getServerList() {
      this.$store.dispatch('server/getMediaServerList').then((data) => {
        this.mediaServerList = data || []
      })
    },
    isZms(item) {
      const t = (item.type || '').toLowerCase()
      return t === 'zms' || t === 'zeromediakit' || t === 'zlm' || t === ''
    },
    typeLabel(item) {
      if (this.isZms(item)) return 'ZMS'
      if ((item.type || '').toLowerCase() === 'abl') return 'ABL'
      return (item.type || '节点').toUpperCase()
    },
    add() {
      this.$refs.edit.openDialog(null, this.initData)
    },
    edit(row) {
      this.$refs.edit.openDialog(row, this.initData)
    },
    del(row) {
      this.$confirm('确认删除此节点？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(() => {
        this.$store.dispatch('server/deleteMediaServer', row.id).then(() => {
          this.$message({ type: 'success', message: '删除成功!' })
          this.getServerList()
        })
      }).catch(() => {})
    },
    getNumberByWidth() {
      const candidateNums = [1, 2, 3, 4, 6, 8, 12, 24]
      const clientWidth = window.innerWidth - 30
      const interval = 40
      const itemWidth = 360
      const num = (clientWidth + interval) / (itemWidth + interval)
      const result = Math.ceil(24 / num)
      for (let i = 0; i < candidateNums.length; i++) {
        const value = candidateNums[i]
        if (i + 1 >= candidateNums.length) return 24
        if (value <= result && candidateNums[i + 1] > result) return value
      }
      return 24
    }
  }
}
</script>

<style scoped>
.media-server-page {
  height: calc(100vh - 124px);
}

.media-server-toolbar {
  margin-bottom: 1rem;
}

.server-card {
  margin-bottom: 16px;
  border-radius: 8px;
  overflow: hidden;
}

.server-card__banner {
  position: relative;
  height: 120px;
  background: linear-gradient(180deg, #e3f2fd 0%, #f8fbff 100%);
  display: flex;
  align-items: center;
  justify-content: center;
}

.server-card__logo {
  width: 72px;
  height: 72px;
  background-repeat: no-repeat;
  background-position: center;
  background-size: contain;
}

.server-card__logo.is-zms {
  background-image: url('../../assets/brand/zero-logo.svg');
}

.server-card__logo.is-abl {
  background-image: url('../../assets/abl-logo.jpg');
  border-radius: 10px;
}

.server-card__badges {
  position: absolute;
  top: 12px;
  right: 12px;
  display: flex;
  gap: 6px;
}

.server-card__body {
  padding: 14px 16px 8px;
}

.server-card__field {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.server-card__label {
  font-size: 12px;
  color: #94a3b8;
  line-height: 1.2;
}

.server-card__value {
  font-size: 14px;
  color: #1e293b;
  font-weight: 600;
  line-height: 1.4;
  word-break: break-all;
}

.server-card__value--id {
  font-size: 16px;
  color: #0d47a1;
}

.server-card__value small {
  font-size: 12px;
  font-weight: 500;
  color: #64748b;
}

.server-card__meta {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 12px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #eef2f7;
}

.server-card__field--load {
  text-align: right;
  min-width: 72px;
}

.server-card__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px 12px;
  border-top: 1px solid #f1f5f9;
  background: #fafbfc;
}

.server-card__time {
  font-size: 12px;
  color: #94a3b8;
}

.server-card__actions {
  display: flex;
  gap: 4px;
}
</style>
