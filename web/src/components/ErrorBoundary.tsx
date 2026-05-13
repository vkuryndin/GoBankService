import { Component } from 'react'
import type { ErrorInfo, ReactNode } from 'react'
import { Button } from './ui/Button'

type ErrorBoundaryProps = {
  children: ReactNode
}

type ErrorBoundaryState = {
  hasError: boolean
  errorMessage: string
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = {
    hasError: false,
    errorMessage: '',
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return {
      hasError: true,
      errorMessage: error.message,
    }
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('Frontend error boundary:', error, errorInfo)
  }

  render() {
    if (!this.state.hasError) {
      return this.props.children
    }

    return (
      <main className="errorBoundaryPage">
        <section className="errorBoundaryCard">
          <p className="eyebrow">Frontend error</p>
          <h1>Что-то пошло не так</h1>
          <p>
            Интерфейс поймал непредвиденную ошибку. Можно перезагрузить страницу и
            продолжить проверку.
          </p>
          {this.state.errorMessage && <pre>{this.state.errorMessage}</pre>}
          <Button type="button" onClick={() => window.location.reload()}>
            Перезагрузить страницу
          </Button>
        </section>
      </main>
    )
  }
}
