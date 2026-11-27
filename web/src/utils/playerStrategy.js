/**
 * 播放相关工具：用户可见文案 + 云台/镜头能力集推导。
 * 网页端统一使用 Jessibuca 单引擎（播放器实例见 views/common/jessibuca.vue）。
 */

/** 用户可见文案（不暴露引擎名称/术语）。 */
export const PLAYER_LABELS = {
  jessibuca: '标准'
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
