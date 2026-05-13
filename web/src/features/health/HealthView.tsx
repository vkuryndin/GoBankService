import { RequestStatus } from '../../components/RequestStatus'
import { Button } from '../../components/ui/Button'
import { Card } from '../../components/ui/Card'
import type { HealthResult } from '../../api/healthApi'
import type { RequestState } from '../../types/common'

type HealthViewProps = {
  healthState: RequestState
  healthResult: HealthResult | null
  onCheckHealth: () => void
}

export function HealthView({ healthState, healthResult, onCheckHealth }: HealthViewProps) {
  return (
    <Card variant="plain" className="panel">
      <div className="panelHeader">
        <div>
          <h2>Проверка backend</h2>
          <p>
            Запрос к <code>GET /health</code>.
          </p>
        </div>

        <Button type="button" onClick={onCheckHealth} disabled={healthState.loading}>
          {healthState.loading ? 'Проверяю...' : 'Проверить'}
        </Button>
      </div>

      <RequestStatus state={healthState} />

      {healthResult !== null && (
        <div className={healthState.error ? 'result error' : 'result success'}>
          <strong>
            HTTP status: <code>{healthResult.statusCode}</code>
          </strong>
          <pre>{JSON.stringify(healthResult.body, null, 2)}</pre>
        </div>
      )}
    </Card>
  )
}
