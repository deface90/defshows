import React from 'react'
import ReactDOM from 'react-dom/client'
import { App } from './app/App'
import { Providers } from './app/providers'
import { bootstrapSession, initAuth } from './shared/auth/session'
import './styles.css'

// Wire the auth store into the HTTP layer, then restore any existing session
// before the first render (avoids an auth flash / wrong redirect).
initAuth()

function render() {
  ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
      <Providers>
        <App />
      </Providers>
    </React.StrictMode>,
  )
}

bootstrapSession().finally(render)
