import { RatesView } from '../features/rates/RatesView'
import { useAuth } from '../hooks/useAuth'
import { useKeyRate } from '../hooks/useKeyRate'

export function RatesPage() {
  const { token } = useAuth()
  const { rate, rateState, loadRate, clearRate } = useKeyRate(token)

  return (
    <RatesView
      token={token}
      rate={rate}
      rateState={rateState}
      onLoadRate={loadRate}
      onClearRate={clearRate}
    />
  )
}
