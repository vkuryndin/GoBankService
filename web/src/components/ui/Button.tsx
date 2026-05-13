import type { ButtonHTMLAttributes, ReactNode } from 'react'

type ButtonVariant = 'primary' | 'secondary' | 'danger'

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant
  children: ReactNode
}

function getVariantClass(variant: ButtonVariant): string {
  if (variant === 'secondary') {
    return 'secondary'
  }

  if (variant === 'danger') {
    return 'danger'
  }

  return ''
}

export function Button({
  variant = 'primary',
  className = '',
  children,
  ...props
}: ButtonProps) {
  const variantClass = getVariantClass(variant)
  const buttonClassName = [variantClass, className].filter(Boolean).join(' ')

  return (
    <button className={buttonClassName || undefined} {...props}>
      {children}
    </button>
  )
}
