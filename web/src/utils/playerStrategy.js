/** WebRTC 支持的音频编码；WebRTC 修好后与无声流均优先 WebRTC */
export const WEBRTC_AUDIO_CODECS = ['OPUS', 'PCMU', 'PCMA', 'G722']

/** WebRTC 播放器尚未就绪，先只用 Jessibuca / H265web */
export const WEBRTC_PLAYER_ENABLED = false

export function normalizeVideoCodec(codec) {
  if (!codec) return ''
  const c = String(codec).toUpperCase().trim()
  if (c.includes('265') || c === 'HEVC') return 'H265'
  if (c.includes('264') || c === 'AVC') return 'H264'
  return c
}

export function normalizeAudioCodec(codec) {
  if (!codec) return ''
  const c = String(codec).toUpperCase().trim()
  if (c === 'MPEG4-GENERIC' || c === 'AAC') return 'AAC'
  return c
}

export function isH265Codec(codec) {
  return normalizeVideoCodec(codec) === 'H265'
}

export function isWebRTCAudioCodec(audioCodec) {
  const c = normalizeAudioCodec(audioCodec)
  return WEBRTC_AUDIO_CODECS.indexOf(c) >= 0
}

/**
 * 按实测编码与音频选择播放器与 URL 优先级。
 * @returns {{ preferredPlayer, allowedPlayers, urlPriority, videoCodec, audioCodec }}
 */
export function resolvePlayerStrategy(opts) {
  opts = opts || {}
  const videoCodec = normalizeVideoCodec(opts.videoCodec || opts.configCodec)
  const audioCodec = normalizeAudioCodec(opts.audioCodec)
  const hasAudio = opts.hasAudio !== false && (!!audioCodec || opts.hasAudio === true)

  let preferredPlayer = 'jessibuca'
  let allowedPlayers = ['jessibuca']
  let urlPriority = ['flv', 'ws_flv', 'https_flv', 'wss_flv']

  if (videoCodec === 'H265') {
    preferredPlayer = 'h265web'
    allowedPlayers = ['h265web']
  }

  // WebRTC 就绪后：Opus/PCMU 等，或无声，均优先 WebRTC
  if (WEBRTC_PLAYER_ENABLED && (isWebRTCAudioCodec(audioCodec) || !hasAudio)) {
    preferredPlayer = 'webRTC'
    allowedPlayers = ['webRTC'].concat(allowedPlayers)
    urlPriority = ['rtc', 'rtcs'].concat(urlPriority)
  }

  return {
    preferredPlayer,
    allowedPlayers,
    urlPriority,
    videoCodec,
    audioCodec,
    hasAudio
  }
}
