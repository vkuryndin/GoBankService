import type { ReactNode } from 'react'

type FieldProps = {
  label: string
  children: ReactNode
  hint?: string
  error?: string
}

export function Field({ label, children, hint, error }: FieldProps) {
  return (
    <label className="uiField">
      <span>{label}</span>
      {children}
      {hint && !error && <small className="fieldHint">{hint}</small>}
      {error && <small className="fieldError">{error}</small>}
    </label>
  )
}
