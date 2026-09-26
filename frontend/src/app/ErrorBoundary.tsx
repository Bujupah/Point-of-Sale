import { Component, type ReactNode } from 'react'
import { AlertTriangle } from 'lucide-react'

// A cashier-facing POS must never go silently blank (brief's own "never
// show a blank/silent failure" principle) — an uncaught render error should
// surface as a message with a recovery action, not an unmounted white page.
export class ErrorBoundary extends Component<{ children: ReactNode }, { error: Error | null }> {
  constructor(props: { children: ReactNode }) {
    super(props)
    this.state = { error: null }
  }

  static getDerivedStateFromError(error: Error) {
    return { error }
  }

  componentDidCatch(error: Error, info: { componentStack: string }) {
    console.error('Unhandled UI error:', error, info.componentStack)
  }

  render() {
    if (this.state.error) {
      return (
        <div className="auth-screen">
          <div className="auth-card card" style={{ alignItems: 'center', textAlign: 'center' }}>
            <AlertTriangle size={40} className="text-danger" />
            <h1 className="auth-title">Something went wrong</h1>
            <p className="text-muted">{this.state.error.message}</p>
            <button className="btn btn-primary btn-lg btn-block" onClick={() => window.location.reload()}>
              Reload
            </button>
          </div>
        </div>
      )
    }
    return this.props.children
  }
}
