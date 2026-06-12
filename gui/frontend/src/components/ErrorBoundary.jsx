import { Component } from 'react'
import { openAppLogFile, reportFrontendError } from '../lib/errorReporting'

export default class ErrorBoundary extends Component {
  constructor(props) {
    super(props)
    this.state = { error: null, errorInfo: null }
  }

  static getDerivedStateFromError(error) {
    return { error }
  }

  componentDidCatch(error, errorInfo) {
    this.setState({ errorInfo })
    reportFrontendError(
      this.props.name || 'ErrorBoundary',
      error,
      errorInfo?.componentStack || '',
    )
  }

  handleRetry = () => {
    this.setState({ error: null, errorInfo: null })
    this.props.onRetry?.()
  }

  render() {
    const { error, errorInfo } = this.state
    if (!error) return this.props.children

    const message = error?.message || String(error)

    return (
      <div className="flex h-full min-h-[12rem] flex-col items-center justify-center rounded-xl border border-red-400/30 bg-red-950/20 p-6 text-center">
        <h2 className="text-lg font-semibold text-red-100">
          {this.props.title || 'Something went wrong'}
        </h2>
        <p className="mt-2 max-w-lg text-sm text-red-200/90">{message}</p>
        {this.props.hint && (
          <p className="mt-2 max-w-lg text-xs text-white/50">{this.props.hint}</p>
        )}
        {errorInfo?.componentStack && (
          <pre className="mt-4 max-h-40 w-full max-w-2xl overflow-auto rounded-lg bg-black/40 p-3 text-left text-xs text-white/60">
            {errorInfo.componentStack.trim()}
          </pre>
        )}
        <p className="mt-3 text-xs text-white/45">
          Details were written to the app log (%APPDATA%\AuraAudioDownloader\logs\app.log).
        </p>
        <div className="mt-4 flex flex-wrap justify-center gap-2">
          <button
            type="button"
            onClick={this.handleRetry}
            className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white hover:bg-accent-muted"
          >
            Try again
          </button>
          <button
            type="button"
            onClick={openAppLogFile}
            className="rounded-lg border border-white/15 px-4 py-2 text-sm text-white/80 hover:bg-white/5"
          >
            Open log file
          </button>
        </div>
      </div>
    )
  }
}
