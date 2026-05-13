import type { RequestState } from '../types/common'

export function RequestMessage({ state }: { state: RequestState }) {
  if (state.error) {
    return <div className="alert">{state.error}</div>
  }

  if (state.success) {
    return <div className="notice">{state.success}</div>
  }

  return null
}
