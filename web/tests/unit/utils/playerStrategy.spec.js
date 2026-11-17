import { resolvePlayerStrategy } from '@/utils/playerStrategy'

describe('resolvePlayerStrategy', () => {
  test('H265 只能用高清播放器', () => {
    const s = resolvePlayerStrategy({ videoCodec: 'H265', audioCodec: 'AAC', hasAudio: true, rtcAvailable: true })
    expect(s.preferredPlayer).toBe('h265web')
    expect(s.allowedPlayers).toEqual(['h265web'])
  })

  test('H264 + AAC 音频：默认标准(Jessibuca)，WebRTC 作为可选(无声)', () => {
    const s = resolvePlayerStrategy({ videoCodec: 'H264', audioCodec: 'AAC', hasAudio: true, rtcAvailable: true })
    expect(s.preferredPlayer).toBe('jessibuca')
    expect(s.allowedPlayers).toEqual(['jessibuca', 'webRTC'])
  })

  test('H264 + 无音频：默认超低延迟(WebRTC)', () => {
    const s = resolvePlayerStrategy({ videoCodec: 'H264', hasAudio: false, rtcAvailable: true })
    expect(s.preferredPlayer).toBe('webRTC')
    expect(s.urlPriority[0]).toBe('rtc')
  })

  test('H264 + OPUS：WebRTC 有声音，优先 WebRTC', () => {
    const s = resolvePlayerStrategy({ videoCodec: 'H264', audioCodec: 'OPUS', hasAudio: true, rtcAvailable: true })
    expect(s.preferredPlayer).toBe('webRTC')
  })

  test('H264 + G711A(PCMA)：WebRTC 原生支持，优先 WebRTC', () => {
    const s = resolvePlayerStrategy({ videoCodec: 'H264', audioCodec: 'G711A', hasAudio: true, rtcAvailable: true })
    expect(s.audioCodec).toBe('PCMA')
    expect(s.webrtcAudioOk).toBe(true)
    expect(s.preferredPlayer).toBe('webRTC')
  })

  test('ZMS 无 WebRTC 能力：只给标准(Jessibuca)', () => {
    const s = resolvePlayerStrategy({ videoCodec: 'H264', audioCodec: 'AAC', hasAudio: true, rtcAvailable: false })
    expect(s.allowedPlayers).toEqual(['jessibuca'])
    expect(s.preferredPlayer).toBe('jessibuca')
  })

  test('编码未知：默认 Jessibuca（不贸然选 WebRTC）', () => {
    const s = resolvePlayerStrategy({ rtcAvailable: true })
    expect(s.preferredPlayer).toBe('jessibuca')
    expect(s.allowedPlayers).toEqual(['jessibuca'])
  })
})
