<template>
  <div
    class="dual-sidebar"
    :class="{ 'is-collapse': isCollapse, 'has-secondary': showSecondary }"
    :style="sidebarStyle"
  >
    <div class="primary-rail">
      <div class="brand" @click="goHome">
        <img v-if="logo" :src="logo" class="brand-logo" alt="logo">
        <span v-if="!isCollapse" class="brand-text">Zeroo</span>
      </div>
      <el-scrollbar class="primary-scroll">
        <div
          v-for="item in menus"
          :key="item.id"
          class="primary-item"
          :class="{ active: activePrimary && activePrimary.id === item.id }"
          :title="item.title"
          @click="onPrimaryClick(item)"
        >
          <svg-icon v-if="!isElIcon(item.icon)" :icon-class="item.icon" class="primary-icon" />
          <i v-else :class="[item.icon, 'primary-icon el-icon']" />
          <span v-if="!isCollapse" class="primary-label">{{ item.title }}</span>
        </div>
      </el-scrollbar>
    </div>

    <div v-show="showSecondary" class="secondary-panel">
      <div class="secondary-header">
        <span class="secondary-title">{{ activePrimary ? activePrimary.title : '' }}</span>
      </div>
      <el-scrollbar class="secondary-scroll">
        <router-link
          v-for="child in secondaryItems"
          :key="child.path"
          :to="child.path"
          class="secondary-item"
          :class="{ active: isSecondaryActive(child.path) }"
        >
          <span class="icon-wrap">
            <svg-icon v-if="!isElIcon(child.icon)" :icon-class="child.icon" class="secondary-icon" />
            <i v-else :class="[child.icon, 'secondary-icon']" />
          </span>
          <span class="secondary-label">{{ child.title }}</span>
        </router-link>
      </el-scrollbar>
    </div>

    <div
      v-if="showSecondary && !isCollapse"
      class="resize-handle"
      title="拖动调整宽度"
      @mousedown="startResize"
    />
  </div>
</template>

<script>
import { mapGetters } from 'vuex'
import { primaryMenus, findPrimaryByPath } from '@/layout/menu'
import { filterMenusByPerm } from '@/utils/permission'

export default {
  name: 'Sidebar',
  data() {
    return {
      activePrimary: null,
      logo: require('@/assets/brand/zero-logo.svg'),
      resizing: false
    }
  },
  computed: {
    ...mapGetters(['sidebar']),
    menus() {
      return filterMenusByPerm(primaryMenus, this.$store.getters.menus)
    },
    isCollapse() {
      return !this.sidebar.opened
    },
    showSecondary() {
      return !this.isCollapse && this.activePrimary && Array.isArray(this.activePrimary.children) && this.activePrimary.children.length > 0
    },
    secondaryItems() {
      return (this.activePrimary && this.activePrimary.children) || []
    },
    primaryWidth() {
      return this.isCollapse ? 64 : 84
    },
    secondaryWidth() {
      return this.showSecondary ? (this.sidebar.secondaryWidth || 176) : 0
    },
    totalWidth() {
      return this.primaryWidth + this.secondaryWidth
    },
    sidebarStyle() {
      return {
        width: this.totalWidth + 'px',
        '--primary-width': this.primaryWidth + 'px',
        '--secondary-width': this.secondaryWidth + 'px'
      }
    }
  },
  watch: {
    $route: {
      immediate: true,
      handler(route) {
        const all = this.menus
        const hit = findPrimaryByPath(route.path)
        // findPrimaryByPath 基于全量菜单；再落到当前可见菜单
        this.activePrimary = all.find(m => m.id === (hit && hit.id)) || all[0] || null
        this.$store.dispatch('app/setSidebarWidth', this.totalWidth)
      }
    },
    menus() {
      const all = this.menus
      const hit = findPrimaryByPath(this.$route.path)
      this.activePrimary = all.find(m => m.id === (hit && hit.id)) || all[0] || null
    },
    totalWidth(val) {
      this.$store.dispatch('app/setSidebarWidth', val)
    },
    showSecondary() {
      this.$store.dispatch('app/setSidebarWidth', this.totalWidth)
    }
  },
  beforeDestroy() {
    this.stopResize()
  },
  methods: {
    goHome() {
      const first = this.menus && this.menus[0]
      if (!first) return
      const path = first.path || (first.children && first.children[0] && first.children[0].path)
      if (path) this.$router.push(path)
    },
    isElIcon(icon) {
      return icon && icon.indexOf('el-icon') === 0
    },
    isSecondaryActive(path) {
      const current = this.$route.path
      return current === path || current.startsWith(path + '/')
    },
    onPrimaryClick(item) {
      this.activePrimary = item
      if (item.path) {
        if (this.$route.path !== item.path) {
          this.$router.push(item.path)
        }
        return
      }
      if (item.children && item.children.length) {
        const first = item.children[0]
        const alreadyIn = item.children.some(c => this.$route.path === c.path || this.$route.path.startsWith(c.path + '/'))
        if (!alreadyIn) {
          this.$router.push(first.path)
        }
      }
    },
    startResize(e) {
      this.resizing = true
      const startX = e.clientX
      const startW = this.sidebar.secondaryWidth || 176
      const onMove = (ev) => {
        const next = Math.min(280, Math.max(140, startW + (ev.clientX - startX)))
        this.$store.dispatch('app/setSecondaryWidth', next)
      }
      const onUp = () => {
        this.stopResize()
        document.removeEventListener('mousemove', onMove)
        document.removeEventListener('mouseup', onUp)
      }
      document.addEventListener('mousemove', onMove)
      document.addEventListener('mouseup', onUp)
    },
    stopResize() {
      this.resizing = false
    }
  }
}
</script>

<style lang="scss" scoped>
.dual-sidebar {
  display: flex;
  height: 100%;
  position: relative;
  background: #fff;
  box-shadow: 2px 0 8px rgba(60, 64, 67, 0.06);
  overflow: hidden;
  user-select: none;
}

.primary-rail {
  width: var(--primary-width, 84px);
  flex-shrink: 0;
  background: #fff;
  border-right: 1px solid #E8EAED;
  display: flex;
  flex-direction: column;
  color: #5F6368;
}

.brand {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  cursor: pointer;
  border-bottom: 1px solid #E8EAED;
  flex-shrink: 0;

  .brand-logo {
    width: 26px;
    height: 26px;
    border-radius: 0;
    background: transparent;
    object-fit: contain;
    box-shadow: none;
  }

  .brand-text {
    font-size: 13px;
    font-weight: 600;
    letter-spacing: 0;
    color: #202124;
  }
}

.primary-scroll {
  flex: 1;
  height: 0;

  ::v-deep .el-scrollbar__wrap {
    overflow-x: hidden;
  }
}

.primary-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  min-height: 64px;
  padding: 10px 6px;
  cursor: pointer;
  color: #5F6368;
  transition: background 0.2s, color 0.2s;

  &:hover {
    background: #F1F3F4;
    color: #202124;
  }

  &.active {
    background: #E8F0FE;
    color: #1A73E8;
    box-shadow: inset 3px 0 0 #1A73E8;
  }

  .primary-icon {
    font-size: 22px;
    width: 22px;
    height: 22px;
  }

  .primary-label {
    font-size: 11px;
    line-height: 1.2;
    text-align: center;
    max-width: 72px;
  }
}

.secondary-panel {
  width: var(--secondary-width, 176px);
  flex-shrink: 0;
  background: #F8F9FA;
  border-right: 1px solid #E8EAED;
  display: flex;
  flex-direction: column;
}

.secondary-header {
  height: 56px;
  display: flex;
  align-items: center;
  padding: 0 16px;
  border-bottom: 1px solid #E8EAED;
  flex-shrink: 0;

  .secondary-title {
    font-size: 14px;
    font-weight: 600;
    color: #202124;
  }
}

.secondary-scroll {
  flex: 1;
  height: 0;
  padding: 8px 0;

  ::v-deep .el-scrollbar__wrap {
    overflow-x: hidden;
  }
}

.secondary-item {
  display: flex !important;
  flex-direction: row;
  align-items: center;
  justify-content: flex-start;
  gap: 12px;
  box-sizing: border-box;
  height: 44px;
  min-height: 44px;
  margin: 4px 10px;
  padding: 0 10px 0 8px;
  border-radius: 8px;
  color: #5F6368;
  font-size: 13px;
  font-weight: 400;
  line-height: 44px;
  background: transparent;
  box-shadow: none;
  transition: background 0.15s ease, color 0.15s ease;

  &:hover {
    background: #F1F3F4;
    color: #202124;

    .icon-wrap {
      background: #E8EAED;
      color: #5F6368;
    }
  }

  &.active {
    background: #E8F0FE;
    color: #1A73E8;
    font-weight: 600;
    box-shadow: inset 3px 0 0 #1A73E8;

    .icon-wrap {
      background: #1A73E8;
      color: #fff;
      box-shadow: none;
    }

    .secondary-label {
      color: #1A73E8;
    }

    .secondary-icon {
      color: #fff;
      fill: currentColor;
    }
  }

  .icon-wrap {
    width: 28px;
    height: 28px;
    border-radius: 7px;
    background: #F1F3F4;
    color: #5F6368;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex: 0 0 28px;
    margin: 0;
    padding: 0;
    line-height: 1;
    transition: background 0.15s ease, color 0.15s ease, box-shadow 0.15s ease;
  }

  .secondary-icon {
    font-size: 15px !important;
    width: 15px !important;
    height: 15px !important;
    margin: 0 !important;
    color: inherit;
    fill: currentColor;
    stroke-width: 1.9;
    vertical-align: middle;
  }

  .secondary-label {
    flex: 1;
    min-width: 0;
    height: 44px;
    line-height: 44px;
    margin: 0;
    padding: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    text-align: left;
  }
}

.resize-handle {
  position: absolute;
  top: 0;
  right: 0;
  width: 4px;
  height: 100%;
  cursor: col-resize;
  z-index: 2;

  &:hover,
  &:active {
    background: rgba(26, 115, 232, 0.35);
  }
}

.is-collapse {
  .primary-item {
    min-height: 52px;
    padding: 8px 4px;
  }
}
</style>
