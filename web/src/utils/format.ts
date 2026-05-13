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

export function getCardDisplayNumber(card: CardResponse): string {
  return card.number || card.masked_number
}

export { createIdempotencyKey }
