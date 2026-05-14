import { forwardRef } from 'react'
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

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { variant = 'primary', className = '', children, ...props },
  ref,
) {
  const variantClass = getVariantClass(variant)
  const buttonClassName = [variantClass, className].filter(Boolean).join(' ')

  return (
    <button ref={ref} className={buttonClassName || undefined} {...props}>
      {children}
    </button>
  )
})
