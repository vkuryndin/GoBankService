import { createContext, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import { createRandomID } from '../utils/random'

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
  const timeoutIDs = useRef(new Map<string, number>())

  const removeToast = useCallback((id: string) => {
    const timeoutID = timeoutIDs.current.get(id)
    if (timeoutID !== undefined) {
      window.clearTimeout(timeoutID)
      timeoutIDs.current.delete(id)
    }

    setToasts((current) => current.filter((toast) => toast.id !== id))
  }, [])

  const showToast = useCallback(
    (message: string, type: ToastType = 'info') => {
      const id = createRandomID()
      setToasts((current) => [...current, { id, type, message }])

      const timeoutID = window.setTimeout(() => removeToast(id), 4000)
      timeoutIDs.current.set(id, timeoutID)
    },
    [removeToast],
  )

  useEffect(() => {
    return () => {
      timeoutIDs.current.forEach((timeoutID) => window.clearTimeout(timeoutID))
      timeoutIDs.current.clear()
    }
  }, [])

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
