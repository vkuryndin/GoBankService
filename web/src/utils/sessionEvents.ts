export const sessionExpiredEventName = 'bank-service-session-expired'

export type SessionExpiredEventDetail = {
  message: string
}

export function emitSessionExpired(message = 'Сессия истекла. Войдите снова.') {
  window.dispatchEvent(
    new CustomEvent<SessionExpiredEventDetail>(sessionExpiredEventName, {
      detail: { message },
    }),
  )
}
