import axios from 'axios'
import { MessageBox, Message } from 'element-ui'
import store from '@/store'
import { getToken } from '@/utils/auth'

let showLoginConfirm = false

// create an axios instance
const service = axios.create({
  baseURL: process.env.VUE_APP_BASE_API, // url = base url + request url
  // withCredentials: true, // send cookies when cross-domain requests
  timeout: 30000 // request timeout
})

// request interceptor
service.interceptors.request.use(
  config => {
    // do something before request is sent
    if (store.getters.token && config.url && config.url.indexOf('api/user/login') < 0) {
      config.headers['access-token'] = getToken()
    }
    return config
  },
  error => {
    console.log(error) // for debug
    return Promise.reject(error)
  }
)

// response interceptor
service.interceptors.response.use(
  response => {
    if (response.config.url && response.config.url.indexOf('/api/user/logout') >= 0) {
      return
    }
    const res = response.data
    // 历史日志文件等接口直接返回纯文本
    if (typeof res === 'string') {
      return res
    }
    if (!res || typeof res !== 'object') {
      const err = new Error('响应无效')
      Message.error({ message: err.message, showClose: true })
      return Promise.reject(err)
    }
    if (res.code && res.code !== 0) {
      const msg = res.msg || '请求失败'
      Message.error({ message: msg, showClose: true })
      return Promise.reject(new Error(msg))
    }
    return res
  },
  error => {
    console.log(error) // for debug
    if (axios.isCancel(error)) {
      return Promise.reject(error)
    }
    // 业务层可能 reject 了普通 Error / 字符串，没有 response
    if (!error || !error.response) {
      let msg = (error && error.message) || (typeof error === 'string' ? error : '网络异常')
      if (/timeout of \d+ms exceeded/i.test(msg)) {
        msg = '请求超时，请检查后端服务或媒体节点是否可达'
      }
      if (msg && store.getters.showConfirmBoxForLoginLose) {
        Message.error({ message: msg, showClose: true })
      }
      return Promise.reject(error || new Error(msg))
    }
    if (error.response.status === 401) {
      if (!showLoginConfirm && store.getters.showConfirmBoxForLoginLose) {
        // to re-login
        showLoginConfirm = true
        MessageBox.confirm('登录已经到期， 是否重新登录', '登录确认', {
          confirmButtonText: '重新登录',
          cancelButtonText: '取消',
          type: 'warning'
        }).then(() => {
          store.dispatch('user/resetToken').then(() => {
            location.reload()
          })
        }).catch(() => {
          store.dispatch('user/closeConfirmBoxForLoginLose')
          Message.warning({
            type: 'warning',
            message: '登录过期提示已经关闭，请注销后重新登录'
          })
        })
      }
    } else if (store.getters.showConfirmBoxForLoginLose) {
      const data = error.response.data
      Message.error({
        message: (data && data.msg) || error.message || '请求失败',
        showClose: true
      })
    }
    return Promise.reject(error)
  }
)

export default service
