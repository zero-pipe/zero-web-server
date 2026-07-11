<template>
  <div class="login-page">
    <div class="login-stage">
      <header class="login-brand">
        <div class="brand-mark" aria-hidden="true">
          <span class="brand-dot" />
          <span class="brand-ring" />
        </div>
        <div class="brand-text">
          <h1>Zero Web Kit</h1>
          <p>智能物联 · 视频接入平台</p>
        </div>
      </header>

      <div class="aiot-hero" aria-hidden="true">
        <svg class="aiot-svg" viewBox="0 0 560 420" role="img">
          <defs>
            <linearGradient id="aiotStroke" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stop-color="#90caf9" />
              <stop offset="100%" stop-color="#1565c0" />
            </linearGradient>
            <radialGradient id="aiotGlow" cx="50%" cy="50%" r="50%">
              <stop offset="0%" stop-color="rgba(21,101,192,0.28)" />
              <stop offset="100%" stop-color="rgba(21,101,192,0)" />
            </radialGradient>
          </defs>

          <circle class="aiot-glow" cx="280" cy="210" r="150" fill="url(#aiotGlow)" />

          <!-- soft orbit rings -->
          <circle class="orbit orbit-a" cx="280" cy="210" r="118" fill="none" stroke="url(#aiotStroke)" stroke-width="1.2" />
          <circle class="orbit orbit-b" cx="280" cy="210" r="168" fill="none" stroke="rgba(21,101,192,0.22)" stroke-width="1" stroke-dasharray="6 10" />

          <!-- connection paths -->
          <path class="link" d="M280 210 L148 118" fill="none" stroke="url(#aiotStroke)" stroke-width="1.5" />
          <path class="link" d="M280 210 L412 118" fill="none" stroke="url(#aiotStroke)" stroke-width="1.5" />
          <path class="link" d="M280 210 L132 268" fill="none" stroke="url(#aiotStroke)" stroke-width="1.5" />
          <path class="link" d="M280 210 L428 268" fill="none" stroke="url(#aiotStroke)" stroke-width="1.5" />
          <path class="link" d="M280 210 L280 338" fill="none" stroke="url(#aiotStroke)" stroke-width="1.5" />

          <!-- satellite nodes -->
          <g class="node node-cam">
            <circle cx="148" cy="118" r="18" fill="#fff" stroke="#1565c0" stroke-width="2" />
            <rect x="140" y="111" width="16" height="12" rx="2" fill="none" stroke="#1565c0" stroke-width="1.6" />
            <circle cx="148" cy="117" r="3.2" fill="#1565c0" />
          </g>
          <g class="node node-cloud">
            <circle cx="412" cy="118" r="18" fill="#fff" stroke="#1565c0" stroke-width="2" />
            <path d="M402 121c0-5 4-9 9-9 1.2-4 5-7 9.5-7 5.5 0 10 4.2 10 9.5v1.2c3 .4 5.5 3 5.5 6.2 0 3.5-2.8 6.3-6.3 6.3h-22c-3.2 0-5.7-2.5-5.7-5.7 0-.5.1-1 .2-1.5z" fill="none" stroke="#1565c0" stroke-width="1.5" />
          </g>
          <g class="node node-sensor">
            <circle cx="132" cy="268" r="18" fill="#fff" stroke="#1565c0" stroke-width="2" />
            <circle cx="132" cy="268" r="6" fill="none" stroke="#1565c0" stroke-width="1.6" />
            <circle cx="132" cy="268" r="2.2" fill="#1565c0" />
          </g>
          <g class="node node-edge">
            <circle cx="428" cy="268" r="18" fill="#fff" stroke="#1565c0" stroke-width="2" />
            <rect x="420" y="260" width="16" height="16" rx="3" fill="none" stroke="#1565c0" stroke-width="1.6" />
            <path d="M424 268h8M428 264v8" stroke="#1565c0" stroke-width="1.5" />
          </g>
          <g class="node node-gateway">
            <circle cx="280" cy="338" r="18" fill="#fff" stroke="#1565c0" stroke-width="2" />
            <path d="M272 342v-8h16v8M276 334v-4h8v4" fill="none" stroke="#1565c0" stroke-width="1.5" />
          </g>

          <!-- center hub -->
          <g class="hub">
            <circle cx="280" cy="210" r="46" fill="#1565c0" />
            <circle class="hub-pulse" cx="280" cy="210" r="46" fill="none" stroke="#90caf9" stroke-width="2" />
            <text x="280" y="206" text-anchor="middle" fill="#fff" font-size="18" font-weight="700" font-family="Segoe UI, PingFang SC, Microsoft YaHei, sans-serif">AIoT</text>
            <text x="280" y="226" text-anchor="middle" fill="rgba(255,255,255,0.86)" font-size="11" font-family="Segoe UI, PingFang SC, Microsoft YaHei, sans-serif">智能物联</text>
          </g>
        </svg>

        <div class="aiot-caption">
          <h2>设备接入 · 实时预览 · 统一运维</h2>
          <p>GB28181 / ONVIF / 媒体转发，一站式智能物联接入</p>
        </div>
      </div>
    </div>

    <aside class="login-panel">
      <el-form
        ref="loginForm"
        :model="loginForm"
        :rules="loginRules"
        class="login-form"
        auto-complete="on"
        label-position="left"
        @submit.native.prevent
      >
        <div class="panel-head">
          <h3>欢迎登录</h3>
          <p>Zero Web Kit 管理平台</p>
        </div>

        <el-form-item prop="username">
          <span class="svg-container">
            <svg-icon icon-class="user" />
          </span>
          <el-input
            ref="username"
            v-model="loginForm.username"
            placeholder="用户名"
            name="username"
            type="text"
            tabindex="1"
            auto-complete="on"
          />
        </el-form-item>

        <el-form-item prop="password">
          <span class="svg-container">
            <svg-icon icon-class="password" />
          </span>
          <el-input
            :key="passwordType"
            ref="password"
            v-model="loginForm.password"
            :type="passwordType"
            placeholder="密码"
            name="password"
            tabindex="2"
            auto-complete="on"
            @keyup.enter.native="handleLogin"
          />
          <span class="show-pwd" @click="showPwd">
            <svg-icon :icon-class="passwordType === 'password' ? 'eye' : 'eye-open'" />
          </span>
        </el-form-item>

        <el-button
          :loading="loading"
          type="primary"
          class="login-btn"
          @click.native.prevent="handleLogin"
        >
          登录
        </el-button>
      </el-form>
    </aside>
  </div>
</template>

<script>
import { validUsername } from '@/utils/validate'

export default {
  name: 'Login',
  data() {
    const validateUsername = (rule, value, callback) => {
      if (!validUsername(value)) {
        callback(new Error('请输入用户名'))
      } else {
        callback()
      }
    }
    const validatePassword = (rule, value, callback) => {
      callback()
    }
    return {
      loginForm: {
        username: '',
        password: ''
      },
      loginRules: {
        username: [{ required: true, trigger: 'blur', validator: validateUsername }],
        password: [{ required: true, trigger: 'blur', validator: validatePassword }]
      },
      loading: false,
      passwordType: 'password',
      redirect: undefined
    }
  },
  watch: {
    $route: {
      handler(route) {
        this.redirect = route.query && route.query.redirect
      },
      immediate: true
    }
  },
  methods: {
    showPwd() {
      this.passwordType = this.passwordType === 'password' ? '' : 'password'
      this.$nextTick(() => {
        this.$refs.password.focus()
      })
    },
    handleLogin() {
      this.$refs.loginForm.validate(valid => {
        if (!valid) return false
        this.loading = true
        this.$store.dispatch('user/login', this.loginForm).then(() => {
          this.$router.push({ path: this.redirect || '/' })
        }).catch((error) => {
          this.$message({
            showClose: true,
            message: error,
            type: 'error'
          })
        }).finally(() => {
          this.loading = false
        })
      })
    }
  }
}
</script>

<style lang="scss">
/* Element 覆盖：仅登录页 */
.login-page {
  .el-input {
    display: inline-block;
    height: 46px;
    width: calc(100% - 42px);

    input {
      background: transparent;
      border: 0;
      border-radius: 0;
      padding: 12px 8px 12px 10px;
      color: #1e293b;
      height: 46px;
      caret-color: #1565c0;

      &:-webkit-autofill {
        box-shadow: 0 0 0 1000px #f5f8fc inset !important;
        -webkit-text-fill-color: #1e293b !important;
      }
    }
  }

  .el-form-item {
    margin-bottom: 18px;
    border: 1px solid #e3ebf5;
    background: #f5f8fc;
    border-radius: 10px;
    color: #64748b;
  }

  .el-form-item__error {
    padding-top: 4px;
  }
}
</style>

<style lang="scss" scoped>
$accent: #1565c0;
$accent-2: #1976d2;
$bg: #eef4fb;
$panel: #ffffff;
$text: #1e293b;
$muted: #64748b;

.login-page {
  min-height: 100vh;
  width: 100%;
  display: grid;
  grid-template-columns: minmax(0, 1fr) min(420px, 100%);
  background:
    radial-gradient(1200px 600px at 18% 20%, rgba(25, 118, 210, 0.16), transparent 60%),
    radial-gradient(900px 500px at 55% 75%, rgba(21, 101, 192, 0.12), transparent 55%),
    linear-gradient(160deg, #f7fbff 0%, $bg 45%, #e3eef9 100%);
  overflow: hidden;
  user-select: none;
}

.login-stage {
  position: relative;
  min-width: 0;
  display: flex;
  flex-direction: column;
  padding: 36px 48px 28px;
}

.login-brand {
  display: flex;
  align-items: center;
  gap: 14px;
  z-index: 1;
}

.brand-mark {
  position: relative;
  width: 42px;
  height: 42px;
}

.brand-dot {
  position: absolute;
  inset: 10px;
  border-radius: 50%;
  background: $accent;
}

.brand-ring {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  border: 2px solid rgba(21, 101, 192, 0.35);
  animation: brand-spin 10s linear infinite;
}

.brand-text h1 {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  color: $text;
  letter-spacing: 0.02em;
}

.brand-text p {
  margin: 4px 0 0;
  font-size: 13px;
  color: $muted;
}

.aiot-hero {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 0;
  padding: 12px 0 24px;
}

.aiot-svg {
  width: min(560px, 86%);
  height: auto;
  max-height: 52vh;
  overflow: visible;
  will-change: transform;
}

.aiot-caption {
  margin-top: 8px;
  text-align: center;
}

.aiot-caption h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 650;
  color: $text;
}

.aiot-caption p {
  margin: 8px 0 0;
  font-size: 13px;
  color: $muted;
}

.orbit-a {
  animation: orbit-rotate 28s linear infinite;
  transform-origin: 280px 210px;
}

.orbit-b {
  animation: orbit-rotate 40s linear infinite reverse;
  transform-origin: 280px 210px;
}

.link {
  stroke-dasharray: 8 12;
  animation: link-flow 2.8s linear infinite;
}

.hub-pulse {
  transform-origin: 280px 210px;
  animation: hub-pulse 2.4s ease-out infinite;
}

.node {
  transform-origin: center;
  animation: node-float 4.5s ease-in-out infinite;
}

.node-cam { animation-delay: 0s; }
.node-cloud { animation-delay: 0.4s; }
.node-sensor { animation-delay: 0.8s; }
.node-edge { animation-delay: 1.2s; }
.node-gateway { animation-delay: 1.6s; }

.login-panel {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px 28px;
  background: $panel;
  border-left: 1px solid #e3ebf5;
  box-shadow: -12px 0 40px rgba(21, 101, 192, 0.08);
}

.login-form {
  width: 100%;
  max-width: 340px;
}

.panel-head {
  margin-bottom: 28px;
}

.panel-head h3 {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  color: $text;
}

.panel-head p {
  margin: 8px 0 0;
  font-size: 13px;
  color: $muted;
}

.svg-container {
  padding: 6px 5px 6px 14px;
  color: $accent;
  vertical-align: middle;
  width: 30px;
  display: inline-block;
}

.show-pwd {
  position: absolute;
  right: 12px;
  top: 8px;
  font-size: 16px;
  color: $muted;
  cursor: pointer;
  user-select: none;

  &:hover {
    color: $accent;
  }
}

.login-btn {
  width: 100%;
  height: 44px;
  margin-top: 8px;
  border: none;
  border-radius: 10px;
  background: linear-gradient(135deg, $accent 0%, $accent-2 100%);
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 0.08em;
  box-shadow: 0 8px 20px rgba(21, 101, 192, 0.28);

  &:hover,
  &:focus {
    background: linear-gradient(135deg, $accent-2 0%, #1e88e5 100%);
  }
}

@keyframes brand-spin {
  to { transform: rotate(360deg); }
}

@keyframes orbit-rotate {
  to { transform: rotate(360deg); }
}

@keyframes link-flow {
  to { stroke-dashoffset: -40; }
}

@keyframes hub-pulse {
  0% { opacity: 0.85; transform: scale(1); }
  70% { opacity: 0; transform: scale(1.35); }
  100% { opacity: 0; transform: scale(1.35); }
}

@keyframes node-float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-4px); }
}

@media (prefers-reduced-motion: reduce) {
  .brand-ring,
  .orbit-a,
  .orbit-b,
  .link,
  .hub-pulse,
  .node {
    animation: none !important;
  }
}

@media (max-width: 960px) {
  .login-page {
    grid-template-columns: 1fr;
    grid-template-rows: minmax(240px, 42vh) auto;
  }

  .login-stage {
    padding: 24px 20px 8px;
  }

  .aiot-svg {
    max-height: 28vh;
  }

  .aiot-caption h2 {
    font-size: 16px;
  }

  .login-panel {
    border-left: none;
    border-top: 1px solid #e3ebf5;
    box-shadow: 0 -8px 28px rgba(21, 101, 192, 0.06);
    padding: 28px 20px 36px;
    align-items: flex-start;
  }
}
</style>
