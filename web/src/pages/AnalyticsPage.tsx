import { useEffect, useState } from 'react'
import { RequestStatus } from '../components/RequestStatus'
import { Button } from '../components/ui/Button'
import { Card } from '../components/ui/Card'
import { useAnalytics } from '../hooks/useAnalytics'
import type { AnalyticsResponse } from '../types/analytics'
import { emptyState, type RequestState } from '../types/common'

type AnalyticsPageProps = {
  token: string
}

export function AnalyticsPage({ token }: AnalyticsPageProps) {
  const analyticsDomain = useAnalytics(token)
  const [analyticsState, setAnalyticsState] = useState<RequestState>(emptyState)
  const [analytics, setAnalytics] = useState<AnalyticsResponse | null>(null)

  useEffect(() => {
    if (analyticsDomain.summaryQuery.data) {
      setAnalytics(analyticsDomain.summaryQuery.data)
    }
  }, [analyticsDomain.summaryQuery.data])

  const loadAnalytics = async () => {
    if (!token) {
      setAnalyticsState({ loading: false, error: 'Сначала нужно войти в систему.', success: '' })
      return
    }

    setAnalyticsState({ loading: true, error: '', success: '' })
    setAnalytics(null)

    try {
      const result = await analyticsDomain.summaryQuery.refetch()
      if (result.error) {
        throw result.error
      }

      setAnalytics(result.data || null)
      setAnalyticsState({ loading: false, error: '', success: 'Сводная аналитика загружена.' })
    } catch (error) {
      setAnalyticsState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load analytics',
        success: '',
      })
    }
  }

  return (
    <Card variant="plain" className="panel analyticsPage">
      <div className="panelHeader">
        <div>
          <h2>Аналитика</h2>
          <p>
            Сводка по текущему пользователю. Операционная статистика перенесена в профильные разделы:
            по счету — в “Счета”, по карте — в “Карты”.
          </p>
        </div>

        <div className="actions">
          <Button type="button" onClick={loadAnalytics} disabled={analyticsState.loading || !token}>
            {analyticsState.loading ? 'Загружаю...' : 'Загрузить сводку'}
          </Button>
        </div>
      </div>

      <RequestStatus state={analyticsState} />

      <div className="analyticsDashboard compactAnalyticsDashboard">
        <section className="subPanel analyticsSummaryPanel">
          <div className="subPanelHeader">
            <div>
              <h3>Сводка за месяц</h3>
              <p className="mutedText">Endpoint: <code>GET /analytics</code>.</p>
            </div>
          </div>

          {!analytics && !analyticsState.error && (
            <div className="empty compactEmpty">Нажми “Загрузить сводку”, чтобы получить данные по текущему пользователю.</div>
          )}

          {analytics && (
            <>
              <div className="analyticsCards">
                <div className="analyticsCard incomeCard">
                  <span>Доходы за месяц</span>
                  <strong>{analytics.income_this_month} RUB</strong>
                </div>
                <div className="analyticsCard expenseCard">
                  <span>Расходы за месяц</span>
                  <strong>{analytics.expense_this_month} RUB</strong>
                </div>
                <div className="analyticsCard creditLoadCard">
                  <span>Кредитная нагрузка</span>
                  <strong>{analytics.credit_load} RUB</strong>
                </div>
              </div>

              <details className="rawDetails">
                <summary>Raw response</summary>
                <pre>{JSON.stringify(analytics, null, 2)}</pre>
              </details>
            </>
          )}
        </section>
      </div>
    </Card>
  )
}
