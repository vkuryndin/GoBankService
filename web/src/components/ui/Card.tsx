import type { HTMLAttributes, ReactNode } from 'react'

type CardVariant = 'panel' | 'subPanel' | 'plain'

type CardProps = HTMLAttributes<HTMLElement> & {
  variant?: CardVariant
  children: ReactNode
}

function getVariantClass(variant: CardVariant): string {
  if (variant === 'plain') {
    return ''
  }

  return variant
}

export function Card({
  variant = 'panel',
  className = '',
  children,
  ...props
}: CardProps) {
  const variantClass = getVariantClass(variant)
  const cardClassName = [variantClass, className].filter(Boolean).join(' ')

  return (
    <section className={cardClassName || undefined} {...props}>
      {children}
    </section>
  )
}
