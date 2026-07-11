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
      <el-alert
        :title="guideTitle"
        type="info"
        :closable="false"
        show-icon
        class="group-edit-guide"
      >
        <ol class="group-edit-steps">
          <li>先填写<strong>节点名称</strong>（必填）</li>
          <li>行政区划<strong>可选</strong>：点「选择」关联已有区划；没有可留空</li>
          <li>点「生成国标编号」按向导选省市区等，系统会自动锁为 {{ typeCodeLabel }}</li>
          <li>确认编号无误后点「确认」保存</li>
        </ol>
        <p class="group-edit-tip">{{ guideTip }}</p>
      </el-alert>

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
          <span class="group-type-desc">{{ typeDesc }}</span>
        </el-form-item>

        <el-form-item label="节点名称" prop="name">
          <el-input
            v-model="group.name"
            clearable
            maxlength="64"
            placeholder="例如：一号楼、东区监控组"
          />
        </el-form-item>

        <el-form-item label="行政区划" prop="civilCode">
          <div class="civil-code-row">
            <el-input
              :value="civilCodeDisplay"
              readonly
              placeholder="可选，未选不影响保存"
            >
              <el-button slot="append" @click="buildCivilCode">选择</el-button>
            </el-input>
            <el-button
              v-if="group.civilCode"
              type="text"
              @click="clearCivilCode"
            >清空</el-button>
          </div>
          <div class="field-hint">用于关联行政区划树中的节点；不选也可直接生成编号</div>
        </el-form-item>

        <el-form-item label="国标编号" prop="deviceId">
          <el-input
            v-model="group.deviceId"
            placeholder="20位国标编码，请点右侧生成"
            maxlength="20"
            show-word-limit
          >
            <el-button slot="append" type="primary" @click="buildDeviceIdCode">生成国标编号</el-button>
          </el-input>
          <div class="field-hint">
            第 11–13 位固定为 {{ lockedTypeCode }}（{{ isBusinessGroupNode ? '业务分组' : '虚拟组织' }}），勿手动改成其他类型
          </div>
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
          { required: true, message: '请先生成或填写20位国标编号', trigger: 'blur' },
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
                callback(new Error('第11-13位须为215（业务分组）或216（虚拟组织）'))
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
    // 新建时：没有所属业务分组编号 → 正在建业务分组(215)
    // 编辑时：看编号类型位
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
    typeDesc() {
      if (this.isBusinessGroupNode) {
        return '顶层分组容器，不能直接挂通道'
      }
      return '可挂通道的组织节点'
    },
    dialogTitle() {
      if (this.isEdit) {
        return '编辑节点'
      }
      return this.isBusinessGroupNode ? '新建业务分组' : '新建虚拟组织'
    },
    guideTitle() {
      if (this.isEdit) {
        return '按需修改名称或重新生成编号'
      }
      if (this.isBusinessGroupNode) {
        return '推荐操作顺序（当前：新建业务分组）'
      }
      return '推荐操作顺序（当前：新建虚拟组织）'
    },
    guideTip() {
      if (this.isBusinessGroupNode) {
        return '建好后请在该分组下再「新建节点」创建虚拟组织(216)，通道只能挂在 216 上。'
      }
      return '保存后选中本节点，即可在右侧「添加通道」。'
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
        if (!valid) {
          this.$message.warning({
            showClose: true,
            message: '请先按提示完成必填项：名称 + 20位国标编号'
          })
          return
        }
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
        if (!code) {
          this.$message.warning({
            showClose: true,
            message: '未选择行政区划节点，可留空继续'
          })
          return
        }
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
.group-edit-guide {
  margin-bottom: 16px;
}
.group-edit-steps {
  margin: 6px 0 0;
  padding-left: 1.2rem;
  line-height: 1.7;
  color: #606266;
}
.group-edit-tip {
  margin: 8px 0 0;
  color: #909399;
  font-size: 12px;
}
.group-edit-form {
  margin-right: 0;
  padding-right: 4px;
}
.group-type-desc {
  margin-left: 8px;
  color: #909399;
  font-size: 12px;
}
.field-hint {
  margin-top: 4px;
  line-height: 1.4;
  color: #909399;
  font-size: 12px;
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
