/**
 * 双列侧栏菜单：一级业务域 + 二级功能页
 * path 与 vue-router 保持一致；hidden 路由不在此列出
 */
export const primaryMenus = [
  {
    id: 'map',
    title: '电子地图',
    icon: 'menu-map',
    path: '/map'
  },
  {
    id: 'device',
    title: '设备管理',
    icon: 'menu-device',
    children: [
      { title: '分屏监控', path: '/live', icon: 'live' },
      { title: '通道列表', path: '/channel', icon: 'channelManger' },
      { title: '国标设备', path: '/device', icon: 'device' },
      { title: '部标设备', path: '/jtDevice', icon: 'jtDevice' },
      { title: 'ONVIF设备', path: '/onvifDevice', icon: 'device' },
      { title: '推流列表', path: '/push', icon: 'streamPush' },
      { title: '拉流代理', path: '/proxy', icon: 'streamProxy' }
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
    id: 'record',
    title: '录像管理',
    icon: 'menu-record',
    children: [
      { title: '录制计划', path: '/recordPlan', icon: 'recordPlan' },
      { title: '云端录像', path: '/cloudRecord', icon: 'cloudRecord' }
    ]
  },
  {
    id: 'ops',
    title: '运维管理',
    icon: 'menu-ops',
    children: [
      { title: '报警管理', path: '/alarm', icon: 'el-icon-bell' },
      { title: '国标级联', path: '/platform', icon: 'platform' },
      { title: '平台信息', path: '/operations/systemInfo', icon: 'systemInfo' },
      { title: '历史日志', path: '/operations/historyLog', icon: 'historyLog' },
      { title: '实时日志', path: '/operations/realLog', icon: 'realLog' }
    ]
  },
  {
    id: 'system',
    title: '系统管理',
    icon: 'menu-system',
    children: [
      { title: '控制台', path: '/dashboard', icon: 'dashboard' },
      { title: '媒体节点', path: '/mediaServer', icon: 'mediaServerList' }
    ]
  },
  {
    id: 'user',
    title: '用户管理',
    icon: 'menu-user',
    path: '/user'
  }
]

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
