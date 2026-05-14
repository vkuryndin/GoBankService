import type { ReactNode } from 'react'

type EmptyStateProps = {
  children: ReactNode
  className?: string
}

export function EmptyState({ children, className = '' }: EmptyStateProps) {
  const emptyClassName = ['empty', className].filter(Boolean).join(' ')

  return <div className={emptyClassName}>{children}</div>
}
