import type { InputHTMLAttributes } from 'react'

type InputProps = InputHTMLAttributes<HTMLInputElement>

export function Input({ className = '', ...props }: InputProps) {
  const inputClassName = ['uiInput', className].filter(Boolean).join(' ')

  return <input className={inputClassName || undefined} {...props} />
}
