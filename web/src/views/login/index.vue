<template>
  <div class="login-page">
    <div class="login-stage">
      <div class="cv-hero" aria-hidden="true">
        <svg class="cv-svg" viewBox="0 0 900 680" role="img">
          <defs>
            <linearGradient id="gStroke" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stop-color="#81d4fa" />
              <stop offset="100%" stop-color="#1565c0" />
            </linearGradient>
            <linearGradient id="gHud" x1="0%" y1="0%" x2="0%" y2="100%">
              <stop offset="0%" stop-color="#071526" />
              <stop offset="55%" stop-color="#0d2748" />
              <stop offset="100%" stop-color="#12375f" />
            </linearGradient>
            <radialGradient id="gGlow" cx="50%" cy="46%" r="52%">
              <stop offset="0%" stop-color="rgba(66,165,245,0.42)" />
              <stop offset="55%" stop-color="rgba(21,101,192,0.16)" />
              <stop offset="100%" stop-color="rgba(21,101,192,0)" />
            </radialGradient>
            <radialGradient id="gCore" cx="50%" cy="50%" r="50%">
              <stop offset="0%" stop-color="rgba(129,212,250,0.55)" />
              <stop offset="100%" stop-color="rgba(21,101,192,0)" />
            </radialGradient>
            <clipPath id="hudClip">
              <rect x="210" y="120" width="480" height="340" rx="18" />
            </clipPath>
          </defs>

          <circle class="ambient" cx="450" cy="310" r="290" fill="url(#gGlow)" />
          <circle class="core-glow" cx="450" cy="290" r="120" fill="url(#gCore)" />

          <!-- multi-layer orbits -->
          <circle class="orbit o1" cx="450" cy="310" r="220" fill="none" stroke="url(#gStroke)" stroke-width="1.4" opacity="0.55" />
          <circle class="orbit o2" cx="450" cy="310" r="270" fill="none" stroke="#42a5f5" stroke-width="1.1" stroke-dasharray="4 11" opacity="0.38" />
          <circle class="orbit o3" cx="450" cy="310" r="312" fill="none" stroke="#90caf9" stroke-width="0.9" stroke-dasharray="1 16" opacity="0.28" />
          <ellipse class="orbit o4" cx="450" cy="310" rx="340" ry="200" fill="none" stroke="rgba(66,165,245,0.22)" stroke-width="1" stroke-dasharray="6 14" />

          <!-- floating particles -->
          <g class="particles">
            <circle class="p p1" cx="180" cy="160" r="2.2" fill="#81d4fa" />
            <circle class="p p2" cx="720" cy="150" r="2.4" fill="#4fc3f7" />
            <circle class="p p3" cx="140" cy="380" r="1.8" fill="#90caf9" />
            <circle class="p p4" cx="760" cy="390" r="2" fill="#64b5f6" />
            <circle class="p p5" cx="300" cy="70" r="1.6" fill="#81d4fa" />
            <circle class="p p6" cx="600" cy="80" r="2" fill="#42a5f5" />
            <circle class="p p7" cx="450" cy="620" r="2.2" fill="#90caf9" />
            <circle class="p p8" cx="250" cy="560" r="1.7" fill="#64b5f6" />
            <circle class="p p9" cx="650" cy="550" r="1.9" fill="#4fc3f7" />
          </g>

          <!-- outer satellites -->
          <g class="sat s1">
            <circle cx="220" cy="110" r="7" fill="#42a5f5" />
            <circle cx="220" cy="110" r="14" fill="none" stroke="#81d4fa" stroke-width="1.2" opacity="0.7" />
            <circle class="sat-ring" cx="220" cy="110" r="22" fill="none" stroke="rgba(129,212,250,0.35)" stroke-width="1" stroke-dasharray="4 6" />
          </g>
          <g class="sat s2">
            <circle cx="680" cy="110" r="7" fill="#42a5f5" />
            <circle cx="680" cy="110" r="14" fill="none" stroke="#81d4fa" stroke-width="1.2" opacity="0.7" />
            <circle class="sat-ring" cx="680" cy="110" r="22" fill="none" stroke="rgba(129,212,250,0.35)" stroke-width="1" stroke-dasharray="4 6" />
          </g>
          <g class="sat s3">
            <circle cx="120" cy="330" r="7" fill="#29b6f6" />
            <circle cx="120" cy="330" r="14" fill="none" stroke="#81d4fa" stroke-width="1.2" opacity="0.7" />
          </g>
          <g class="sat s4">
            <circle cx="780" cy="330" r="7" fill="#29b6f6" />
            <circle cx="780" cy="330" r="14" fill="none" stroke="#81d4fa" stroke-width="1.2" opacity="0.7" />
          </g>
          <g class="sat s5">
            <circle cx="450" cy="590" r="7" fill="#4fc3f7" />
            <circle cx="450" cy="590" r="14" fill="none" stroke="#81d4fa" stroke-width="1.2" opacity="0.7" />
            <circle class="sat-ring" cx="450" cy="590" r="24" fill="none" stroke="rgba(129,212,250,0.3)" stroke-width="1" stroke-dasharray="5 7" />
          </g>

          <path class="beam" d="M450 290 L220 110" fill="none" stroke="url(#gStroke)" stroke-width="1.5" opacity="0.55" />
          <path class="beam" d="M450 290 L680 110" fill="none" stroke="url(#gStroke)" stroke-width="1.5" opacity="0.55" />
          <path class="beam" d="M240 330 L120 330" fill="none" stroke="url(#gStroke)" stroke-width="1.4" opacity="0.45" />
          <path class="beam" d="M660 330 L780 330" fill="none" stroke="url(#gStroke)" stroke-width="1.4" opacity="0.45" />
          <path class="beam" d="M450 460 L450 590" fill="none" stroke="url(#gStroke)" stroke-width="1.4" opacity="0.45" />

          <!-- main HUD -->
          <g class="hud">
            <rect x="210" y="120" width="480" height="340" rx="18" fill="url(#gHud)" stroke="#4fc3f7" stroke-width="1.8" />
            <rect x="218" y="128" width="464" height="324" rx="14" fill="none" stroke="rgba(129,212,250,0.18)" stroke-width="1" />

            <!-- corner brackets -->
            <path d="M228 140h34M228 140v34" fill="none" stroke="#81d4fa" stroke-width="2.4" />
            <path d="M672 140h-34M672 140v34" fill="none" stroke="#81d4fa" stroke-width="2.4" />
            <path d="M228 440h34M228 440v-34" fill="none" stroke="#81d4fa" stroke-width="2.4" />
            <path d="M672 440h-34M672 440v-34" fill="none" stroke="#81d4fa" stroke-width="2.4" />

            <g clip-path="url(#hudClip)">
              <!-- perspective grid -->
              <g class="grid-layer">
                <path d="M210 180h480M210 230h480M210 280h480M210 330h480M210 380h480M210 430h480" stroke="rgba(129,212,250,0.11)" stroke-width="1" />
                <path d="M270 120v340M330 120v340M390 120v340M450 120v340M510 120v340M570 120v340M630 120v340" stroke="rgba(129,212,250,0.09)" stroke-width="1" />
                <path d="M210 420 L320 300 L450 340 L580 270 L690 380 L690 460 L210 460 Z" fill="rgba(41,182,246,0.08)" />
                <path d="M210 420 L320 300 L450 340 L580 270 L690 380" fill="none" stroke="rgba(79,195,247,0.22)" stroke-width="1.2" />
              </g>

              <!-- dual pose -->
              <g class="pose pose-a">
                <circle cx="320" cy="190" r="10" fill="none" stroke="#81d4fa" stroke-width="1.8" />
                <path d="M320 200v48M320 222l-22 18M320 222l22 18M320 248l-18 40M320 248l18 40" fill="none" stroke="#4fc3f7" stroke-width="2" stroke-linecap="round" />
                <rect class="track t1" x="286" y="176" width="68" height="120" rx="4" fill="none" stroke="#4fc3f7" stroke-width="1.7" />
                <path d="M286 176h12M286 176v12M354 176h-12M354 176v12M286 296h12M286 296v-12M354 296h-12M354 296v-12" fill="none" stroke="#81d4fa" stroke-width="2" />
              </g>

              <g class="pose pose-b">
                <circle cx="410" cy="210" r="8" fill="none" stroke="#64b5f6" stroke-width="1.5" />
                <path d="M410 218v36M410 234l-16 14M410 234l16 14M410 254l-14 30M410 254l14 30" fill="none" stroke="#64b5f6" stroke-width="1.6" stroke-linecap="round" />
                <rect class="track t4" x="384" y="198" width="52" height="92" rx="3" fill="none" stroke="#64b5f6" stroke-width="1.4" opacity="0.85" />
              </g>

              <!-- vehicle + motion trail -->
              <g class="vehicle">
                <rect class="track t2" x="500" y="250" width="120" height="62" rx="4" fill="none" stroke="#90caf9" stroke-width="1.7" />
                <path d="M512 281h96M530 258v46M590 258v46" stroke="rgba(144,202,249,0.4)" stroke-width="1.1" />
                <path class="trail" d="M500 281 C470 275 450 290 430 286" fill="none" stroke="rgba(79,195,247,0.45)" stroke-width="2" stroke-dasharray="4 5" />
                <path d="M500 250h14M500 250v14M620 250h-14M620 250v14M500 312h14M500 312v-14M620 312h-14M620 312v-14" fill="none" stroke="#81d4fa" stroke-width="1.8" />
              </g>

              <!-- face ROI -->
              <g class="face">
                <rect class="track t3" x="545" y="150" width="70" height="84" rx="4" fill="none" stroke="#29b6f6" stroke-width="1.6" />
                <circle cx="566" cy="178" r="3" fill="#81d4fa" />
                <circle cx="594" cy="178" r="3" fill="#81d4fa" />
                <path d="M570 204c8 7 18 7 26 0" fill="none" stroke="#81d4fa" stroke-width="1.4" />
                <path d="M545 150h12M545 150v12M615 150h-12M615 150v12" fill="none" stroke="#4fc3f7" stroke-width="1.8" />
              </g>

              <!-- segmentation cloud -->
              <g class="seg">
                <circle cx="360" cy="360" r="3.2" fill="#42a5f5" opacity="0.75" />
                <circle cx="372" cy="350" r="2.6" fill="#64b5f6" opacity="0.65" />
                <circle cx="348" cy="352" r="2.8" fill="#90caf9" opacity="0.6" />
                <circle cx="384" cy="362" r="2.2" fill="#4fc3f7" opacity="0.7" />
                <circle cx="356" cy="372" r="2.4" fill="#42a5f5" opacity="0.55" />
                <circle cx="370" cy="376" r="1.8" fill="#81d4fa" opacity="0.5" />
                <circle cx="340" cy="364" r="2" fill="#64b5f6" opacity="0.55" />
                <circle cx="392" cy="348" r="1.6" fill="#90caf9" opacity="0.45" />
              </g>

              <!-- hex mesh accents -->
              <g class="hex" opacity="0.35">
                <path d="M250 340l10-6 10 6v12l-10 6-10-6z" fill="none" stroke="#81d4fa" stroke-width="1" />
                <path d="M270 352l10-6 10 6v12l-10 6-10-6z" fill="none" stroke="#64b5f6" stroke-width="1" />
                <path d="M250 364l10-6 10 6v12l-10 6-10-6z" fill="none" stroke="#90caf9" stroke-width="1" />
              </g>

              <!-- lidar / radar -->
              <g class="radar" transform="translate(450 360)">
                <circle r="86" fill="none" stroke="rgba(79,195,247,0.16)" stroke-width="1.2" />
                <circle r="58" fill="none" stroke="rgba(79,195,247,0.14)" stroke-width="1" />
                <circle r="30" fill="none" stroke="rgba(79,195,247,0.12)" stroke-width="1" />
                <path class="radar-sweep" d="M0 0 L0 -86 A86 86 0 0 1 61 -61 Z" fill="rgba(66,165,245,0.18)" />
                <circle class="radar-blip b1" cx="36" cy="-28" r="3" fill="#4fc3f7" />
                <circle class="radar-blip b2" cx="-42" cy="18" r="2.4" fill="#81d4fa" />
                <circle class="radar-blip b3" cx="18" cy="40" r="2.6" fill="#90caf9" />
              </g>

              <!-- dual scan lines -->
              <rect class="scan scan-a" x="210" y="120" width="480" height="3" fill="#4fc3f7" opacity="0.55" />
              <rect class="scan scan-b" x="210" y="120" width="480" height="2" fill="#81d4fa" opacity="0.35" />

              <!-- vertical sweep -->
              <rect class="vscan" x="210" y="120" width="2" height="340" fill="#29b6f6" opacity="0.35" />
            </g>

            <!-- side iris lock -->
            <g class="iris" transform="translate(720 290)">
              <circle r="42" fill="none" stroke="rgba(79,195,247,0.28)" stroke-width="1.2" />
              <circle class="iris-ring" r="30" fill="none" stroke="#4fc3f7" stroke-width="1.6" stroke-dasharray="10 8" />
              <circle class="iris-ring2" r="18" fill="none" stroke="#81d4fa" stroke-width="1.2" stroke-dasharray="5 7" />
              <circle r="7" fill="none" stroke="#90caf9" stroke-width="1.3" />
              <circle r="2.8" fill="#4fc3f7" />
            </g>
          </g>

          <!-- neural net bottom -->
          <g class="neural">
            <circle cx="260" cy="540" r="4.5" fill="#64b5f6" />
            <circle cx="320" cy="520" r="4.5" fill="#42a5f5" />
            <circle cx="380" cy="548" r="4.5" fill="#90caf9" />
            <circle cx="450" cy="518" r="5" fill="#4fc3f7" />
            <circle cx="520" cy="548" r="4.5" fill="#90caf9" />
            <circle cx="580" cy="522" r="4.5" fill="#42a5f5" />
            <circle cx="640" cy="540" r="4.5" fill="#64b5f6" />
            <path class="beam" d="M260 540 L320 520 L380 548 L450 518 L520 548 L580 522 L640 540" fill="none" stroke="rgba(66,165,245,0.5)" stroke-width="1.3" />
            <path class="beam" d="M320 520 L380 548 L520 548 L580 522" fill="none" stroke="rgba(129,212,250,0.28)" stroke-width="1" />
          </g>
        </svg>
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
        <div class="panel-brand">
          <div class="zero-logo" aria-hidden="true">
            <svg viewBox="0 0 48 48" class="zero-logo-svg">
              <circle cx="24" cy="24" r="18" fill="none" stroke="#1565c0" stroke-width="2.4" />
              <circle cx="24" cy="24" r="7" fill="#1565c0" />
              <path d="M24 6v6M24 36v6M6 24h6M36 24h6" stroke="#90caf9" stroke-width="2" stroke-linecap="round" />
              <path d="M14 14l4 4M30 30l4 4M34 14l-4 4M18 30l-4 4" stroke="#1565c0" stroke-width="1.6" stroke-linecap="round" opacity="0.55" />
            </svg>
          </div>
          <div class="panel-brand-text">
            <h3>Zero</h3>
            <p>Zero Media AIoT Platform</p>
          </div>
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
$bg: #e8f1fa;
$panel: #ffffff;
$text: #1e293b;
$muted: #64748b;

.login-page {
  min-height: 100vh;
  width: 100%;
  display: grid;
  grid-template-columns: minmax(0, 1fr) min(380px, 100%);
  background:
    radial-gradient(1000px 520px at 22% 30%, rgba(66, 165, 245, 0.18), transparent 58%),
    radial-gradient(800px 480px at 60% 80%, rgba(21, 101, 192, 0.12), transparent 55%),
    linear-gradient(155deg, #f4f9ff 0%, $bg 48%, #dceaf7 100%);
  overflow: hidden;
  user-select: none;
}

.login-stage {
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 12px 8px 12px 20px;
}

.cv-hero {
  width: min(980px, 98%);
}

.cv-svg {
  width: 100%;
  height: auto;
  max-height: 92vh;
  overflow: visible;
  filter: drop-shadow(0 22px 48px rgba(21, 101, 192, 0.22));
}

.orbit.o1 {
  transform-origin: 450px 310px;
  animation: spin 24s linear infinite;
}

.orbit.o2 {
  transform-origin: 450px 310px;
  animation: spin 36s linear infinite reverse;
}

.orbit.o3 {
  transform-origin: 450px 310px;
  animation: spin 48s linear infinite;
}

.orbit.o4 {
  transform-origin: 450px 310px;
  animation: spin 56s linear infinite reverse;
}

.beam {
  stroke-dasharray: 6 11;
  animation: dash 2.2s linear infinite;
}

.sat {
  animation: pulse 2.6s ease-in-out infinite;
}

.sat-ring {
  transform-box: fill-box;
  transform-origin: center;
  animation: spin 8s linear infinite;
}

.s2 { animation-delay: 0.35s; }
.s3 { animation-delay: 0.7s; }
.s4 { animation-delay: 1.05s; }
.s5 { animation-delay: 1.4s; }

.p {
  animation: particle 3.4s ease-in-out infinite;
}
.p2 { animation-delay: 0.3s; }
.p3 { animation-delay: 0.6s; }
.p4 { animation-delay: 0.9s; }
.p5 { animation-delay: 1.2s; }
.p6 { animation-delay: 1.5s; }
.p7 { animation-delay: 1.8s; }
.p8 { animation-delay: 2.1s; }
.p9 { animation-delay: 2.4s; }

.track {
  animation: blink 2s ease-in-out infinite;
}
.t2 { animation-delay: 0.4s; }
.t3 { animation-delay: 0.8s; }
.t4 { animation-delay: 1.1s; }

.scan-a {
  animation: scan-y 2.6s ease-in-out infinite;
}
.scan-b {
  animation: scan-y 3.4s ease-in-out infinite reverse;
}
.vscan {
  animation: scan-x 4.2s ease-in-out infinite;
}

.radar-sweep {
  transform-origin: 0 0;
  animation: spin 3.2s linear infinite;
}

.radar-blip {
  animation: pulse 1.8s ease-in-out infinite;
}
.b2 { animation-delay: 0.4s; }
.b3 { animation-delay: 0.9s; }

.iris-ring {
  transform-origin: 0 0;
  animation: spin 3.8s linear infinite reverse;
}
.iris-ring2 {
  transform-origin: 0 0;
  animation: spin 5.2s linear infinite;
}

.pose,
.vehicle,
.face,
.seg,
.hex {
  animation: float-y 4.8s ease-in-out infinite;
}
.pose-b { animation-delay: 0.5s; }
.vehicle { animation-delay: 0.8s; }
.face { animation-delay: 1.2s; }
.trail {
  stroke-dasharray: 4 6;
  animation: dash 1.6s linear infinite;
}

.login-panel {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 36px 28px;
  background: $panel;
  border-left: 1px solid #e3ebf5;
  box-shadow: -12px 0 40px rgba(21, 101, 192, 0.08);
}

.login-form {
  width: 100%;
  max-width: 320px;
}

.panel-brand {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 36px;
}

.zero-logo {
  width: 52px;
  height: 52px;
  flex-shrink: 0;
}

.zero-logo-svg {
  width: 100%;
  height: 100%;
  display: block;
}

.panel-brand-text h3 {
  margin: 0;
  font-size: 28px;
  font-weight: 750;
  color: $text;
  letter-spacing: 0.04em;
}

.panel-brand-text p {
  margin: 6px 0 0;
  font-size: 12px;
  color: $muted;
  letter-spacing: 0.06em;
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

  &:hover { color: $accent; }
}

.login-btn {
  width: 100%;
  height: 44px;
  margin-top: 10px;
  border: none;
  border-radius: 10px;
  background: linear-gradient(135deg, $accent 0%, $accent-2 100%);
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 0.1em;
  box-shadow: 0 8px 20px rgba(21, 101, 192, 0.28);

  &:hover,
  &:focus {
    background: linear-gradient(135deg, $accent-2 0%, #1e88e5 100%);
  }
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

@keyframes dash {
  to { stroke-dashoffset: -48; }
}

@keyframes pulse {
  0%, 100% { opacity: 0.4; }
  50% { opacity: 1; }
}

@keyframes blink {
  0%, 100% { opacity: 0.4; }
  50% { opacity: 1; }
}

@keyframes scan-y {
  0% { transform: translateY(0); opacity: 0.12; }
  10% { opacity: 0.7; }
  100% { transform: translateY(334px); opacity: 0.1; }
}

@keyframes scan-x {
  0% { transform: translateX(0); opacity: 0.1; }
  12% { opacity: 0.55; }
  100% { transform: translateX(476px); opacity: 0.08; }
}

@keyframes float-y {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-4px); }
}

@keyframes particle {
  0%, 100% { opacity: 0.25; transform: translateY(0); }
  50% { opacity: 0.95; transform: translateY(-6px); }
}

@media (prefers-reduced-motion: reduce) {
  .orbit,
  .beam,
  .sat,
  .sat-ring,
  .track,
  .scan-a,
  .scan-b,
  .vscan,
  .radar-sweep,
  .radar-blip,
  .iris-ring,
  .iris-ring2,
  .pose,
  .vehicle,
  .face,
  .seg,
  .hex,
  .trail,
  .p {
    animation: none !important;
  }
}

@media (max-width: 960px) {
  .login-page {
    grid-template-columns: 1fr;
    grid-template-rows: minmax(300px, 50vh) auto;
  }

  .cv-svg {
    max-height: 46vh;
  }

  .login-panel {
    border-left: none;
    border-top: 1px solid #e3ebf5;
    box-shadow: 0 -8px 28px rgba(21, 101, 192, 0.06);
    padding: 28px 20px 36px;
  }
}
</style>
