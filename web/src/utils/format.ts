import { createIdempotencyKey } from './idempotency'
import type { AccountResponse } from '../types/account'
import type { CardResponse } from '../types/card'

export function formatDate(value?: string): string {
  if (!value) {
    return '-'
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return date.toLocaleString('ru-RU')
}

export function isAccountClosed(account?: AccountResponse | null): boolean {
  return account?.status === 'closed'
}

export function getAccountBadgeClass(account: AccountResponse): string {
  if (account.status === 'closed') {
    return 'badge mutedBadge'
  }

  if (account.is_blocked) {
    return 'badge dangerBadge'
  }

  return 'badge successBadge'
}

export function getAccountStatusText(account: AccountResponse): string {
  if (account.status === 'closed') {
    return 'closed'
  }

  if (account.is_blocked) {
    return 'blocked'
  }

  return 'active'
}

export function isCardClosed(card?: CardResponse | null): boolean {
  return card?.status === 'closed'
}

export function getCardBadgeClass(card: CardResponse): string {
  if (card.status === 'closed') {
    return 'badge mutedBadge'
  }

  return 'badge successBadge'
}

export function getCardStatusText(card: CardResponse): string {
  if (card.status === 'closed') {
    return 'closed'
  }

  return 'active'
}

export function formatCardNumber(value?: string): string {
  if (!value) {
    return '-'
  }

  const normalized = value.trim()
  if (!normalized) {
    return '-'
  }

  const digitsOnly = normalized.replace(/\D/g, '')
  const valueWithoutSpacesAndHyphens = normalized.replace(/[\s-]/g, '')
  const hasMaskSymbols = /[*•xX]/.test(normalized)

  if (
    !hasMaskSymbols &&
    digitsOnly.length >= 12 &&
    digitsOnly.length <= 19 &&
    valueWithoutSpacesAndHyphens === digitsOnly
  ) {
    return digitsOnly.replace(/(\d{4})(?=\d)/g, '$1 ').trim()
  }

  return normalized
}

export function getCardDisplayNumber(card: CardResponse): string {
  return card.number ? formatCardNumber(card.number) : formatCardNumber(card.masked_number)
}

export { createIdempotencyKey }
