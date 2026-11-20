<template>
  <div class="login-page">
    <section class="login-stage">
      <div class="stage-brand">
        <img :src="brandLogo" alt="Zeroo" class="stage-logo">
        <div class="stage-wordmark">Zeroo <span>Web</span></div>
      </div>
    </section>

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
      brandLogo: require('@/assets/brand/zero-logo.svg'),
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
      font-size: 13px;
      color: #202124;
      height: 46px;
      caret-color: #1A73E8;

      &:-webkit-autofill {
        box-shadow: 0 0 0 1000px #FFFFFF inset !important;
        -webkit-text-fill-color: #202124 !important;
      }
    }
  }

  .el-form-item {
    margin-bottom: 16px;
    border: 1px solid #DADCE0;
    background: #FFFFFF;
    border-radius: 8px;
    color: #5F6368;
    transition: border-color 0.15s ease, box-shadow 0.15s ease;

    &:focus-within {
      border-color: #1A73E8;
      box-shadow: 0 0 0 3px rgba(26, 115, 232, 0.12);
    }
  }

  .el-form-item__error {
    padding-top: 4px;
  }
}
</style>

<style lang="scss" scoped>
$accent: #1A73E8;
$accent-dark: #1765CC;
$text: #202124;
$muted: #5F6368;
$border: #E8EAED;
$bg: #F8F9FA;

.login-page {
  min-height: 100vh;
  width: 100%;
  display: grid;
  grid-template-columns: minmax(0, 1fr) min(380px, 100%);
  background: $bg;
  overflow: hidden;
  user-select: none;
}

.login-stage {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
}

.stage-brand {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 24px;
}

.stage-logo {
  width: 80px;
  height: 80px;
  display: block;
  object-fit: contain;
}

.stage-wordmark {
  font-size: 32px;
  font-weight: 600;
  letter-spacing: -0.4px;
  color: $text;

  span {
    font-weight: 400;
    color: $muted;
  }
}

.login-panel {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 36px;
  background: #FFFFFF;
  border-left: 1px solid $border;
}

.login-form {
  width: 100%;
  max-width: 320px;
}

.svg-container {
  padding: 6px 5px 6px 14px;
  color: $muted;
  vertical-align: middle;
  width: 30px;
  display: inline-block;
}

.show-pwd {
  position: absolute;
  right: 12px;
  top: 10px;
  font-size: 16px;
  color: $muted;
  cursor: pointer;

  &:hover {
    color: $accent;
  }
}

.login-btn {
  width: 100%;
  height: 44px;
  margin-top: 8px;
  border: none;
  border-radius: 8px;
  background: $accent;
  font-size: 13px;
  font-weight: 500;
  letter-spacing: 0.05em;
  box-shadow: none;

  &:hover,
  &:focus {
    background: $accent-dark;
  }

  &:active {
    background: #174EA6;
  }
}

@media (max-width: 900px) {
  .login-page {
    grid-template-columns: 1fr;
    grid-template-rows: auto 1fr;
  }

  .login-stage {
    padding: 48px 24px 24px;
  }

  .stage-logo {
    width: 64px;
    height: 64px;
  }

  .stage-wordmark {
    font-size: 24px;
  }

  .login-panel {
    border-left: none;
    padding: 32px 24px 48px;
  }
}
</style>
