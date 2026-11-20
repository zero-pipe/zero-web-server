import Vue from 'vue'

import 'normalize.css/normalize.css' // A modern alternative to CSS resets

import ElementUI from 'element-ui'
import 'element-ui/lib/theme-chalk/index.css'
import locale from 'element-ui/lib/locale/lang/en' // lang i18n

import '@/styles/index.scss' // global css

import App from './App'
import store from './store'
import router from './router'

import '@/icons' // icon
import '@/permission' // permission control

import VueClipboards from 'vue-clipboards'
import Contextmenu from 'vue-contextmenujs'
import VueClipboard from 'vue-clipboard2'

/**
 * If you don't want to use mock-server
 * you want to use MockJs for mock api
 * you can execute: mockXHR()
 *
 * Currently MockJs will be used in the production environment,
 * please remove it before going online ! ! !
 */
if (process.env.NODE_ENV === 'production') {
  const { mockXHR } = require('../mock')
  mockXHR()
}

Vue.use(ElementUI)
Vue.use(VueClipboards)
Vue.use(Contextmenu)
Vue.use(VueClipboard)

Vue.config.productionTip = false

Vue.prototype.$channelTypeList = {
  1: { id: 1, name: '国标设备', style: { color: '#1A73E8', borderColor: '#AECBFA' } },
  2: { id: 2, name: '推流设备', style: { color: '#34A853', borderColor: '#CEEAD6' } },
  3: { id: 3, name: '拉流代理', style: { color: '#FBBC05', borderColor: '#FEEFC3' } },
  4: { id: 4, name: 'ONVIF设备', style: { color: '#7c3aed', borderColor: '#ddd6fe' } },
  200: { id: 200, name: '部标设备', style: { color: '#EA4335', borderColor: '#F6AEA9' } }
}

new Vue({
  el: '#app',
  router,
  store,
  render: h => h(App)
})
