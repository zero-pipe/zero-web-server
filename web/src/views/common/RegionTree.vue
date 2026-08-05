<template>
  <div id="regionTree" class="org-tree-panel">
    <div class="org-tree-search">
      <el-input size="small" v-model="searchStr" @input="searchChange" suffix-icon="el-icon-search" placeholder="请输入搜索内容" clearable />
    </div>
    <div v-if="!searchStr" class="org-tree-body">
      <div v-if="edit" class="org-tree-toolbar">
        <div class="org-tree-actions">
          <el-button
            type="primary"
            size="mini"
            icon="el-icon-folder-add"
            :disabled="!canAddNode"
            @click="toolbarAddNode"
          >新建</el-button>
          <el-button
            v-if="enableAddChannel"
            type="primary"
            plain
            size="mini"
            icon="el-icon-plus"
            :disabled="!canMountChannel"
            @click="toolbarAddChannel"
          >添加通道</el-button>
          <el-dropdown size="mini" @command="toolbarMoreCommand">
            <el-button size="mini" :disabled="!activeCatalogNode">
              操作<i class="el-icon-arrow-down el-icon--right" />
            </el-button>
            <el-dropdown-menu slot="dropdown">
              <el-dropdown-item command="refresh" :disabled="!activeCatalogNode">刷新目录</el-dropdown-item>
              <el-dropdown-item command="edit" :disabled="!canEditNode">编辑目录</el-dropdown-item>
              <el-dropdown-item command="delete" :disabled="!canEditNode">删除目录</el-dropdown-item>
            </el-dropdown-menu>
          </el-dropdown>
        </div>
        <label class="org-tree-toolbar-label">
          <el-checkbox v-model="showCode" />
          编号
        </label>
      </div>

      <vue-easy-tree
        ref="veTree"
        class="flow-tree"
        node-key="treeId"
        :height="'100%'"
        lazy
        :load="loadNode"
        :data="treeData"
        :props="props"
        :default-expanded-keys="['']"
        @node-contextmenu="contextmenuEventHandler"
        @node-click="nodeClickHandler"
      >
        <template v-slot:default="{ node, data }" class="custom-tree-node">
          <span class="custom-tree-node">
            <span
              v-if="node.data.type === 0 && chooseId !== node.data.deviceId"
              style="color: #409EFF"
              class="iconfont icon-bianzubeifen3"
            />
            <span
              v-if="node.data.type === 0 && chooseId === node.data.deviceId"
              style="color: #c60135;"
              class="iconfont icon-bianzubeifen3"
            />
            <span
              v-if="node.data.type === 1 && node.data.status === 'ON'"
              style="color: #409EFF"
              class="iconfont icon-shexiangtou2"
            />
            <span
              v-if="node.data.type === 1 && node.data.status !== 'ON'"
              style="color: #808181"
              class="iconfont icon-shexiangtou2"
            />
            <span
              style=" padding-left: 1px"
              :title="node.data.deviceId"
            >{{ node.label }}</span>
            <span v-if="node.data.deviceId !=='' && showCode">（编号：{{ node.data.deviceId }}）</span>
            <span v-if="node.data.longitude && showPosition" class="iconfont icon-gps"></span>
          </span>
        </template>
      </vue-easy-tree>
    </div>
    <div v-if="searchStr" class="org-tree-body org-tree-search-result">
      <ul v-if="regionList.length > 0" class="org-result-list">
        <li v-for="item in regionList" :key="item.id" class="channel-list-li" @click="listClickHandler(item)">
          <span
            v-if="chooseId !== item.deviceId"
            style="color: #409EFF;  font-size: 20px"
            class="iconfont icon-bianzubeifen3"
          />
          <span
            v-if="chooseId === item.deviceId"
            style="color: #c60135;  font-size: 20px"
            class="iconfont icon-bianzubeifen3"
          />
          <div>
            <div style="margin-left: 4px; margin-bottom: 3px; font-size: 15px">{{item.name}}</div>
            <div style="margin-left: 4px; font-size: 13px; color: #808181">{{item.deviceId}}</div>
          </div>
        </li>
      </ul>

      <ul v-if="channelList.length > 0" class="org-result-list">
        <li v-for="item in channelList" :key="item.id" class="channel-list-li" @click="channelLstClickHandler(item)" @contextmenu.prevent="contextmenuEventHandlerForLi($event, item)">
          <span
            v-if="item.gbStatus === 'ON'"
            style="color: #409EFF; font-size: 20px"
            class="iconfont icon-shexiangtou2"
          />
          <span
            v-if="item.gbStatus !== 'ON'"
            style="color: #808181; font-size: 20px"
            class="iconfont icon-shexiangtou2"
          />
          <div>
            <div style="margin-left: 4px; margin-bottom: 3px; font-size: 15px">{{item.gbName}}</div>
            <div style="margin-left: 4px; font-size: 13px; color: #808181">{{item.gbDeviceId}}</div>
          </div>

        </li>
      </ul>
      <div v-if="this.currentPage * this.count < this.total" style="text-align: center;">
        <el-button type="text" @click="loadListMore">加载更多</el-button>
      </div>
    </div>
    <regionEdit ref="regionEdit" />
    <gbDeviceSelect ref="gbDeviceSelect" />
    <GbChannelSelect ref="gbChannelSelect" data-type="civilCode" />
  </div>
</template>

<script>
import VueEasyTree from '@wchbrad/vue-easy-tree'
import regionEdit from './../dialog/regionEdit'
import gbDeviceSelect from './../dialog/GbDeviceSelect'
import GbChannelSelect from '../dialog/GbChannelSelect.vue'

export default {
  name: 'DeviceTree',
  components: {
    GbChannelSelect,
    VueEasyTree, regionEdit, gbDeviceSelect
  },
  props: ['edit', 'enableAddChannel', 'onChannelChange', 'showHeader', 'hasChannel', 'addChannelToCivilCode', 'treeHeight', 'showPosition', 'contextmenu'],
  data() {
    return {
      props: {
        label: 'name',
        children: 'children'
      },
      searchType: 0,
      showCode: false,
      showAlert: true,
      treeLimit: 50,
      searchStr: '',
      chooseId: '',
      activeNode: null,
      treeData: [],
      currentPage: this.defaultPage | 1,
      count: this.defaultCount | 15,
      total: 0,
      regionList: [],
      channelList: []
    }
  },
  computed: {
    activeCatalogNode() {
      return this.activeNode && this.activeNode.data && this.activeNode.data.type === 0
        ? this.activeNode
        : null
    },
    canAddNode() {
      return !!this.activeCatalogNode
    },
    canEditNode() {
      return !!(this.activeCatalogNode && this.activeCatalogNode.level > 1)
    },
    canMountChannel() {
      return !!(this.enableAddChannel && this.activeCatalogNode && this.activeCatalogNode.level > 1)
    },
    currentNodeShort() {
      if (!this.activeCatalogNode || !this.activeCatalogNode.data) return ''
      return this.activeCatalogNode.data.name || this.activeCatalogNode.data.deviceId || ''
    }
  },
  created() {
  },
  destroyed() {
    // if (this.jessibuca) {
    //   this.jessibuca.destroy();
    // }
    // this.playing = false;
    // this.loaded = false;
    // this.performance = "";
  },
  methods: {
    searchChange() {
      this.currentPage = 1
      this.total = 0
      if (this.edit) {
        this.regionList = []
        this.queryRegion()
      }else {
        this.channelList = []
        this.queryChannelList()
      }
    },
    loadListMore: function() {
      this.currentPage += 1
      if (this.edit) {
        this.queryRegion()
      }else {
        this.queryChannelList()
      }
    },
    queryRegion: function() {
      this.$store.dispatch('region/queryTree', {
        query: this.searchStr,
        page: this.currentPage,
        count: this.count
      }).then(data => {
        this.total = data.total
        this.regionList = this.regionList.concat(data.list)
      })
    },
    queryChannelList: function() {
      this.$store.dispatch('commonChanel/getList', {
        page: this.currentPage,
        count: this.count,
        query: this.searchStr
      }).then(data => {
        this.total = data.total
        this.channelList = this.channelList.concat(data.list)
      })
    },
    loadNode: function(node, resolve) {
      console.log(22222)
      if (node.level === 0) {
        resolve([{
          treeId: '',
          deviceId: '',
          name: '根资源组',
          isLeaf: false,
          type: 0
        }])
      } else if (node.data.type === 0) {
        if (node.data.leaf) {
          resolve([])
          return
        }
        this.$store.dispatch('region/getTreeList', {
          query: this.searchStr,
          parent: node.data.id,
          hasChannel: this.hasChannel
        })
          .then(data => {
            if (data.length > 0) {
              this.showAlert = false
            }
            if (data.length > this.treeLimit) {
              let subData = data.splice(0, this.treeLimit)
              subData.push({
                treeId: '---',
                deviceId: '---',
                name: '加载更多...',
                isLeaf: true,
                leaf: true,
                type: 100,
                children: [],
                nextData: data
              })
              resolve(subData)
            } else {
              resolve(data)
            }

          }).finally(() => {
            this.locading = false
          })
      } else {
        resolve([])
      }
    },
    reset: function() {
      this.$forceUpdate()
    },
    contextmenuEventHandler: function(event, data, node, element) {
      // 目录管理改走顶部按钮；右键仅保留通道节点的自定义菜单
      if (!this.contextmenu || !node || !node.data || node.data.type !== 1) {
        return
      }
      const allMenuItem = []
      for (let i = 0; i < this.contextmenu.length; i++) {
        const item = this.contextmenu[i]
        if (item.type === node.data.type) {
          allMenuItem.push({
            label: item.label,
            icon: item.icon,
            onClick: () => {
              item.onClick(event, data, node)
            }
          })
        }
      }
      if (allMenuItem.length === 0) {
        return
      }
      this.$contextmenu({
        items: allMenuItem,
        event,
        customClass: 'custom-class',
        zIndex: 3000
      })
      return false
    },
    removeRegion: function(id, node) {
      this.$store.dispatch('region/deleteRegion', node.data.id)
        .then((data) => {
          console.log('移除成功')
          this.$emit('onChannelChange', node.data.deviceId)
          node.parent.loaded = false
          node.parent.expand()
        })
        .catch((error) => {
          this.$message({
            showClose: true,
            message: error,
            type: 'error'
          })
        })
    },
    addChannelFormDevice: function(id, node) {
      this.$refs.gbDeviceSelect.openDialog((rows) => {
        const deviceIds = []
        for (let i = 0; i < rows.length; i++) {
          deviceIds.push(rows[i].id)
        }
        this.$store.dispatch('commonChanel/addDeviceToRegion', {
          civilCode: node.data.deviceId,
          deviceIds: deviceIds
        }).then((data) => {
          this.$message.success({
            showClose: true,
            message: '保存成功'
          })
          this.$emit('onChannelChange', node.data.deviceId)
          node.loaded = false
          node.expand()
        }).catch(function(error) {
          console.log(error)
        }).finally(() => {
          this.loading = false
        })
      })
    },
    removeChannelFormDevice: function(id, node) {
      this.$refs.gbDeviceSelect.openDialog((rows) => {
        const deviceIds = []
        for (let i = 0; i < rows.length; i++) {
          deviceIds.push(rows[i].id)
        }
        this.$store.dispatch('commonChanel/deleteDeviceFromRegion', deviceIds)
          .then((data) => {
            this.$message.success({
              showClose: true,
              message: '保存成功'
            })
            this.$emit('onChannelChange', node.data.deviceId)
            node.loaded = false
            node.expand()
          }).catch(function(error) {
            console.log(error)
          }).finally(() => {
            this.loading = false
          })
      })
    },
    addChannel: function(id, node) {
      this.$refs.gbChannelSelect.openDialog((data) => {
        console.log('选择的数据')
        console.log(data)
        this.addChannelToCivilCode(node.data.deviceId, data)
      })
    },
    refreshNode: function(node) {
      node.loaded = false
      node.expand()
    },
    refresh: function(deviceId) {
      const tree = this.$refs.veTree
      if (!tree || !tree.store) {
        return
      }
      const refreshNode = (node) => {
        if (!node || !node.data) {
          return false
        }
        if (node.data.deviceId === deviceId) {
          node.loaded = false
          node.expand()
          return true
        }
        if (node.childNodes) {
          for (let i = 0; i < node.childNodes.length; i++) {
            if (refreshNode(node.childNodes[i])) {
              return true
            }
          }
        }
        return false
      }
      const rootNodes = tree.store.root.childNodes || []
      for (let i = 0; i < rootNodes.length; i++) {
        if (refreshNode(rootNodes[i])) {
          break
        }
      }
    },
    addRegion: function(id, node) {
      console.log(node)

      this.$refs.regionEdit.openDialog(form => {
        node.loaded = false
        node.expand()
      }, {
        deviceId: '',
        name: '',
        parentId: node.data.id,
        parentDeviceId: node.data.deviceId
      })
    },
    editCatalog: function(data, node) {
      // 打开添加弹窗
      this.$refs.regionEdit.openDialog(form => {
        node.loaded = false
        node.expand()
      }, node.data)
    },
    nodeClickHandler: function(data, node, tree) {
      if (data && data.nextData && data.nextData.length > 0) {
        const parentNode = node.parent
        let nextData = data.nextData
        if (nextData.length > this.treeLimit) {
          let subData = nextData.splice(0, this.treeLimit)
          subData.push({
            treeId: '---',
            deviceId: '---',
            name: '加载更多...',
            isLeaf: true,
            leaf: true,
            type: 100,
            children: [],
            nextData: nextData
          })
          this.$refs.veTree.remove(data, parentNode)
          for (let item of subData) {
            this.$refs.veTree.append(item, parentNode)
          }
        } else {
          this.$refs.veTree.remove(data, parentNode)
          for (let item of nextData) {
            this.$refs.veTree.append(item, parentNode)
          }
        }
      } else {
        this.chooseId = data.deviceId
        this.activeNode = (data && data.type === 0) ? node : null
        this.$emit('clickEvent', data)
      }
    },
    toolbarAddNode() {
      if (!this.activeCatalogNode) return
      this.addRegion(this.activeCatalogNode.data.id, this.activeCatalogNode)
    },
    toolbarAddChannel() {
      if (!this.activeCatalogNode || !this.canMountChannel) {
        this.$message.warning({
          showClose: true,
          message: '请先选中行政区划节点再添加通道'
        })
        return
      }
      this.addChannel(this.activeCatalogNode.data.id, this.activeCatalogNode)
    },
    toolbarMoreCommand(cmd) {
      const node = this.activeCatalogNode
      if (!node) return
      if (cmd === 'refresh') {
        this.refreshNode(node)
      } else if (cmd === 'edit') {
        if (!this.canEditNode) return
        this.editCatalog(node.data, node)
      } else if (cmd === 'delete') {
        if (!this.canEditNode) return
        this.$confirm(`确定删除目录「${this.currentNodeShort}」？其下子节点也会一并删除。`, '删除目录', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }).then(() => {
          this.removeRegion(node.data.id, node)
        }).catch(() => {})
      }
    },
    listClickHandler: function(data) {
      this.chooseId = data.deviceId
      this.$emit('clickEvent', data)
    },
    channelLstClickHandler: function(data) {
      this.$emit('clickEvent', {
        leaf: true,
        id: data.gbId
      })
    },
    contextmenuEventHandlerForLi(event, data) {
      console.log(data)
      const allMenuItem = []
      if (this.contextmenu) {
        for (let i = 0; i < this.contextmenu.length; i++) {
          let item = this.contextmenu[i]
          allMenuItem.push({
            label: item.label,
            icon: item.icon,
            onClick: () => {
              item.onClick(event, {
                id: data.gbId
              })
            }
          })
        }
      }
      if (allMenuItem.length === 0) {
        return
      }

      this.$contextmenu({
        items: allMenuItem,
        event, // 鼠标事件信息
        customClass: 'custom-class', // 自定义菜单 class
        zIndex: 3000 // 菜单样式 z-index
      })
    }
  }
}
</script>

<style scoped>
.org-tree-panel {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: #fff;
  border: 1px solid #e3ebf5;
  border-radius: 12px;
  box-shadow: 0 2px 10px rgba(21, 101, 192, 0.06);
  overflow: hidden;
  box-sizing: border-box;
}

.org-tree-search {
  flex-shrink: 0;
  padding: 14px 14px 10px;
}

.org-tree-search ::v-deep .el-input__inner {
  border-radius: 8px;
  border-color: #d7e3f2;
  background: #f7faff;
}

.org-tree-search ::v-deep .el-input__inner:focus {
  border-color: #1565c0;
  background: #fff;
}

.org-tree-body {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 0 8px 12px;
  color: #606266;
  overflow: hidden;
}

.org-tree-alert {
  margin: 0 6px 10px;
  border-radius: 8px;
}

.org-tree-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin: 0 6px 8px;
  padding: 6px 8px;
  border-radius: 8px;
  background: #f5f8fc;
  font-size: 13px;
  color: #5a6a7a;
  flex-shrink: 0;
}

.org-tree-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.org-tree-toolbar-label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  line-height: 1;
  white-space: nowrap;
  flex-shrink: 0;
  cursor: pointer;
  margin: 0;
}

.org-tree-hint {
  margin: 0 6px 8px;
  padding: 0 2px;
  font-size: 12px;
  color: #909399;
  flex-shrink: 0;
}

.org-tree-menu-target {
  color: #606266 !important;
  font-size: 12px;
  cursor: default !important;
}

.custom-tree-node .el-radio__label {
  padding-left: 4px !important;
}

.flow-tree {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 4px 6px 8px;
}

.flow-tree ::v-deep .vue-recycle-scroller__item-wrapper {
  height: 100%;
  overflow-x: auto;
}

.org-result-list {
  list-style: none;
  margin: 0;
  padding: 4px 6px;
}

.channel-list-li {
  min-height: 40px;
  align-items: center;
  cursor: pointer;
  display: grid;
  grid-template-columns: 26px 1fr;
  margin-bottom: 8px;
  padding: 6px 8px;
  border-radius: 8px;
}

.channel-list-li:hover {
  background: #f0f6ff;
}
</style>
