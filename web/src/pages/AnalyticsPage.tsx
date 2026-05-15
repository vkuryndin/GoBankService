import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { queryKeys } from '../api/queryKeys'
import { OperationStatisticsPanel } from '../components/analytics/OperationStatisticsPanel'
import { CreditOperationHistoryView } from '../components/analytics/CreditOperationHistoryView'
import { RequestStatus } from '../components/RequestStatus'
import { Button } from '../components/ui/Button'
import { Card } from '../components/ui/Card'
import { useAccounts } from '../hooks/useAccounts'
import { useAnalytics } from '../hooks/useAnalytics'
import { useCards } from '../hooks/useCards'
import { useCredits } from '../hooks/useCredits'
import type { PredictBalanceResponse } from '../types/account'
import type { OperationStatisticsResponse, AnalyticsResponse } from '../types/analytics'
import type { CreditOperationResponse } from '../types/credit'
import { emptyState, type RequestState } from '../types/common'
import { formatCardNumber } from '../utils/format'
import { validateDays } from '../utils/validation'

type AnalyticsPageProps = {
  token: string
}

function isValidLimit(value: string): boolean {
  const limit = Number(value)
  return Number.isInteger(limit) && limit > 0 && limit <= 500
}

export function AnalyticsPage({ token }: AnalyticsPageProps) {
  const queryClient = useQueryClient()
  const analyticsDomain = useAnalytics(token)
  const accountsDomain = useAccounts(token)
  const cardsDomain = useCards(token)
  const creditsDomain = useCredits(token)

  const [analyticsState, setAnalyticsState] = useState<RequestState>(emptyState)
  const [predictionState, setPredictionState] = useState<RequestState>(emptyState)
  const [accountStatisticsState, setAccountStatisticsState] = useState<RequestState>(emptyState)
  const [cardStatisticsState, setCardStatisticsState] = useState<RequestState>(emptyState)
  const [creditOperationsState, setCreditOperationsState] = useState<RequestState>(emptyState)

  const [analytics, setAnalytics] = useState<AnalyticsResponse | null>(null)
  const [prediction, setPrediction] = useState<PredictBalanceResponse | null>(null)
  const [accountOperationStatistics, setAccountOperationStatistics] =
    useState<OperationStatisticsResponse | null>(null)
  const [cardOperationStatistics, setCardOperationStatistics] =
    useState<OperationStatisticsResponse | null>(null)
  const [creditOperations, setCreditOperations] = useState<CreditOperationResponse[]>([])

  const [predictionAccountId, setPredictionAccountId] = useState('')
  const [predictionDays, setPredictionDays] = useState('30')
  const [accountStatisticsAccountId, setAccountStatisticsAccountId] = useState('')
  const [accountStatisticsLimit, setAccountStatisticsLimit] = useState('100')
  const [cardStatisticsCardId, setCardStatisticsCardId] = useState('')
  const [cardStatisticsLimit, setCardStatisticsLimit] = useState('100')
  const [creditOperationsCreditId, setCreditOperationsCreditId] = useState('')

  const accounts = accountsDomain.accounts
  const cards = cardsDomain.cards
  const credits = creditsDomain.credits

  useEffect(() => {
    if (analyticsDomain.summaryQuery.data) {
      setAnalytics(analyticsDomain.summaryQuery.data)
    }
  }, [analyticsDomain.summaryQuery.data])

  useEffect(() => {
    if (accounts.length === 0) {
      return
    }

    if (!predictionAccountId) {
      setPredictionAccountId(String(accounts[0].id))
    }

    if (!accountStatisticsAccountId) {
      setAccountStatisticsAccountId(String(accounts[0].id))
    }
  }, [accounts, predictionAccountId, accountStatisticsAccountId])

  useEffect(() => {
    if (cards.length > 0 && !cardStatisticsCardId) {
      setCardStatisticsCardId(String(cards[0].id))
    }
  }, [cards, cardStatisticsCardId])

  useEffect(() => {
    if (credits.length > 0 && !creditOperationsCreditId) {
      setCreditOperationsCreditId(String(credits[0].id))
    }
  }, [credits, creditOperationsCreditId])

  useEffect(() => {
    const accountID = Number(accountStatisticsAccountId)
    const limit = Number(accountStatisticsLimit)

    if (!Number.isInteger(accountID) || accountID <= 0 || !Number.isInteger(limit)) {
      setAccountOperationStatistics(null)
      return
    }

    const cachedStatistics = queryClient.getQueryData<OperationStatisticsResponse>(
      queryKeys.accounts.operationStatistics(accountID, limit),
    )
    setAccountOperationStatistics(cachedStatistics || null)
  }, [accountStatisticsAccountId, accountStatisticsLimit, queryClient])

  useEffect(() => {
    const cardID = Number(cardStatisticsCardId)
    const limit = Number(cardStatisticsLimit)

    if (!Number.isInteger(cardID) || cardID <= 0 || !Number.isInteger(limit)) {
      setCardOperationStatistics(null)
      return
    }

    const cachedStatistics = queryClient.getQueryData<OperationStatisticsResponse>(
      queryKeys.cards.operationStatistics(cardID, limit),
    )
    setCardOperationStatistics(cachedStatistics || null)
  }, [cardStatisticsCardId, cardStatisticsLimit, queryClient])

  useEffect(() => {
    const creditID = Number(creditOperationsCreditId)

    if (!Number.isInteger(creditID) || creditID <= 0) {
      setCreditOperations([])
      return
    }

    const cachedOperations = queryClient.getQueryData<CreditOperationResponse[]>(
      queryKeys.credits.operations(creditID),
    )
    setCreditOperations(cachedOperations || [])
  }, [creditOperationsCreditId, queryClient])

  const requireToken = (setState: (state: RequestState) => void): boolean => {
    if (token) {
      return true
    }

    setState({ loading: false, error: 'Сначала нужно войти в систему.', success: '' })
    return false
  }

  const loadAnalytics = async () => {
    if (!requireToken(setAnalyticsState)) {
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

  const loadBalancePrediction = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!requireToken(setPredictionState)) {
      return
    }

    const accountID = Number(predictionAccountId)
    if (!Number.isInteger(accountID) || accountID <= 0) {
      setPredictionState({ loading: false, error: 'Выбери счет для прогноза.', success: '' })
      return
    }

    const daysError = validateDays(predictionDays)
    if (daysError) {
      setPredictionState({ loading: false, error: daysError, success: '' })
      return
    }

    setPredictionState({ loading: true, error: '', success: '' })
    setPrediction(null)

    try {
      const days = Number(predictionDays)
      const data = await accountsDomain.predictionMutation.mutateAsync({ accountID, days })

      setPrediction(data)
      setPredictionState({ loading: false, error: '', success: 'Прогноз баланса получен.' })
    } catch (error) {
      setPredictionState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load prediction',
        success: '',
      })
    }
  }

  const loadAccountOperationStatistics = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!requireToken(setAccountStatisticsState)) {
      return
    }

    const accountID = Number(accountStatisticsAccountId)
    if (!Number.isInteger(accountID) || accountID <= 0) {
      setAccountStatisticsState({ loading: false, error: 'Выбери счет.', success: '' })
      return
    }

    if (!isValidLimit(accountStatisticsLimit)) {
      setAccountStatisticsState({ loading: false, error: 'Limit должен быть от 1 до 500.', success: '' })
      return
    }

    setAccountStatisticsState({ loading: true, error: '', success: '' })
    setAccountOperationStatistics(null)

    try {
      const data = await accountsDomain.operationStatisticsMutation.mutateAsync({
        accountID,
        limit: Number(accountStatisticsLimit),
      })

      setAccountOperationStatistics(data)
      setAccountStatisticsState({
        loading: false,
        error: '',
        success: 'Статистика операций по счету загружена.',
      })
    } catch (error) {
      setAccountStatisticsState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load account operation statistics',
        success: '',
      })
    }
  }

  const loadCardOperationStatistics = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!requireToken(setCardStatisticsState)) {
      return
    }

    const cardID = Number(cardStatisticsCardId)
    if (!Number.isInteger(cardID) || cardID <= 0) {
      setCardStatisticsState({ loading: false, error: 'Выбери карту.', success: '' })
      return
    }

    if (!isValidLimit(cardStatisticsLimit)) {
      setCardStatisticsState({ loading: false, error: 'Limit должен быть от 1 до 500.', success: '' })
      return
    }

    setCardStatisticsState({ loading: true, error: '', success: '' })
    setCardOperationStatistics(null)

    try {
      const data = await cardsDomain.operationStatisticsMutation.mutateAsync({
        cardID,
        limit: Number(cardStatisticsLimit),
      })

      setCardOperationStatistics(data)
      setCardStatisticsState({
        loading: false,
        error: '',
        success: 'Статистика операций по карте загружена.',
      })
    } catch (error) {
      setCardStatisticsState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load card operation statistics',
        success: '',
      })
    }
  }

  const loadCreditOperations = async () => {
    if (!requireToken(setCreditOperationsState)) {
      return
    }

    const creditID = Number(creditOperationsCreditId)
    if (!Number.isInteger(creditID) || creditID <= 0) {
      setCreditOperationsState({ loading: false, error: 'Выбери кредит.', success: '' })
      return
    }

    setCreditOperationsState({ loading: true, error: '', success: '' })
    setCreditOperations([])

    try {
      const data = await creditsDomain.operationsMutation.mutateAsync(creditID)

      setCreditOperations(Array.isArray(data) ? data : [])
      setCreditOperationsState({
        loading: false,
        error: '',
        success: 'История операций по кредиту загружена.',
      })
    } catch (error) {
      setCreditOperationsState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load credit operation history',
        success: '',
      })
    }
  }

  return (
    <Card variant="plain" className="panel analyticsHubPage">
      <div className="panelHeader">
        <div>
          <h2>Аналитика</h2>
          <p>
            Хаб аналитики: месячная сводка, прогноз баланса, операции по счетам, картам и кредитам.
          </p>
        </div>

        <div className="actions">
          <Button type="button" onClick={loadAnalytics} disabled={analyticsState.loading || !token}>
            {analyticsState.loading ? 'Загружаю...' : 'Загрузить сводку'}
          </Button>
        </div>
      </div>

      <section className="subPanel analyticsSummaryPanel">
        <div className="subPanelHeader">
          <div>
            <h3>Сводка за месяц</h3>
            <p className="mutedText">Endpoint: <code>GET /analytics</code>.</p>
          </div>
        </div>

        <RequestStatus state={analyticsState} />

        {analytics ? (
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
              <div className="analyticsCard activeCreditsCard">
                <span>Активные кредиты</span>
                <strong>{analytics.active_credits_count}</strong>
              </div>
            </div>

            <details className="rawDetails">
              <summary>Raw response</summary>
              <pre>{JSON.stringify(analytics, null, 2)}</pre>
            </details>
          </>
        ) : (
          <div className="empty compactEmpty">Нажми “Загрузить сводку”, чтобы получить месячную аналитику.</div>
        )}
      </section>

      <section className="subPanel analyticsHubSection">
        <div className="subPanelHeader">
          <div>
            <h3>Прогноз баланса</h3>
            <p className="mutedText">
              Прогноз перенесён сюда из раздела “Счета”. Endpoint: <code>GET /accounts/{'{accountId}'}/predict?days=N</code>.
            </p>
          </div>
        </div>

        <form className="analyticsHubControls" onSubmit={loadBalancePrediction}>
          <label>
            <span>Account</span>
            <select
              value={predictionAccountId}
              onChange={(event) => setPredictionAccountId(event.target.value)}
            >
              {accounts.length === 0 && <option value="">Нет счетов</option>}
              {accounts.map((account) => (
                <option key={account.id} value={account.id}>
                  ID {account.id} · {account.account_number} · {account.balance} {account.currency}
                </option>
              ))}
            </select>
          </label>

          <label>
            <span>Days</span>
            <input
              value={predictionDays}
              onChange={(event) => setPredictionDays(event.target.value)}
              placeholder="1-365"
            />
          </label>

          <Button type="submit" disabled={predictionState.loading || accounts.length === 0}>
            {predictionState.loading ? 'Считаю...' : 'Получить прогноз'}
          </Button>
        </form>

        <RequestStatus state={predictionState} />

        {prediction && (
          <div className="analyticsPredictionGrid">
            <div><span>Текущий баланс</span><strong>{prediction.current_balance} RUB</strong></div>
            <div><span>Прогнозный баланс</span><strong>{prediction.predicted_balance} RUB</strong></div>
            <div><span>Ожидаемые доходы</span><strong>{prediction.expected_income} RUB</strong></div>
            <div><span>Ожидаемые расходы</span><strong>{prediction.expected_expense} RUB</strong></div>
            <div><span>Кредитные платежи</span><strong>{prediction.scheduled_credit_payments} RUB</strong></div>
            <div><span>Период статистики</span><strong>{prediction.statistics_period_days} дней</strong></div>
          </div>
        )}

        {prediction && (
          <details className="rawDetails">
            <summary>Raw prediction</summary>
            <pre>{JSON.stringify(prediction, null, 2)}</pre>
          </details>
        )}
      </section>

      <section className="subPanel analyticsHubSection">
        <div className="subPanelHeader">
          <div>
            <h3>Счета</h3>
            <p className="mutedText">Выбери счет и загрузи статистику операций по нему.</p>
          </div>
        </div>

        <div className="analyticsHubControls">
          <label>
            <span>Account</span>
            <select
              value={accountStatisticsAccountId}
              onChange={(event) => setAccountStatisticsAccountId(event.target.value)}
            >
              {accounts.length === 0 && <option value="">Нет счетов</option>}
              {accounts.map((account) => (
                <option key={account.id} value={account.id}>
                  ID {account.id} · {account.account_number} · {account.balance} {account.currency}
                </option>
              ))}
            </select>
          </label>
        </div>

        <OperationStatisticsPanel
          title="Операции по счету"
          description="Статистика и последние операции по выбранному счету."
          endpointLabel="GET /accounts/{accountId}/operations/statistics"
          limit={accountStatisticsLimit}
          state={accountStatisticsState}
          statistics={accountOperationStatistics}
          disabled={accounts.length === 0}
          emptyText="Выбери счет и нажми “Получить статистику”."
          onLimitChange={setAccountStatisticsLimit}
          onSubmit={loadAccountOperationStatistics}
        />
      </section>

      <section className="subPanel analyticsHubSection">
        <div className="subPanelHeader">
          <div>
            <h3>Карты</h3>
            <p className="mutedText">Выбери карту и загрузи статистику операций по ней.</p>
          </div>
        </div>

        <div className="analyticsHubControls">
          <label>
            <span>Card</span>
            <select
              value={cardStatisticsCardId}
              onChange={(event) => setCardStatisticsCardId(event.target.value)}
            >
              {cards.length === 0 && <option value="">Нет карт</option>}
              {cards.map((card) => (
                <option key={card.id} value={card.id}>
                  ID {card.id} · {formatCardNumber(card.masked_number)} · account {card.account_id} · {card.status}
                </option>
              ))}
            </select>
          </label>
        </div>

        <OperationStatisticsPanel
          title="Операции по карте"
          description="Статистика и последние операции по выбранной карте."
          endpointLabel="GET /cards/{cardId}/operations/statistics"
          limit={cardStatisticsLimit}
          state={cardStatisticsState}
          statistics={cardOperationStatistics}
          disabled={cards.length === 0}
          emptyText="Выбери карту и нажми “Получить статистику”."
          onLimitChange={setCardStatisticsLimit}
          onSubmit={loadCardOperationStatistics}
        />
      </section>

      <section className="subPanel analyticsHubSection">
        <div className="subPanelHeader">
          <div>
            <h3>Кредиты</h3>
            <p className="mutedText">Выбери кредит и загрузи историю операций по нему.</p>
          </div>
        </div>

        <div className="analyticsHubControls">
          <label>
            <span>Credit</span>
            <select
              value={creditOperationsCreditId}
              onChange={(event) => setCreditOperationsCreditId(event.target.value)}
            >
              {credits.length === 0 && <option value="">Нет кредитов</option>}
              {credits.map((credit) => (
                <option key={credit.id} value={credit.id}>
                  credit_id {credit.id} · account {credit.account_id} · {credit.principal_amount} RUB · {credit.status}
                </option>
              ))}
            </select>
          </label>

          <Button
            type="button"
            onClick={loadCreditOperations}
            disabled={creditOperationsState.loading || credits.length === 0}
          >
            {creditOperationsState.loading ? 'Загружаю...' : 'Показать историю операций'}
          </Button>
        </div>

        <RequestStatus state={creditOperationsState} />

        {creditOperations.length === 0 && !creditOperationsState.error && (
          <div className="empty compactEmpty">Выбери кредит и нажми “Показать историю операций”.</div>
        )}

        <CreditOperationHistoryView operations={creditOperations} />

        {creditOperations.length > 0 && (
          <details className="rawDetails">
            <summary>Raw credit operations</summary>
            <pre>{JSON.stringify(creditOperations, null, 2)}</pre>
          </details>
        )}
      </section>
    </Card>
  )
}
