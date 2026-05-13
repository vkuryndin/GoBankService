import type { RequestState } from '../types/common'
import { RequestStatus } from './RequestStatus'

export function RequestMessage({ state }: { state: RequestState }) {
  return <RequestStatus state={state} />
}
