import defaultSettings from '@/settings'

const title = defaultSettings.title || 'Zero Web Kit'

export default function getPageTitle(pageTitle) {
  if (pageTitle) {
    return `${pageTitle} - ${title}`
  }
  return `${title}`
}
