import { HealthView } from '../features/health/HealthView'
import { useHealthCheck } from '../hooks/useHealthCheck'

export function HealthPage() {
  const { healthState, healthResult, checkHealth } = useHealthCheck()

  return (
    <HealthView
      healthState={healthState}
      healthResult={healthResult}
      onCheckHealth={checkHealth}
    />
  )
}
