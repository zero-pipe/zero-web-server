/**
 * 播放策略：把 mpegts.js（B站开源，HTTP-FLV/MSE）与 webRTC 两个引擎编排成用户无感的播放器。
 * 唯一决策入口 resolvePlayerStrategy，输入视频/音频编码与 ZMS WebRTC 能力，
 * 输出首选播放器、可用播放器列表与拉流 URL 优先级。
 */

/** 用户可见文案（不暴露引擎名称/术语）。 */
export const PLAYER_LABELS = {
  mpegts: '标准',
  webRTC: '超低延迟'
}

/** 全局播放模式：所有页面共用同一个选择，内部再按码流能力换算成具体引擎。 */
export const PLAYER_MODE_OPTIONS = [
  { key: 'auto', label: '自动' },
  { key: 'mpegts', label: '标准' },
  { key: 'webRTC', label: '超低延迟' }
]

/** 全局播放模式持久化键（沿用历史键名，兼容老版本已保存的值）。 */
export const PLAYER_MODE_STORAGE_KEY = 'globalPlayer'

/** 归一化历史/非法值：jessibuca/h265web 等旧值统一归入「标准」(mpegts)。 */
export function normalizePlayerMode(mode) {
  if (mode === 'webRTC') return 'webRTC'
  return 'mpegts'
}

/** 模式 → PlayerTabs 的 preferredPlayer（空串表示交给策略自动决定）。 */
export function modeToPreferredPlayer(mode) {
  const m = normalizePlayerMode(mode)
  return m === 'auto' ? '' : m
}

/** WebRTC 能承载的音频编码；AAC 等需转码，默认不选 WebRTC（会无声）。 */
export const WEBRTC_AUDIO_CODECS = ['OPUS', 'PCMU', 'PCMA', 'G722']

export function normalizeVideoCodec(codec) {
  if (!codec) return ''
  const c = String(codec).toUpperCase().trim()
  if (c.includes('265') || c === 'HEVC') return 'H265'
  if (c.includes('264') || c === 'AVC') return 'H264'
  return c
}

export function normalizeAudioCodec(codec) {
  if (!codec) return ''
  const c = String(codec).toUpperCase().replace(/[\s/_-]/g, '').trim()
  if (c === 'MPEG4GENERIC' || c === 'AAC' || c === 'AACLC') return 'AAC'
  // G.711 A律/U律：WebRTC 原生支持（PCMA/PCMU）
  if (c === 'G711A' || c === 'PCMA' || c === 'ALAW' || c === 'G711ALAW') return 'PCMA'
  if (c === 'G711U' || c === 'PCMU' || c === 'MULAW' || c === 'ULAW' || c === 'G711ULAW') return 'PCMU'
  if (c === 'G722') return 'G722'
  if (c === 'OPUS') return 'OPUS'
  return c
}

export function isH265Codec(codec) {
  return normalizeVideoCodec(codec) === 'H265'
}

export function isWebRTCAudioCodec(audioCodec) {
  return WEBRTC_AUDIO_CODECS.indexOf(normalizeAudioCodec(audioCodec)) >= 0
}

const FLV_URL_PRIORITY = ['flv', 'https_flv', 'ws_flv', 'wss_flv']

/**
 * @param {object} opts
 * @param {string} opts.videoCodec 视频编码（H264/H265/空）
 * @param {string} opts.audioCodec 音频编码
 * @param {boolean} opts.hasAudio 是否有音频（false 明确表示无）
 * @param {boolean} opts.rtcAvailable ZMS 是否具备 WebRTC 能力（rtc 字段存在）
 * @returns {{preferredPlayer, allowedPlayers, urlPriority, videoCodec, audioCodec, hasAudio, webrtcAudioOk}}
 */
export function resolvePlayerStrategy(opts) {
  opts = opts || {}
  const videoCodec = normalizeVideoCodec(opts.videoCodec || opts.configCodec)
  const audioCodec = normalizeAudioCodec(opts.audioCodec)
  const hasAudio = opts.hasAudio === true || (opts.hasAudio !== false && !!audioCodec)
  const rtcAvailable = opts.rtcAvailable === true

  // 1) H265：浏览器 MSE 能否解取决于浏览器(部分 Chromium 支持 H265 MSE)；统一用 mpegts 尝试。
  //    不支持会由引擎报错，交由集成层提示，不再走独立的 h265web 软解。
  if (videoCodec === 'H265') {
    return {
      preferredPlayer: 'mpegts',
      allowedPlayers: ['mpegts'],
      urlPriority: FLV_URL_PRIORITY.slice(),
      videoCodec,
      audioCodec,
      hasAudio,
      webrtcAudioOk: false
    }
  }

  // 2) H264 / 未知：mpegts.js(HTTP-FLV / MSE) 为首选。WebRTC 仅作最后备用项。
  const allowed = ['mpegts']
  const preferred = 'mpegts'
  const webrtcAudioOk = !hasAudio || isWebRTCAudioCodec(audioCodec)

  if (rtcAvailable && videoCodec === 'H264') {
    allowed.push('webRTC')
  }

  return {
    preferredPlayer: preferred,
    allowedPlayers: allowed,
    urlPriority: FLV_URL_PRIORITY.slice(),
    videoCodec,
    audioCodec,
    hasAudio,
    webrtcAudioOk
  }
}

/**
 * 由通道 PTZType 推导播放器云台/镜头能力集（唯一真相）。
 *
 * PTZType 枚举沿用 GB28181 目录定义（2016 A.2.1 g / 2022 A.2.1.9）：
 *   1 球机 / 2 半球 / 4 遥控枪机 / 5 遥控半球 → 有转台方向
 *   3 固定枪机 / 6 全景通道 / 7 分割通道 → 无转台方向
 * 其余（0 未知）按无处理。
 *
 * 现在只按形态粗判；等后端把细分能力（lens_zoom/focus/iris、preset）下放到
 * ChannelView.caps 后，只需改这一个函数读真值即可，UI 与面板不必动。
 *
 * @param {number|string} ptzType
 * @returns {{direction:boolean, zoom:boolean, focus:boolean, iris:boolean}}
 *   direction=转台方向盘；zoom/focus/iris=电动镜头（变倍/聚焦/光圈）。
 */
export function derivePtzCaps(ptzType) {
  const t = Number(ptzType) || 0
  const hasPT = t === 1 || t === 2 || t === 4 || t === 5
  return {
    direction: hasPT,
    zoom: hasPT,
    focus: hasPT,
    iris: hasPT
  }
}

/**
 * 「不做能力过滤、全部显示」的能力集。
 * 用于不做按通道能力检测的入口（分屏监控器、历史 playerPtzPanel 包裹层），
 * 保持其原有「所有方向/镜头按钮都可见」的行为。新播放器一律传 derivePtzCaps 结果。
 */
export const FULL_PTZ_CAPS = { direction: true, zoom: true, focus: true, iris: true }
