<template>
  <div id="groupEdit" v-loading="loading">
    <el-dialog
      v-el-drag-dialog
      :title="dialogTitle"
      width="560px"
      top="2rem"
      :append-to-body="true"
      :close-on-click-modal="false"
      :visible.sync="showDialog"
      :destroy-on-close="true"
      @close="close()"
    >
      <el-form
        ref="form"
        :model="group"
        :rules="rules"
        label-width="108px"
        class="group-edit-form"
        @submit.native.prevent
      >
        <el-form-item label="节点类型">
          <el-tag :type="isBusinessGroupNode ? 'warning' : 'success'" size="small">
            {{ typeCodeLabel }}
          </el-tag>
        </el-form-item>

        <el-form-item label="节点名称" prop="name">
          <el-input
            v-model="group.name"
            clearable
            maxlength="64"
            placeholder="请输入名称"
          />
        </el-form-item>

        <el-form-item label="行政区划" prop="civilCode">
          <div class="civil-code-row">
            <el-input
              :value="civilCodeDisplay"
              readonly
              placeholder="可选"
            >
              <el-button slot="append" @click="buildCivilCode">选择</el-button>
            </el-input>
            <el-button
              v-if="group.civilCode"
              type="text"
              @click="clearCivilCode"
            >清空</el-button>
          </div>
        </el-form-item>

        <el-form-item label="国标编号" prop="deviceId">
          <el-input
            v-model="group.deviceId"
            placeholder="20位国标编码"
            maxlength="20"
            show-word-limit
          >
            <el-button slot="append" type="primary" @click="buildDeviceIdCode">生成</el-button>
          </el-input>
        </el-form-item>

        <el-form-item class="group-edit-actions">
          <el-button type="primary" @click="onSubmit">确认</el-button>
          <el-button @click="close">取消</el-button>
        </el-form-item>
      </el-form>
    </el-dialog>
    <channelCode ref="channelCode" />
    <chooseCivilCode ref="chooseCivilCode" />
  </div>
</template>

<script>
import channelCode from './channelCode.vue'
import ChooseCivilCode from './chooseCivilCode.vue'
import elDragDialog from '@/directive/el-drag-dialog'

export default {
  name: 'GroupEdit',
  directives: { elDragDialog },
  components: { ChooseCivilCode, channelCode },
  props: [],
  data() {
    return {
      submitCallback: null,
      showDialog: false,
      loading: false,
      civilCodeName: '',
      group: {
        id: 0,
        deviceId: '',
        name: '',
        parentDeviceId: '',
        businessGroup: '',
        civilCode: '',
        platformId: '',
        parentId: null
      },
      rules: {
        name: [
          { required: true, message: '请填写节点名称', trigger: 'blur' }
        ],
        deviceId: [
          { required: true, message: '请填写国标编号', trigger: 'blur' },
          {
            validator: (rule, value, callback) => {
              const v = (value || '').trim()
              if (!v) {
                callback()
                return
              }
              if (v.length !== 20) {
                callback(new Error('国标编号必须是20位'))
                return
              }
              const type = v.substring(10, 13)
              if (type !== '215' && type !== '216') {
                callback(new Error('第11-13位须为215或216'))
                return
              }
              callback()
            },
            trigger: 'blur'
          }
        ]
      }
    }
  },
  computed: {
    isEdit() {
      return !!(this.group && this.group.id)
    },
    isBusinessGroupNode() {
      const id = (this.group.deviceId || '').trim()
      if (id.length >= 13) {
        return id.substring(10, 13) === '215'
      }
      if (this.isEdit) {
        return !this.group.businessGroup || this.group.businessGroup === this.group.deviceId
      }
      return !this.group.businessGroup
    },
    lockedTypeCode() {
      return this.isBusinessGroupNode ? '215' : '216'
    },
    typeCodeLabel() {
      return this.isBusinessGroupNode ? '业务分组 (215)' : '虚拟组织 (216)'
    },
    dialogTitle() {
      if (this.isEdit) {
        return '编辑节点'
      }
      return this.isBusinessGroupNode ? '新建业务分组' : '新建虚拟组织'
    },
    civilCodeDisplay() {
      if (!this.group.civilCode) {
        return ''
      }
      if (this.civilCodeName) {
        return `${this.civilCodeName}（${this.group.civilCode}）`
      }
      return this.group.civilCode
    }
  },
  methods: {
    openDialog: function(group, callback) {
      this.civilCodeName = ''
      if (group) {
        this.group = Object.assign({
          id: 0,
          deviceId: '',
          name: '',
          parentDeviceId: '',
          businessGroup: '',
          civilCode: '',
          platformId: '',
          parentId: null
        }, group)
      }
      this.showDialog = true
      this.submitCallback = callback
      this.$nextTick(() => {
        if (this.$refs.form) {
          this.$refs.form.clearValidate()
        }
      })
    },
    onSubmit: function() {
      this.$refs.form.validate(valid => {
        if (!valid) return
        const action = this.group.id ? 'group/update' : 'group/add'
        this.loading = true
        this.$store.dispatch(action, this.group)
          .then(() => {
            this.$message.success({
              showClose: true,
              message: '保存成功'
            })
            if (this.submitCallback) this.submitCallback(this.group)
            if (!this.group.id) {
              this.close()
            }
          })
          .catch((error) => {
            this.$message({
              showClose: true,
              message: error,
              type: 'error'
            })
          })
          .finally(() => {
            this.loading = false
          })
      })
    },
    buildSeedCode: function(typeCode) {
      let civil = String(this.group.civilCode || '').replace(/\D/g, '')
      if (civil.length > 8) {
        civil = civil.substring(0, 8)
      }
      if (civil.length === 0) {
        civil = '11010101'
      } else if (civil.length === 2) {
        civil += '010101'
      } else if (civil.length === 4) {
        civil += '0101'
      } else if (civil.length === 6) {
        civil += '01'
      } else if (civil.length === 7) {
        civil += '0'
      }
      return civil + '00' + typeCode + '0' + '000001'
    },
    buildDeviceIdCode: function() {
      const lockContent = this.lockedTypeCode
      const seed = (this.group.deviceId && this.group.deviceId.length === 20)
        ? this.group.deviceId
        : this.buildSeedCode(lockContent)
      this.$refs.channelCode.openDialog(code => {
        this.group.deviceId = code
        this.$nextTick(() => {
          if (this.$refs.form) this.$refs.form.validateField('deviceId')
        })
      }, seed, 5, lockContent)
    },
    buildCivilCode: function() {
      this.$refs.chooseCivilCode.openDialog((code, name) => {
        if (!code) return
        this.group.civilCode = code
        this.civilCodeName = name || ''
      })
    },
    clearCivilCode: function() {
      this.group.civilCode = ''
      this.civilCodeName = ''
    },
    close: function() {
      this.showDialog = false
    }
  }
}
</script>

<style scoped>
.group-edit-form {
  margin-right: 0;
  padding-right: 4px;
}
.group-edit-actions {
  margin-bottom: 0;
  text-align: right;
}
.civil-code-row {
  display: flex;
  align-items: center;
  gap: 4px;
}
.civil-code-row .el-input {
  flex: 1;
}
</style>
