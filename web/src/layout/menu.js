/**
 * 双列侧栏：按「物联中台底座」能力域划分一级菜单。
 * id = 权限码（与 rbac.AllMenus 对齐）；path 与 vue-router 一致。
 *
 * 能力映射：
 * - access  接入：设备 / 通道 / 国标配置 / 级联
 * - media   媒体：节点集群 / 推拉流（调度配置，不含人看画面）
 * - storage 存储：对象存储对接 / 录像元数据
 * - app     应用：地图 / 分屏 / 报警（上层使用入口）
 */
export const primaryMenus = [
  {
    id: 'app',
    title: '应用管理',
    icon: 'menu-app',
    children: [
      { title: '电子地图', path: '/map', icon: 'menu-map' },
      { title: '分屏监控', path: '/live', icon: 'live' },
      { title: '报警管理', path: '/alarm', icon: 'alarm' },
      { title: '事件中心', path: '/events', icon: 'menu-record' },
      { title: '自定义事件', path: '/customEvent', icon: 'menu-record' }
    ]
  },
  {
    id: 'access',
    title: '接入管理',
    icon: 'menu-access',
    children: [
      { title: '插件管理', path: '/plugins', icon: 'el-icon-box' },
      { title: '设备接入', path: '/devices', icon: 'devices' },
      { title: '设备资产', path: '/assets', icon: 'menu-access' },
      { title: '国标配置', path: '/gbConfig', icon: 'gbConfig' },
      { title: '国标级联', path: '/platform', icon: 'gbCascade' }
    ]
  },
  {
    id: 'media',
    title: '媒体管理',
    icon: 'menu-media',
    children: [
      { title: '媒体节点', path: '/mediaServer', icon: 'media-node' },
      { title: '媒体会话', path: '/mediaSession', icon: 'streamPush' }
    ]
  },
  {
    id: 'storage',
    title: '存储管理',
    icon: 'menu-storage',
    children: [
      { title: '基础设施', path: '/infra', icon: 'menu-storage' },
      { title: '录制计划', path: '/recordPlan', icon: 'recordPlan' },
      { title: '云端录像', path: '/cloudRecord', icon: 'cloudRecord' }
    ]
  },
  {
    id: 'org',
    title: '组织管理',
    icon: 'menu-org',
    children: [
      { title: '行政区划', path: '/commonChannel/region', icon: 'region' },
      { title: '业务分组', path: '/commonChannel/group', icon: 'tree' }
    ]
  },
  {
    id: 'ops',
    title: '运维管理',
    icon: 'menu-ops',
    children: [
      { title: '接入鉴权', path: '/accessToken', icon: 'el-icon-key' },
      { title: '资源监控', path: '/dashboard', icon: 'dashboard' },
      { title: '平台信息', path: '/operations/systemInfo', icon: 'systemInfo' },
      { title: '历史日志', path: '/operations/historyLog', icon: 'historyLog' },
      { title: '实时日志', path: '/operations/realLog', icon: 'realLog' }
    ]
  },
  {
    id: 'user',
    title: '用户管理',
    icon: 'menu-user',
    children: [
      { title: '用户列表', path: '/user', icon: 'user' },
      { title: '角色管理', path: '/role', icon: 'role' }
    ]
  }
]

/** 旧权限码 → 新能力码（角色库兼容） */
export const legacyMenuAlias = {
  device: 'access',
  system: 'access',
  record: 'storage',
  live: 'app',
  map: 'app',
  alarm: 'app',
  observe: 'app'
}

export function findPrimaryByPath(routePath) {
  const path = (routePath || '').split('?')[0]
  for (const item of primaryMenus) {
    if (item.path && (path === item.path || path.startsWith(item.path + '/'))) {
      return item
    }
    if (item.children) {
      const hit = item.children.find(c => path === c.path || path.startsWith(c.path + '/'))
      if (hit) return item
    }
  }
  return primaryMenus[0]
}

export function findSecondaryByPath(primary, routePath) {
  if (!primary || !primary.children) return null
  const path = (routePath || '').split('?')[0]
  return primary.children.find(c => path === c.path || path.startsWith(c.path + '/')) || primary.children[0]
}
