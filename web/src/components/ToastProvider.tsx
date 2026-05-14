import { createContext, useCallback, useMemo, useState } from 'react'
import type { ReactNode } from 'react'

export type ToastType = 'success' | 'error' | 'info'

export type Toast = {
  id: string
  type: ToastType
  message: string
}

type ToastContextValue = {
  showToast: (message: string, type?: ToastType) => void
  removeToast: (id: string) => void
}

export const ToastContext = createContext<ToastContextValue | null>(null)

type ToastProviderProps = {
  children: ReactNode
}

export function ToastProvider({ children }: ToastProviderProps) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const removeToast = useCallback((id: string) => {
    setToasts((current) => current.filter((toast) => toast.id !== id))
  }, [])

  const showToast = useCallback(
    (message: string, type: ToastType = 'info') => {
      const id = crypto.randomUUID()
      setToasts((current) => [...current, { id, type, message }])
      window.setTimeout(() => removeToast(id), 4000)
    },
    [removeToast],
  )

  const value = useMemo(
    () => ({ showToast, removeToast }),
    [removeToast, showToast],
  )

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="toastViewport" aria-live="polite" aria-relevant="additions removals">
        {toasts.map((toast) => (
          <button
            key={toast.id}
            className={`toast toast-${toast.type}`}
            type="button"
            onClick={() => removeToast(toast.id)}
          >
            {toast.message}
          </button>
        ))}
      </div>
    </ToastContext.Provider>
  )
}
