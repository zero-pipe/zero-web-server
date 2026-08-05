export function gbTypeCode(deviceId) {
  if (!deviceId || deviceId.length < 13) {
    return ''
  }
  return deviceId.substring(10, 13)
}

export function isBusinessGroupNode(group) {
  if (!group || !group.deviceId) {
    return false
  }
  if (group.deviceId === group.businessGroup) {
    return true
  }
  return gbTypeCode(group.deviceId) === '215'
}

export function isVirtualOrgNode(group) {
  return !!(group && group.deviceId && gbTypeCode(group.deviceId) === '216')
}
