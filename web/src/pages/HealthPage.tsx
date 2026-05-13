import { useState } from 'react'
import { readResponseBody, getErrorMessage } from '../api/http'
import { RequestStatus } from '../components/RequestStatus'
import { emptyState, type RequestState } from '../types/common'
import { Button } from '../components/ui/Button'
import { Card } from '../components/ui/Card'

type HealthResult = {
  statusCode: number
  body: unknown
}

export function HealthPage() {
  const [healthState, setHealthState] = useState<RequestState>(emptyState)
  const [healthResult, setHealthResult] = useState<HealthResult | null>(null)

  const checkHealth = async () => {
    setHealthState({
      loading: true,
      error: '',
      success: '',
    })
    setHealthResult(null)

    try {
      const response = await fetch('/api/health', {
        method: 'GET',
        headers: {
          Accept: 'application/json',
        },
      })

      const body = await readResponseBody(response)

      setHealthResult({
        statusCode: response.status,
        body,
      })

      setHealthState({
        loading: false,
        error: response.ok ? '' : getErrorMessage(body) || `HTTP ${response.status}`,
        success: response.ok ? 'Backend отвечает.' : '',
      })
    } catch (error) {
      setHealthState({
        loading: false,
        error: error instanceof Error ? error.message : 'Health check failed',
        success: '',
      })
    }
  }

  return (
    <Card variant="plain" className="panel">
      <div className="panelHeader">
        <div>
          <h2>Проверка backend</h2>
          <p>
            Запрос к <code>GET /health</code>.
          </p>
        </div>

        <Button type="button" onClick={checkHealth} disabled={healthState.loading}>
          {healthState.loading ? 'Проверяю...' : 'Проверить'}
        </Button>
      </div>

      <RequestStatus state={healthState} />

      {healthResult && (
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
