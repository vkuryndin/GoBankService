import type { RequestState } from '../types/common'

type RequestStatusProps = {
  state: RequestState
  loadingText?: string
}

export function RequestStatus({
  state,
  loadingText = 'Выполняю запрос...',
}: RequestStatusProps) {
  if (state.loading) {
    return <div className="loadingNotice">{loadingText}</div>
  }

  if (state.error) {
    return <div className="alert">{state.error}</div>
  }

  if (state.success) {
    return <div className="notice">{state.success}</div>
  }

  return null
}
