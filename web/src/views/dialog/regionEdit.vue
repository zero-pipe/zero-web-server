<template>
  <el-dialog
    v-el-drag-dialog
    title="生成行政区划编码"
    width="72rem"
    top="2rem"
    center
    :append-to-body="true"
    :close-on-click-modal="false"
    :visible.sync="showVideoDialog"
    :destroy-on-close="false"
  >
    <el-tabs v-model="activeKey" class="region-code-tabs" @tab-click="getRegionList">
      <el-tab-pane name="0">
        <div slot="label">
          <div class="show-code-item">{{ allVal[0].val || '--' }}</div>
          <div class="show-code-label">{{ allVal[0].meaning }}</div>
        </div>
        <div class="code-toolbar">
          <el-input
            v-model="filterKeyword"
            size="small"
            clearable
            placeholder="按名称或编码筛选"
            prefix-icon="el-icon-search"
            style="width: 16rem"
          />
          <span class="code-hint">按编码升序排列</span>
        </div>
        <div class="code-option-grid">
          <el-radio
            v-for="item in filteredRegionList"
            :key="item.deviceId"
            v-model="allVal[0].val"
            :label="item.deviceId"
            class="code-option"
            @input="onProvinceChange(item)"
          >
            <span class="code-option-code">{{ item.deviceId }}</span>
            <span class="code-option-name">{{ item.name }}</span>
          </el-radio>
        </div>
      </el-tab-pane>
      <el-tab-pane name="1">
        <div slot="label">
          <div class="show-code-item">{{ allVal[1].val || '--' }}</div>
          <div class="show-code-label">{{ allVal[1].meaning }}</div>
        </div>
        <div class="code-toolbar">
          <el-input
            v-model="filterKeyword"
            size="small"
            clearable
            placeholder="按名称或编码筛选"
            prefix-icon="el-icon-search"
            style="width: 16rem"
          />
          <span class="code-hint">按编码升序排列</span>
        </div>
        <div class="code-option-grid">
          <el-radio
            v-model="allVal[1].val"
            label=""
            class="code-option"
            @input="onCityChange(null)"
          >
            <span class="code-option-name">不添加</span>
          </el-radio>
          <el-radio
            v-for="item in filteredRegionList"
            :key="item.deviceId"
            v-model="allVal[1].val"
            :label="item.deviceId.substring(2)"
            class="code-option"
            @input="onCityChange(item)"
          >
            <span class="code-option-code">{{ item.deviceId.substring(2) }}</span>
            <span class="code-option-name">{{ item.name }}</span>
          </el-radio>
        </div>
      </el-tab-pane>
      <el-tab-pane name="2">
        <div slot="label">
          <div class="show-code-item">{{ allVal[2].val || '--' }}</div>
          <div class="show-code-label">{{ allVal[2].meaning }}</div>
        </div>
        <div class="code-toolbar">
          <el-input
            v-model="filterKeyword"
            size="small"
            clearable
            placeholder="按名称或编码筛选"
            prefix-icon="el-icon-search"
            style="width: 16rem"
          />
          <span class="code-hint">按编码升序排列</span>
        </div>
        <div class="code-option-grid">
          <el-radio
            v-model="allVal[2].val"
            label=""
            class="code-option"
            @input="onDistrictChange(null)"
          >
            <span class="code-option-name">不添加</span>
          </el-radio>
          <el-radio
            v-for="item in filteredRegionList"
            :key="item.deviceId"
            v-model="allVal[2].val"
            :label="item.deviceId.substring(4)"
            class="code-option"
            @input="onDistrictChange(item)"
          >
            <span class="code-option-code">{{ item.deviceId.substring(4) }}</span>
            <span class="code-option-name">{{ item.name }}</span>
          </el-radio>
        </div>
      </el-tab-pane>
      <el-tab-pane name="3">
        <div slot="label">
          <div class="show-code-item">{{ allVal[3].val || '--' }}</div>
          <div class="show-code-label">{{ allVal[3].meaning }}</div>
        </div>
        <div class="code-toolbar">
          <span class="code-hint">基层接入单位编码为两位数字（01–99），按序号排列；可不添加</span>
        </div>
        <div class="code-option-grid code-option-grid--compact">
          <el-radio
            v-model="allVal[3].val"
            label=""
            class="code-option"
            @input="deviceChange(null)"
          >
            <span class="code-option-name">不添加</span>
          </el-radio>
          <el-radio
            v-for="code in baseUnitCodes"
            :key="'base-' + code"
            v-model="allVal[3].val"
            :label="code"
            class="code-option"
            @input="deviceChange(null)"
          >
            <span class="code-option-code">{{ code }}</span>
          </el-radio>
        </div>
      </el-tab-pane>
    </el-tabs>
    <el-form ref="form" class="region-code-form" label-position="top" size="mini">
      <el-form-item label="名称" prop="name">
        <el-input v-model="form.name" autocomplete="off" />
      </el-form-item>
      <el-form-item label="编号" prop="deviceId">
        <el-input v-model="form.deviceId" autocomplete="off" />
      </el-form-item>
      <el-form-item label="　" class="region-code-actions">
        <el-button type="primary" @click="handleOk">保存</el-button>
        <el-button @click="closeModel">取消</el-button>
      </el-form-item>
    </el-form>
  </el-dialog>
</template>

<script>

import elDragDialog from '@/directive/el-drag-dialog'

function pad2(n) {
  return n < 10 ? '0' + n : '' + n
}

export default {
  directives: { elDragDialog },
  props: {},
  data() {
    return {
      showVideoDialog: false,
      activeKey: '0',
      filterKeyword: '',
      form: {
        name: '',
        deviceId: '',
        parentId: ''
      },
      allVal: [
        { meaning: '省级编码', val: '' },
        { meaning: '市级编码', val: '' },
        { meaning: '区级编码', val: '' },
        { meaning: '基层接入单位编码', val: '' }
      ],
      regionList: [],
      endCallBck: null,
      baseUnitCodes: Array.from({ length: 99 }, (_, i) => pad2(i + 1))
    }
  },
  computed: {
    filteredRegionList() {
      const list = (this.regionList || []).slice().sort((a, b) => {
        return String(a.deviceId).localeCompare(String(b.deviceId), 'en')
      })
      const kw = (this.filterKeyword || '').trim().toLowerCase()
      if (!kw) return list
      return list.filter(item => {
        const id = String(item.deviceId || '').toLowerCase()
        const name = String(item.name || '').toLowerCase()
        return id.indexOf(kw) !== -1 || name.indexOf(kw) !== -1
      })
    }
  },
  methods: {
    openDialog: function(endCallBck, region, code, lockContent) {
      this.showVideoDialog = true
      this.activeKey = '0'
      this.filterKeyword = ''
      this.regionList = []
      this.form = region
      this.allVal = [
        { meaning: '省级编码', val: '' },
        { meaning: '市级编码', val: '' },
        { meaning: '区级编码', val: '' },
        { meaning: '基层接入单位编码', val: '' }
      ]
      if (this.form.deviceId) {
        if (this.form.deviceId.length >= 2) {
          this.allVal[0].val = this.form.deviceId.substring(0, 2)
          this.activeKey = '0'
        }
        if (this.form.deviceId.length >= 4) {
          this.allVal[1].val = this.form.deviceId.substring(2, 4)
          this.activeKey = '1'
        }
        if (this.form.deviceId.length >= 6) {
          this.allVal[2].val = this.form.deviceId.substring(4, 6)
          this.activeKey = '2'
        }
        if (this.form.deviceId.length === 8) {
          this.allVal[3].val = this.form.deviceId.substring(6, 8)
          this.activeKey = '3'
        }
      } else if (this.form.parentDeviceId) {
        if (this.form.parentDeviceId.length >= 2) {
          this.allVal[0].val = this.form.parentDeviceId.substring(0, 2)
          this.activeKey = '1'
        }
        if (this.form.parentDeviceId.length >= 4) {
          this.allVal[1].val = this.form.parentDeviceId.substring(2, 4)
          this.activeKey = '2'
        }
        if (this.form.parentDeviceId.length >= 6) {
          this.allVal[2].val = this.form.parentDeviceId.substring(4, 6)
          this.activeKey = '3'
        }
      }

      this.getRegionList()
      this.endCallBck = endCallBck
    },
    getRegionList: function() {
      this.filterKeyword = ''
      if (this.activeKey === '0') {
        this.queryChildList()
        return
      }
      if (this.activeKey === '1' || this.activeKey === '2') {
        let parent = ''
        if (this.activeKey === '1') {
          parent = this.allVal[0].val
        }
        if (this.activeKey === '2') {
          parent = this.allVal[1].val === '' ? '' : (this.allVal[0].val + this.allVal[1].val)
        }
        if (parent === '') {
          this.regionList = []
          this.$message.error({
            showClose: true,
            message: '请先选择上级行政区划'
          })
          return
        }
        this.queryChildList(parent)
      }
    },
    queryChildList: function(parent) {
      this.regionList = []
      this.$store.dispatch('region/queryChildListInBase', parent)
        .then(data => {
          this.regionList = (data || []).slice().sort((a, b) => {
            return String(a.deviceId).localeCompare(String(b.deviceId), 'en')
          })
        })
        .catch((error) => {
          this.$message.error({
            showClose: true,
            message: error
          })
        })
    },
    closeModel: function() {
      this.showVideoDialog = false
    },
    onProvinceChange: function(item) {
      this.allVal[1].val = ''
      this.allVal[2].val = ''
      this.allVal[3].val = ''
      this.deviceChange(item)
    },
    onCityChange: function(item) {
      this.allVal[2].val = ''
      this.allVal[3].val = ''
      this.deviceChange(item)
    },
    onDistrictChange: function(item) {
      this.allVal[3].val = ''
      this.deviceChange(item)
    },
    deviceChange: function(item) {
      let code = this.allVal[0].val || ''
      if (this.allVal[1].val) {
        code += this.allVal[1].val
        if (this.allVal[2].val) {
          code += this.allVal[2].val
          if (this.allVal[3].val) {
            code += this.allVal[3].val
          }
        }
      }
      this.form.deviceId = code
      if (item && item.name) {
        this.form.name = item.name
      }
    },
    handleOk: function() {
      if (this.form.id) {
        this.$store.dispatch('region/update', this.form)
          .then(() => {
            if (typeof this.endCallBck === 'function') {
              this.endCallBck(this.form)
            }
            this.showVideoDialog = false
          })
          .catch((error) => {
            this.$message.error({
              showClose: true,
              message: error
            })
          })
      } else {
        this.$store.dispatch('region/add', this.form)
          .then(() => {
            if (typeof this.endCallBck === 'function') {
              this.endCallBck(this.form)
            }
            this.showVideoDialog = false
          })
          .catch((error) => {
            this.$message.error({
              showClose: true,
              message: error
            })
          })
      }
    }
  }
}
</script>

<style>
.show-code-item {
  text-align: center;
  font-size: 2.4rem;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}
.show-code-label {
  text-align: center;
  font-size: 12px;
  color: #606266;
}
.region-code-tabs {
  padding: 0 1rem;
}
.code-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}
.code-hint {
  font-size: 12px;
  color: #909399;
}
.code-option-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 6px 12px;
  max-height: 22rem;
  overflow-y: auto;
  padding: 4px 2px 8px;
}
.code-option-grid--compact {
  grid-template-columns: repeat(10, minmax(0, 1fr));
}
.code-option {
  margin-right: 0 !important;
  margin-left: 0 !important;
  display: flex;
  align-items: center;
  line-height: 1.6;
  white-space: nowrap;
  overflow: hidden;
}
.code-option .el-radio__label {
  display: inline-flex;
  align-items: baseline;
  gap: 6px;
  padding-left: 6px;
  overflow: hidden;
}
.code-option-code {
  font-variant-numeric: tabular-nums;
  font-family: Consolas, Monaco, monospace;
  color: #303133;
  min-width: 2.2em;
}
.code-option-name {
  color: #606266;
  overflow: hidden;
  text-overflow: ellipsis;
}
.region-code-form {
  display: grid;
  padding: 1rem 2rem 0.5rem;
  grid-template-columns: 1fr 1fr auto;
  gap: 1rem;
  align-items: end;
}
.region-code-form .el-form-item {
  margin-bottom: 0;
}
.region-code-actions {
  text-align: right;
}
.region-code-actions >>> .el-form-item__label {
  visibility: hidden;
  user-select: none;
}
@media (max-width: 1100px) {
  .code-option-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
  .code-option-grid--compact {
    grid-template-columns: repeat(8, minmax(0, 1fr));
  }
}
</style>
