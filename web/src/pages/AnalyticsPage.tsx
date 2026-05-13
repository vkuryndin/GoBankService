import { useState } from 'react'
import type { FormEvent } from 'react'
import { apiRequest } from '../api/http'
import { RequestMessage } from '../components/RequestMessage'
import type { AccountResponse, PredictBalanceResponse } from '../types/account'
import type { AnalyticsResponse } from '../types/analytics'
import { emptyState, type RequestState } from '../types/common'
import {
  getAccountBadgeClass,
  getAccountStatusText,
} from '../utils/format'

type AnalyticsPageProps = {
  token: string
}

export function AnalyticsPage({ token }: AnalyticsPageProps) {
  const [analyticsState, setAnalyticsState] = useState<RequestState>(emptyState)
  const [accountsState, setAccountsState] = useState<RequestState>(emptyState)
  const [predictionState, setPredictionState] = useState<RequestState>(emptyState)

  const [analytics, setAnalytics] = useState<AnalyticsResponse | null>(null)
  const [accounts, setAccounts] = useState<AccountResponse[]>([])
  const [selectedAccountId, setSelectedAccountId] = useState('')
  const [predictionDays, setPredictionDays] = useState('30')
  const [prediction, setPrediction] = useState<PredictBalanceResponse | null>(null)

  const requireToken = (setState: (state: RequestState) => void): boolean => {
    if (token) {
      return true
    }

    setState({
      loading: false,
      error: 'Сначала нужно войти в систему.',
      success: '',
    })
    return false
  }

  const loadAnalytics = async () => {
    if (!requireToken(setAnalyticsState)) {
      return
    }

    setAnalyticsState({
      loading: true,
      error: '',
      success: '',
    })
    setAnalytics(null)

    try {
      const data = await apiRequest<AnalyticsResponse>('/api/analytics', { token })

      setAnalytics(data)
      setAnalyticsState({
        loading: false,
        error: '',
        success: 'Сводная аналитика загружена.',
      })
    } catch (error) {
      setAnalyticsState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load analytics',
        success: '',
      })
    }
  }

  const loadAccounts = async () => {
    if (!requireToken(setAccountsState)) {
      return
    }

    setAccountsState({
      loading: true,
      error: '',
      success: '',
    })

    try {
      const data = await apiRequest<AccountResponse[]>('/api/accounts', { token })
      const list = Array.isArray(data) ? data : []

      setAccounts(list)
      if (list.length > 0 && !selectedAccountId) {
        setSelectedAccountId(String(list[0].id))
      }

      setAccountsState({
        loading: false,
        error: '',
        success: 'Список счетов загружен.',
      })
    } catch (error) {
      setAccountsState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load accounts',
        success: '',
      })
    }
  }

  const selectAccount = (account: AccountResponse) => {
    setSelectedAccountId(String(account.id))
    setPrediction(null)
    setPredictionState(emptyState)
  }

  const loadPrediction = async (event?: FormEvent<HTMLFormElement>) => {
    event?.preventDefault()

    if (!requireToken(setPredictionState)) {
      return
    }

    const accountID = Number(selectedAccountId)
    if (!Number.isInteger(accountID) || accountID <= 0) {
      setPredictionState({
        loading: false,
        error: 'Выбери счет или укажи корректный account_id.',
        success: '',
      })
      return
    }

    const days = Number(predictionDays)
    if (!Number.isInteger(days) || days <= 0 || days > 365) {
      setPredictionState({
        loading: false,
        error: 'Количество дней должно быть от 1 до 365.',
        success: '',
      })
      return
    }

    setPredictionState({
      loading: true,
      error: '',
      success: '',
    })
    setPrediction(null)

    try {
      const data = await apiRequest<PredictBalanceResponse>(
        `/api/accounts/${accountID}/predict?days=${days}`,
        { token },
      )

      setPrediction(data)
      setPredictionState({
        loading: false,
        error: '',
        success: 'Прогноз баланса загружен.',
      })
    } catch (error) {
      setPredictionState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load balance prediction',
        success: '',
      })
    }
  }

  return (
    <section className="panel analyticsPage">
      <div className="panelHeader">
        <div>
          <h2>Аналитика</h2>
          <p>
            Сводная аналитика считается по пользователю, а прогноз баланса — по конкретному счету и количеству дней.
          </p>
        </div>

        <div className="actions">
          <button type="button" onClick={loadAnalytics} disabled={analyticsState.loading || !token}>
            {analyticsState.loading ? 'Загружаю...' : 'Загрузить сводку'}
          </button>
          <button
            className="secondary"
            type="button"
            onClick={loadAccounts}
            disabled={accountsState.loading || !token}
          >
            {accountsState.loading ? 'Загружаю...' : 'Загрузить счета'}
          </button>
        </div>
      </div>

      <RequestMessage state={analyticsState} />
      <RequestMessage state={accountsState} />

      <div className="analyticsDashboard">
        <section className="subPanel analyticsSummaryPanel">
          <div className="subPanelHeader">
            <div>
              <h3>Сводка за месяц</h3>
              <p className="mutedText">Endpoint: <code>GET /analytics</code>.</p>
            </div>
          </div>

          {!analytics && !analyticsState.error && (
            <div className="empty compactEmpty">
              Нажми “Загрузить сводку”, чтобы получить данные по текущему пользователю.
            </div>
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

        <section className="subPanel analyticsAccountsPanel">
          <div className="subPanelHeader">
            <div>
              <h3>Счета</h3>
              <p className="mutedText">Выбери счет для прогноза.</p>
            </div>
            <span>{accounts.length}</span>
          </div>

          {accounts.length === 0 && (
            <div className="empty compactEmpty">
              Нажми “Загрузить счета”.
            </div>
          )}

          {accounts.length > 0 && (
            <div className="analyticsAccountList">
              {accounts.map((account) => (
                <button
                  key={account.id}
                  className={
                    selectedAccountId === String(account.id)
                      ? 'analyticsAccountItem selected'
                      : 'analyticsAccountItem'
                  }
                  type="button"
                  onClick={() => selectAccount(account)}
                >
                  <span className="analyticsAccountMain">
                    <span className="analyticsAccountNumber">{account.account_number}</span>
                    <span className={getAccountBadgeClass(account)}>
                      {getAccountStatusText(account)}
                    </span>
                  </span>
                  <span className="analyticsAccountMeta">
                    <span>ID {account.id}</span>
                    <span>{account.balance} {account.currency}</span>
                  </span>
                </button>
              ))}
            </div>
          )}
        </section>

        <section className="subPanel analyticsPredictionPanel">
          <div className="subPanelHeader">
            <div>
              <h3>Прогноз баланса</h3>
              <p className="mutedText">
                Endpoint: <code>GET /accounts/{'{accountId}'}/predict?days=N</code>.
              </p>
            </div>
          </div>

          <form className="analyticsPredictionForm" onSubmit={loadPrediction}>
            <label>
              <span>Account ID</span>
              <input
                value={selectedAccountId}
                onChange={(event) => setSelectedAccountId(event.target.value)}
                placeholder="ID счета"
              />
            </label>

            <label>
              <span>Days</span>
              <input
                value={predictionDays}
                onChange={(event) => setPredictionDays(event.target.value)}
                placeholder="1-365"
              />
            </label>

            <button type="submit" disabled={predictionState.loading || !token}>
              {predictionState.loading ? 'Считаю...' : 'Получить прогноз'}
            </button>
          </form>

          <RequestMessage state={predictionState} />

          {!prediction && !predictionState.error && (
            <div className="empty compactEmpty">
              Выбери счет, укажи количество дней и нажми “Получить прогноз”.
            </div>
          )}

          {prediction && (
            <>
              <div className="predictionCards">
                <div>
                  <span>Текущий баланс</span>
                  <strong>{prediction.current_balance} RUB</strong>
                </div>
                <div>
                  <span>Прогноз</span>
                  <strong>{prediction.predicted_balance} RUB</strong>
                </div>
                <div>
                  <span>Ожидаемые доходы</span>
                  <strong>{prediction.expected_income} RUB</strong>
                </div>
                <div>
                  <span>Ожидаемые расходы</span>
                  <strong>{prediction.expected_expense} RUB</strong>
                </div>
                <div>
                  <span>Кредитные платежи</span>
                  <strong>{prediction.scheduled_credit_payments} RUB</strong>
                </div>
                <div>
                  <span>Период</span>
                  <strong>{prediction.days} дней</strong>
                </div>
              </div>

              <details className="rawDetails">
                <summary>Raw response</summary>
                <pre>{JSON.stringify(prediction, null, 2)}</pre>
              </details>
            </>
          )}
        </section>
      </div>
    </section>
  )
}
