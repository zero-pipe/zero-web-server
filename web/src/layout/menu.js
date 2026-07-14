/**
 * 双列侧栏：按「物联中台底座」能力域划分一级菜单。
 * id = 权限码（与 rbac.AllMenus 对齐）；path 与 vue-router 一致。
 *
 * 能力映射：
 * - access  接入：设备 / 通道 / 国标配置 / 级联
 * - media   媒体：节点集群调度 / 推拉流 / 分屏观察
 * - storage 存储：对象存储对接 / 录像元数据
 * - observe 观察入口：地图 / 报警（非底座核心，可裁剪）
 */
export const primaryMenus = [
  {
    id: 'access',
    title: '接入',
    icon: 'menu-device',
    children: [
      { title: '设备列表', path: '/devices', icon: 'devices' },
      { title: '通道列表', path: '/channel', icon: 'channelManger' },
      { title: '国标配置', path: '/gbConfig', icon: 'gbConfig' },
      { title: '国标级联', path: '/platform', icon: 'gbCascade' }
    ]
  },
  {
    id: 'media',
    title: '媒体',
    icon: 'mediaServerList',
    children: [
      { title: '媒体节点', path: '/mediaServer', icon: 'mediaServerList' },
      { title: '推流列表', path: '/push', icon: 'streamPush' },
      { title: '拉流代理', path: '/proxy', icon: 'streamProxy' },
      { title: '分屏监控', path: '/live', icon: 'live' }
    ]
  },
  {
    id: 'storage',
    title: '存储',
    icon: 'cloudRecord',
    children: [
      { title: '对象存储', path: '/objectStore', icon: 'cloudRecord' },
      { title: '录制计划', path: '/recordPlan', icon: 'recordPlan' },
      { title: '云端录像', path: '/cloudRecord', icon: 'cloudRecord' }
    ]
  },
  {
    id: 'org',
    title: '组织',
    icon: 'menu-org',
    children: [
      { title: '行政区划', path: '/commonChannel/region', icon: 'region' },
      { title: '业务分组', path: '/commonChannel/group', icon: 'tree' }
    ]
  },
  {
    id: 'observe',
    title: '观察',
    icon: 'menu-map',
    children: [
      { title: '电子地图', path: '/map', icon: 'menu-map' },
      { title: '报警管理', path: '/alarm', icon: 'el-icon-bell' }
    ]
  },
  {
    id: 'ops',
    title: '运维',
    icon: 'menu-ops',
    children: [
      { title: '控制台', path: '/dashboard', icon: 'dashboard' },
      { title: '平台信息', path: '/operations/systemInfo', icon: 'systemInfo' },
      { title: '历史日志', path: '/operations/historyLog', icon: 'historyLog' },
      { title: '实时日志', path: '/operations/realLog', icon: 'realLog' }
    ]
  },
  {
    id: 'user',
    title: '用户',
    icon: 'menu-user',
    children: [
      { title: '用户列表', path: '/user', icon: 'user' },
      { title: '角色管理', path: '/role', icon: 'el-icon-s-check' }
    ]
  }
]

/** 旧权限码 → 新能力码（角色库兼容） */
export const legacyMenuAlias = {
  device: 'access',
  system: 'access',
  record: 'storage',
  live: 'media',
  map: 'observe',
  alarm: 'observe'
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
