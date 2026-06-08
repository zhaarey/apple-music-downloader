import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import ErrorBoundary from './components/ErrorBoundary'
import { installGlobalErrorHandlers } from './lib/errorReporting'
import './index.css'

installGlobalErrorHandlers()

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <ErrorBoundary name="AppRoot" title="Aura Audio Downloader crashed">
      <App />
    </ErrorBoundary>
  </React.StrictMode>,
)
