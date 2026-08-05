<template>
  <el-dialog
    v-el-drag-dialog
    title="生成国标编码"
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
        <el-radio-group v-model="allVal[0].val" :disabled="allVal[0].lock" class="code-option-grid" @input="onProvinceChange">
          <el-radio
            v-for="item in filteredRegionList"
            :key="item.deviceId"
            :label="item.deviceId"
            class="code-option"
          >
            <span class="code-option-code">{{ item.deviceId }}</span>
            <span class="code-option-name">{{ item.name }}</span>
          </el-radio>
        </el-radio-group>
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
        <el-radio-group v-model="allVal[1].val" :disabled="allVal[1].lock" class="code-option-grid" @input="onCityChange">
          <el-radio
            v-for="item in filteredRegionList"
            :key="item.deviceId"
            :label="item.deviceId.substring(2)"
            class="code-option"
          >
            <span class="code-option-code">{{ item.deviceId.substring(2) }}</span>
            <span class="code-option-name">{{ item.name }}</span>
          </el-radio>
        </el-radio-group>
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
        <el-radio-group v-model="allVal[2].val" :disabled="allVal[2].lock" class="code-option-grid" @input="onDistrictChange">
          <el-radio
            v-for="item in filteredRegionList"
            :key="item.deviceId"
            :label="item.deviceId.substring(4)"
            class="code-option"
          >
            <span class="code-option-code">{{ item.deviceId.substring(4) }}</span>
            <span class="code-option-name">{{ item.name }}</span>
          </el-radio>
        </el-radio-group>
      </el-tab-pane>
      <el-tab-pane name="3">
        <div slot="label">
          <div class="show-code-item">{{ allVal[3].val || '--' }}</div>
          <div class="show-code-label">{{ allVal[3].meaning }}</div>
        </div>
        <div class="code-toolbar">
          <span class="code-hint">基层接入单位编码为两位数字（01–99），按序号排列</span>
        </div>
        <el-radio-group v-model="allVal[3].val" :disabled="allVal[3].lock" class="code-option-grid code-option-grid--compact">
          <el-radio
            v-for="code in baseUnitCodes"
            :key="'base-' + code"
            :label="code"
            class="code-option"
          >
            <span class="code-option-code">{{ code }}</span>
          </el-radio>
        </el-radio-group>
      </el-tab-pane>
      <el-tab-pane name="4">
        <div slot="label">
          <div class="show-code-item">{{ allVal[4].val || '--' }}</div>
          <div class="show-code-label">{{ allVal[4].meaning }}</div>
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
          <span class="code-hint">行业编码按 code 升序</span>
        </div>
        <el-radio-group v-model="allVal[4].val" :disabled="allVal[4].lock" class="code-option-grid">
          <el-radio
            v-for="item in filteredIndustryList"
            :key="item.code"
            :label="item.code"
            class="code-option"
          >
            <span class="code-option-code">{{ item.code }}</span>
            <span class="code-option-name">{{ item.name }}</span>
          </el-radio>
        </el-radio-group>
      </el-tab-pane>
      <el-tab-pane name="5">
        <div slot="label">
          <div class="show-code-item" :class="{ 'is-locked-code': allVal[5].lock }">{{ allVal[5].val || '--' }}</div>
          <div class="show-code-label">{{ allVal[5].meaning }}</div>
        </div>
        <el-alert
          v-if="allVal[5].lock"
          :title="lockedTypeTitle"
          type="warning"
          :closable="false"
          show-icon
          class="locked-type-alert"
        >
          <div class="locked-type-body">
            <p>当前在<strong>业务分组</strong>里生成的是<strong>树节点编号</strong>，不是摄像机编号。</p>
            <ul>
              <li><strong>215</strong>：业务分组（容器，不能挂通道）</li>
              <li><strong>216</strong>：虚拟组织（可挂通道）</li>
              <li><strong>131</strong>：摄像机类型，属于通道/设备，不能填在这里</li>
            </ul>
            <p>摄像机请在本节点保存后，选中该虚拟组织，再点右侧「添加通道」挂上去。</p>
            <p class="locked-type-current">本段已锁定为 <strong>{{ allVal[5].val }}</strong>，请点上方其它分段继续，或直接点「保存」。</p>
          </div>
        </el-alert>
        <template v-else>
          <div class="code-toolbar">
            <el-input
              v-model="filterKeyword"
              size="small"
              clearable
              placeholder="按名称或编码筛选"
              prefix-icon="el-icon-search"
              style="width: 16rem"
            />
            <span class="code-hint">类型编码按 code 升序</span>
          </div>
          <el-radio-group v-model="allVal[5].val" class="code-option-grid">
            <el-radio
              v-for="item in filteredDeviceTypeList"
              :key="item.code"
              :label="item.code"
              class="code-option"
            >
              <span class="code-option-code">{{ item.code }}</span>
              <span class="code-option-name">{{ item.name }}</span>
            </el-radio>
          </el-radio-group>
        </template>
      </el-tab-pane>
      <el-tab-pane name="6">
        <div slot="label">
          <div class="show-code-item">{{ allVal[6].val || '--' }}</div>
          <div class="show-code-label">{{ allVal[6].meaning }}</div>
        </div>
        <div class="code-toolbar">
          <span class="code-hint">网络标识按 code 升序</span>
        </div>
        <el-radio-group v-model="allVal[6].val" :disabled="allVal[6].lock" class="code-option-grid">
          <el-radio
            v-for="item in sortedNetworkList"
            :key="item.code"
            :label="item.code"
            class="code-option"
          >
            <span class="code-option-code">{{ item.code }}</span>
            <span class="code-option-name">{{ item.name }}</span>
          </el-radio>
        </el-radio-group>
      </el-tab-pane>
      <el-tab-pane name="7">
        <div slot="label">
          <div class="show-code-item">{{ allVal[7].val || '--' }}</div>
          <div class="show-code-label">{{ allVal[7].meaning }}</div>
        </div>
        <div class="code-toolbar">
          <span class="code-hint">设备/用户序号为六位数字</span>
        </div>
        <el-input
          v-model="allVal[7].val"
          type="text"
          placeholder="请输入六位序号"
          maxlength="6"
          :disabled="allVal[7].lock"
          show-word-limit
          style="width: 16rem"
        />
      </el-tab-pane>
    </el-tabs>
    <el-form class="region-code-form region-code-form--single">
      <el-form-item class="region-code-actions">
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

function sortByCode(list, key) {
  return (list || []).slice().sort((a, b) => {
    return String(a[key]).localeCompare(String(b[key]), 'en')
  })
}

function filterByKeyword(list, keyCode, keyName, kw) {
  const keyword = (kw || '').trim().toLowerCase()
  if (!keyword) return list
  return list.filter(item => {
    const code = String(item[keyCode] || '').toLowerCase()
    const name = String(item[keyName] || '').toLowerCase()
    return code.indexOf(keyword) !== -1 || name.indexOf(keyword) !== -1
  })
}

export default {
  directives: { elDragDialog },
  props: {},
  data() {
    return {
      showVideoDialog: false,
      activeKey: '0',
      filterKeyword: '',
      allVal: [
        { meaning: '省级编码', val: '11', lock: false },
        { meaning: '市级编码', val: '01', lock: false },
        { meaning: '区级编码', val: '01', lock: false },
        { meaning: '基层接入单位编码', val: '01', lock: false },
        { meaning: '行业编码', val: '00', lock: false },
        { meaning: '类型编码', val: '132', lock: false },
        { meaning: '网络标识编码', val: '0', lock: false },
        { meaning: '设备/用户序号', val: '000001', lock: false }
      ],
      regionList: [],
      deviceTypeList: [],
      industryCodeTypeList: [],
      networkIdentificationTypeList: [],
      endCallBck: null,
      baseUnitCodes: Array.from({ length: 99 }, (_, i) => pad2(i + 1))
    }
  },
  computed: {
    filteredRegionList() {
      return filterByKeyword(sortByCode(this.regionList, 'deviceId'), 'deviceId', 'name', this.filterKeyword)
    },
    filteredIndustryList() {
      return filterByKeyword(sortByCode(this.industryCodeTypeList, 'code'), 'code', 'name', this.filterKeyword)
    },
    filteredDeviceTypeList() {
      return filterByKeyword(sortByCode(this.deviceTypeList, 'code'), 'code', 'name', this.filterKeyword)
    },
    sortedNetworkList() {
      return sortByCode(this.networkIdentificationTypeList, 'code')
    },
    lockedTypeTitle() {
      const code = this.allVal[5] && this.allVal[5].val
      if (code === '215') {
        return '类型位已锁定为 215（业务分组），不能改成摄像机等设备类型'
      }
      if (code === '216') {
        return '类型位已锁定为 216（虚拟组织），不能改成摄像机等设备类型'
      }
      return '类型位已锁定，当前场景不允许修改'
    }
  },
  methods: {
    openDialog: function(endCallBck, code, lockIndex, lockContent) {
      this.showVideoDialog = true
      this.activeKey = '0'
      this.filterKeyword = ''
      this.regionList = []
      this.allVal = [
        { meaning: '省级编码', val: '11', lock: false },
        { meaning: '市级编码', val: '01', lock: false },
        { meaning: '区级编码', val: '01', lock: false },
        { meaning: '基层接入单位编码', val: '01', lock: false },
        { meaning: '行业编码', val: '00', lock: false },
        { meaning: '类型编码', val: '132', lock: false },
        { meaning: '网络标识编码', val: '0', lock: false },
        { meaning: '设备/用户序号', val: '000001', lock: false }
      ]

      this.getRegionList()
      if (typeof code !== 'undefined' && code && code.length === 20) {
        this.allVal[0].val = code.substring(0, 2)
        this.allVal[1].val = code.substring(2, 4)
        this.allVal[2].val = code.substring(4, 6)
        this.allVal[3].val = code.substring(6, 8)
        this.allVal[4].val = code.substring(8, 10)
        this.allVal[5].val = code.substring(10, 13)
        this.allVal[6].val = code.substring(13, 14)
        this.allVal[7].val = code.substring(14)
      }
      if (typeof lockIndex !== 'undefined') {
        this.allVal[lockIndex].lock = true
        this.allVal[lockIndex].val = lockContent
      }
      this.endCallBck = endCallBck
    },
    getRegionList: function() {
      this.filterKeyword = ''
      if (this.activeKey === '0' || this.activeKey === '1' || this.activeKey === '2') {
        let parent = ''
        if (this.activeKey === '1') {
          parent = this.allVal[0].val
        }
        if (this.activeKey === '2') {
          parent = this.allVal[0].val + this.allVal[1].val
        }
        if (this.activeKey !== '0' && parent === '') {
          this.$message.error({
            showClose: true,
            message: '请先选择上级行政区划'
          })
        }
        this.queryChildList(parent)
      } else if (this.activeKey === '4') {
        this.queryIndustryCodeList()
      } else if (this.activeKey === '5') {
        this.queryDeviceTypeList()
      } else if (this.activeKey === '6') {
        this.queryNetworkIdentificationTypeList()
      }
    },
    onProvinceChange: function() {
      this.allVal[1].val = '01'
      this.allVal[2].val = '01'
    },
    onCityChange: function() {
      this.allVal[2].val = '01'
    },
    onDistrictChange: function() {},
    queryChildList: function(parent) {
      this.regionList = []
      this.$store.dispatch('region/queryChildListInBase', parent)
        .then(data => {
          this.regionList = sortByCode(data, 'deviceId')
        })
        .catch((error) => {
          this.$message.error({
            showClose: true,
            message: error
          })
        })
    },
    queryIndustryCodeList: function() {
      this.industryCodeTypeList = []
      this.$store.dispatch('commonChanel/getIndustryList')
        .then(data => {
          this.industryCodeTypeList = sortByCode(data, 'code')
        })
        .catch((error) => {
          this.$message.error({
            showClose: true,
            message: error
          })
        })
    },
    queryDeviceTypeList: function() {
      this.deviceTypeList = []
      this.$store.dispatch('commonChanel/getTypeList')
        .then(data => {
          this.deviceTypeList = sortByCode(data, 'code')
        })
        .catch((error) => {
          this.$message.error({
            showClose: true,
            message: error
          })
        })
    },
    queryNetworkIdentificationTypeList: function() {
      this.networkIdentificationTypeList = []
      this.$store.dispatch('commonChanel/getNetworkIdentificationList')
        .then(data => {
          this.networkIdentificationTypeList = sortByCode(data, 'code')
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
    handleOk: function() {
      const code =
        this.allVal[0].val +
        this.allVal[1].val +
        this.allVal[2].val +
        this.allVal[3].val +
        this.allVal[4].val +
        this.allVal[5].val +
        this.allVal[6].val +
        this.allVal[7].val
      if (this.endCallBck) {
        this.endCallBck(code)
      }
      this.showVideoDialog = false
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
.show-code-item.is-locked-code {
  color: #e6a23c;
}
.show-code-label {
  text-align: center;
  font-size: 12px;
  color: #606266;
}
.locked-type-alert {
  margin: 4px 0 12px;
}
.locked-type-body {
  line-height: 1.7;
  color: #606266;
}
.locked-type-body p {
  margin: 0 0 8px;
}
.locked-type-body ul {
  margin: 0 0 8px;
  padding-left: 1.2rem;
}
.locked-type-current {
  color: #e6a23c;
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
  width: 100%;
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
  padding: 1rem 2rem 0;
  grid-template-columns: 1fr 1fr auto;
  gap: 1rem;
  align-items: end;
}
.region-code-form--single {
  grid-template-columns: 1fr;
}
.region-code-actions {
  margin-bottom: 0;
  text-align: right;
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
