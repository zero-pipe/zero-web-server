/**
 * 播放策略：把 jessibuca / h265web / webRTC 三个引擎“编排”成一个用户无感的播放器。
 * 唯一决策入口 resolvePlayerStrategy，输入视频/音频编码与 ZMS WebRTC 能力，
 * 输出首选播放器、可用播放器列表与拉流 URL 优先级。
 *
 * 能力来源：ZMS 上报的 WebRTC 能力已体现在点播结果里——rtc 字段存在即代表可用，
 * 因此前端用 rtcAvailable = !!streamInfo.rtc 判断，无需硬开关。
 */

/** 用户可见文案（不暴露引擎名称/术语）。 */
export const PLAYER_LABELS = {
  jessibuca: '标准',
  h265web: '高清(H265)',
  webRTC: '超低延迟'
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
const RTC_URL_PRIORITY = ['rtc', 'rtcs'].concat(FLV_URL_PRIORITY)

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

  // 1) H265：浏览器软解只有 h265web 能放，唯一选项。
  if (videoCodec === 'H265') {
    return {
      preferredPlayer: 'h265web',
      allowedPlayers: ['h265web'],
      urlPriority: FLV_URL_PRIORITY.slice(),
      videoCodec,
      audioCodec,
      hasAudio,
      webrtcAudioOk: false
    }
  }

  // 2) H264 / 未知：以 Jessibuca(HTTP-FLV) 兜底，兼容性与音频最稳。
  const allowed = ['jessibuca']
  let preferred = 'jessibuca'
  const webrtcAudioOk = !hasAudio || isWebRTCAudioCodec(audioCodec)

  // 仅当“已知是 H264”且 ZMS 有 WebRTC 能力时才提供 WebRTC，避免未知编码（可能是 H265）误选。
  // 选择 A：有音频且非 WebRTC 可用编码（如 AAC）时，默认仍走 Jessibuca（有声）；WebRTC 作为「超低延迟(无声)」手动可选。
  if (rtcAvailable && videoCodec === 'H264') {
    allowed.push('webRTC')
    if (webrtcAudioOk) {
      preferred = 'webRTC'
    }
  }

  const urlPriority = preferred === 'webRTC' ? RTC_URL_PRIORITY.slice() : FLV_URL_PRIORITY.slice()
  return { preferredPlayer: preferred, allowedPlayers: allowed, urlPriority, videoCodec, audioCodec, hasAudio, webrtcAudioOk }
}
