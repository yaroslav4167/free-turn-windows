import {useEffect, useState, useSyncExternalStore} from 'react'
import './loadCatalogs'
import App from './App'
import {desktop} from './desktop'
import {configureLanguage, languageSnapshot, subscribeLanguage, systemLanguage} from './language'

export default function LocalizedApp() {
  useSyncExternalStore(subscribeLanguage, languageSnapshot)
  const [ready, setReady] = useState(false)
  useEffect(() => {
    let active = true
    Promise.resolve().then(() => desktop().DesktopSettings()).then(settings => {
      if (active) configureLanguage(settings.language, settings.systemLanguage)
    }).catch(() => {
      if (active) configureLanguage('system', systemLanguage(navigator.languages || [navigator.language]))
    }).finally(() => { if (active) setReady(true) })
    return () => { active = false }
  }, [])
  return ready ? <App /> : null
}
