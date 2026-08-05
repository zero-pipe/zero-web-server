const getters = {
  sidebar: state => state.app.sidebar,
  device: state => state.app.device,
  token: state => state.user.token,
  showConfirmBoxForLoginLose: state => state.user.showConfirmBoxForLoginLose,
  serverId: state => state.user.serverId,
  name: state => state.user.name,
  menus: state => state.user.menus,
  roleId: state => state.user.roleId,
  visitedViews: state => state.tagsView.visitedViews,
  cachedViews: state => state.tagsView.cachedViews
}
export default getters
