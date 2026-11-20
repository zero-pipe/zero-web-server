import defaultSettings from '@/settings'

const title = defaultSettings.title || 'Zeroo Web'

export default function getPageTitle(pageTitle) {
  if (pageTitle) {
    return `${pageTitle} - ${title}`
  }
  return `${title}`
}
