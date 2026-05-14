import { Button } from './Button'

type ConfirmDialogProps = {
  open: boolean
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  danger?: boolean
  loading?: boolean
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmDialog({
  open,
  title,
  message,
  confirmText = 'Подтвердить',
  cancelText = 'Отмена',
  danger = false,
  loading = false,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  if (!open) {
    return null
  }

  return (
    <div className="confirmOverlay" role="presentation" onMouseDown={onCancel}>
      <section
        className="confirmDialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="confirm-dialog-title"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <h3 id="confirm-dialog-title">{title}</h3>
        <p>{message}</p>
        <div className="actions confirmActions">
          <Button className="secondary" type="button" onClick={onCancel} disabled={loading}>
            {cancelText}
          </Button>
          <Button
            variant={danger ? 'danger' : 'primary'}
            type="button"
            onClick={onConfirm}
            disabled={loading}
          >
            {loading ? 'Выполняю...' : confirmText}
          </Button>
        </div>
      </section>
    </div>
  )
}
