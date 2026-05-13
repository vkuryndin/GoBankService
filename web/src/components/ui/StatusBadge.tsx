import type { HTMLAttributes, ReactNode } from 'react'

type StatusTone = 'success' | 'danger' | 'muted'

type StatusBadgeProps = HTMLAttributes<HTMLSpanElement> & {
  tone?: StatusTone
  children: ReactNode
}

function getToneClass(tone: StatusTone): string {
  if (tone === 'success') {
    return 'successBadge'
  }

  if (tone === 'danger') {
    return 'dangerBadge'
  }

  return 'mutedBadge'
}

export function StatusBadge({
  tone = 'muted',
  className = '',
  children,
  ...props
}: StatusBadgeProps) {
  const badgeClassName = ['badge', getToneClass(tone), className]
    .filter(Boolean)
    .join(' ')

  return (
    <span className={badgeClassName} {...props}>
      {children}
    </span>
  )
}
