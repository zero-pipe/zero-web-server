import router from './router'
import store from './store'
import NProgress from 'nprogress' // progress bar
import 'nprogress/nprogress.css' // progress bar style
import { getToken, getName, getServerId, getMenus } from '@/utils/auth'
import getPageTitle from '@/utils/get-page-title'
import { canAccessPath, firstAllowedPath } from '@/utils/permission'

NProgress.configure({ showSpinner: false })

const whiteList = ['/login', '/play/share']

router.beforeEach(async(to, from, next) => {
  NProgress.start()
  document.title = getPageTitle(to.meta.title)

  const hasToken = getToken()

  if (hasToken) {
    if (to.path === '/login') {
      next({ path: firstAllowedPath(store.getters.menus || getMenus()) })
      NProgress.done()
      return
    }

    if (!store.getters.name) {
      store.commit('user/SET_NAME', getName())
      store.commit('user/SET_SERVER_ID', getServerId())
    }

    // 刷新后 menus 可能为空，拉一次用户信息
    let menus = store.getters.menus
    if (!menus || !menus.length) {
      try {
        const data = await store.dispatch('user/getUserInfo')
        menus = (data && data.menus) || []
      } catch (e) {
        await store.dispatch('user/resetToken')
        next(`/login?redirect=${to.path}`)
        NProgress.done()
        return
      }
    }

    if (to.path === '/' || to.path === '') {
      next({ path: firstAllowedPath(menus) })
      NProgress.done()
      return
    }

    if (!canAccessPath(menus, to.path)) {
      next({ path: firstAllowedPath(menus), replace: true })
      NProgress.done()
      return
    }

    next()
  } else {
    if (whiteList.indexOf(to.path) !== -1) {
      next()
    } else {
      next(`/login?redirect=${to.path}`)
      NProgress.done()
    }
  }
})

router.afterEach(() => {
  NProgress.done()
})
