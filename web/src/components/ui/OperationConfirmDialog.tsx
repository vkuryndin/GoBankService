import { useEffect, useRef } from 'react'
import type { ReactNode } from 'react'
import { Button } from './Button'

type OperationConfirmDialogProps = {
  open: boolean
  title: string
  message?: string
  children: ReactNode
  result?: ReactNode
  error?: string
  confirmText?: string
  cancelText?: string
  closeText?: string
  loading?: boolean
  onConfirm: () => void
  onClose: () => void
}

export function OperationConfirmDialog({
  open,
  title,
  message,
  children,
  result,
  error,
  confirmText = 'Подтвердить',
  cancelText = 'Отмена',
  closeText = 'Закрыть',
  loading = false,
  onConfirm,
  onClose,
}: OperationConfirmDialogProps) {
  const cancelButtonRef = useRef<HTMLButtonElement>(null)
  const hasFinalState = Boolean(result || error)

  useEffect(() => {
    if (!open) {
      return
    }

    cancelButtonRef.current?.focus()

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !loading) {
        onClose()
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [loading, onClose, open])

  if (!open) {
    return null
  }

  return (
    <div
      className="confirmOverlay"
      role="presentation"
      onMouseDown={() => {
        if (!loading) {
          onClose()
        }
      }}
    >
      <section
        className="confirmDialog operationDialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="operation-dialog-title"
        aria-describedby="operation-dialog-message"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <h3 id="operation-dialog-title">{title}</h3>
        {message && <p id="operation-dialog-message">{message}</p>}

        {children}

        {loading && (
          <div className="operationDialogState operationDialogLoading">
            <strong>Операция выполняется...</strong>
            <span>Дождись ответа backend. Не закрывай страницу.</span>
          </div>
        )}

        {error && !loading && (
          <div className="operationDialogState operationDialogError">
            <strong>Операция не выполнена</strong>
            <span>{error}</span>
          </div>
        )}

        {result && !loading && (
          <div className="operationDialogState operationDialogSuccess">
            <strong>Операция выполнена</strong>
            {result}
          </div>
        )}

        <div className="actions confirmActions">
          <Button
            ref={cancelButtonRef}
            className="secondary"
            type="button"
            onClick={onClose}
            disabled={loading}
          >
            {hasFinalState ? closeText : cancelText}
          </Button>

          {!result && (
            <Button
              type="button"
              onClick={onConfirm}
              disabled={loading}
              aria-busy={loading}
            >
              {loading ? 'Выполняю...' : error ? 'Повторить' : confirmText}
            </Button>
          )}
        </div>
      </section>
    </div>
  )
}
