import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { apiRequest } from '../api/http'
import { RequestMessage } from '../components/RequestMessage'
import type { AccountResponse, CloseAccountResponse, PredictBalanceResponse } from '../types/account'
import { emptyState, type RequestState } from '../types/common'
import {
  createIdempotencyKey,
  formatDate,
  getAccountBadgeClass,
  getAccountStatusText,
  isAccountClosed,
} from '../utils/format'

type AccountsPageProps = {
  token: string
  sharedAccountId: string
  onSharedAccountIdChange: (accountId: string) => void
}

export function AccountsPage({
  token,
  sharedAccountId,
  onSharedAccountIdChange,
}: AccountsPageProps) {
  const [accountsState, setAccountsState] = useState<RequestState>(emptyState)
  const [createAccountState, setCreateAccountState] = useState<RequestState>(emptyState)
  const [accountDetailsState, setAccountDetailsState] = useState<RequestState>(emptyState)
  const [depositState, setDepositState] = useState<RequestState>(emptyState)
  const [withdrawMfaState, setWithdrawMfaState] = useState<RequestState>(emptyState)
  const [withdrawState, setWithdrawState] = useState<RequestState>(emptyState)
  const [predictState, setPredictState] = useState<RequestState>(emptyState)
  const [closeAccountState, setCloseAccountState] = useState<RequestState>(emptyState)

  const [accounts, setAccounts] = useState<AccountResponse[]>([])
  const [selectedAccountId, setSelectedAccountId] = useState(sharedAccountId)
  const [selectedAccount, setSelectedAccount] = useState<AccountResponse | null>(null)
  const [depositAmount, setDepositAmount] = useState('100.00')
  const [withdrawAmount, setWithdrawAmount] = useState('50.00')
  const [withdrawMfaCode, setWithdrawMfaCode] = useState('')
  const [predictDays, setPredictDays] = useState('30')
  const [predictResult, setPredictResult] = useState<PredictBalanceResponse | null>(null)
  const [closeResult, setCloseResult] = useState<CloseAccountResponse | null>(null)

  useEffect(() => {
    if (sharedAccountId && !selectedAccountId) {
      setSelectedAccountId(sharedAccountId)
    }
  }, [sharedAccountId, selectedAccountId])

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

  const selectedAccountIDNumber = (): number | null => {
    const id = Number(selectedAccountId)
    return Number.isInteger(id) && id > 0 ? id : null
  }

  const selectAccount = (account: AccountResponse) => {
    setSelectedAccountId(String(account.id))
    setSelectedAccount(account)
    onSharedAccountIdChange(String(account.id))
    setPredictResult(null)
    setCloseResult(null)
  }

  const upsertAccount = (account: AccountResponse) => {
    setAccounts((current) => {
      const exists = current.some((item) => item.id === account.id)
      return exists
        ? current.map((item) => (item.id === account.id ? account : item))
        : [account, ...current]
    })
    selectAccount(account)
  }

  const applyClosedAccount = (response: CloseAccountResponse) => {
    setAccounts((current) =>
      current.map((account) =>
        account.id === response.id
          ? { ...account, status: response.status, closed_at: response.closed_at }
          : account,
      ),
    )

    setSelectedAccount((account) =>
      account && account.id === response.id
        ? { ...account, status: response.status, closed_at: response.closed_at }
        : account,
    )
  }

  const loadAccounts = async () => {
    if (!requireToken(setAccountsState)) {
      return
    }

    setAccountsState({ loading: true, error: '', success: '' })

    try {
      const data = await apiRequest<AccountResponse[]>('/api/accounts', { token })
      const list = Array.isArray(data) ? data : []
      setAccounts(list)
      setAccountsState({ loading: false, error: '', success: 'Список счетов загружен.' })

      if (list.length > 0) {
        const account =
          list.find((item) => String(item.id) === selectedAccountId) || list[0]
        selectAccount(account)
      } else {
        setSelectedAccountId('')
        setSelectedAccount(null)
      }
    } catch (error) {
      setAccountsState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load accounts',
        success: '',
      })
    }
  }

  const createAccount = async () => {
    if (!requireToken(setCreateAccountState)) {
      return
    }

    setCreateAccountState({ loading: true, error: '', success: '' })

    try {
      const account = await apiRequest<AccountResponse>('/api/accounts', {
        method: 'POST',
        token,
      })

      upsertAccount(account)
      setCreateAccountState({
        loading: false,
        error: '',
        success: `Счет создан: ${account.account_number}.`,
      })
    } catch (error) {
      setCreateAccountState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to create account',
        success: '',
      })
    }
  }

  const loadAccountDetails = async () => {
    if (!requireToken(setAccountDetailsState)) {
      return
    }

    const accountID = selectedAccountIDNumber()
    if (!accountID) {
      setAccountDetailsState({ loading: false, error: 'Выбери счет.', success: '' })
      return
    }

    setAccountDetailsState({ loading: true, error: '', success: '' })

    try {
      const account = await apiRequest<AccountResponse>(`/api/accounts/${accountID}`, {
        token,
      })
      upsertAccount(account)
      setAccountDetailsState({ loading: false, error: '', success: 'Данные счета обновлены.' })
    } catch (error) {
      setAccountDetailsState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load account',
        success: '',
      })
    }
  }

  const handleDeposit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!requireToken(setDepositState)) {
      return
    }

    const accountID = selectedAccountIDNumber()
    if (!accountID) {
      setDepositState({ loading: false, error: 'Выбери счет.', success: '' })
      return
    }

    setDepositState({ loading: true, error: '', success: '' })

    try {
      const account = await apiRequest<AccountResponse>(
        `/api/accounts/${accountID}/deposit`,
        {
          method: 'POST',
          token,
          headers: { 'Idempotency-Key': createIdempotencyKey() },
          body: { amount: depositAmount },
        },
      )

      upsertAccount(account)
      setDepositState({
        loading: false,
        error: '',
        success: `Счет пополнен на ${depositAmount} RUB.`,
      })
    } catch (error) {
      setDepositState({
        loading: false,
        error: error instanceof Error ? error.message : 'Deposit failed',
        success: '',
      })
    }
  }

  const requestWithdrawMFA = async () => {
    if (!requireToken(setWithdrawMfaState)) {
      return
    }

    const accountID = selectedAccountIDNumber()
    if (!accountID) {
      setWithdrawMfaState({ loading: false, error: 'Выбери счет.', success: '' })
      return
    }

    setWithdrawMfaState({ loading: true, error: '', success: '' })

    try {
      await apiRequest<{ message: string }>('/api/mfa/request', {
        method: 'POST',
        token,
        body: {
          purpose: 'withdraw',
          account_id: accountID,
          amount: withdrawAmount,
        },
      })

      setWithdrawMfaState({
        loading: false,
        error: '',
        success: 'MFA-код для списания отправлен.',
      })
    } catch (error) {
      setWithdrawMfaState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to request MFA code',
        success: '',
      })
    }
  }

  const handleWithdraw = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!requireToken(setWithdrawState)) {
      return
    }

    const accountID = selectedAccountIDNumber()
    if (!accountID) {
      setWithdrawState({ loading: false, error: 'Выбери счет.', success: '' })
      return
    }

    setWithdrawState({ loading: true, error: '', success: '' })

    try {
      const account = await apiRequest<AccountResponse>(
        `/api/accounts/${accountID}/withdraw`,
        {
          method: 'POST',
          token,
          headers: { 'Idempotency-Key': createIdempotencyKey() },
          body: {
            amount: withdrawAmount,
            mfa_code: withdrawMfaCode,
          },
        },
      )

      upsertAccount(account)
      setWithdrawMfaCode('')
      setWithdrawState({
        loading: false,
        error: '',
        success: `Со счета списано ${withdrawAmount} RUB.`,
      })
    } catch (error) {
      setWithdrawState({
        loading: false,
        error: error instanceof Error ? error.message : 'Withdraw failed',
        success: '',
      })
    }
  }

  const loadPrediction = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!requireToken(setPredictState)) {
      return
    }

    const accountID = selectedAccountIDNumber()
    if (!accountID) {
      setPredictState({ loading: false, error: 'Выбери счет.', success: '' })
      return
    }

    setPredictState({ loading: true, error: '', success: '' })
    setPredictResult(null)

    try {
      const days = Number(predictDays)
      const query = Number.isInteger(days) && days > 0 ? `?days=${days}` : ''
      const data = await apiRequest<PredictBalanceResponse>(
        `/api/accounts/${accountID}/predict${query}`,
        { token },
      )

      setPredictResult(data)
      setPredictState({ loading: false, error: '', success: 'Прогноз баланса получен.' })
    } catch (error) {
      setPredictState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load prediction',
        success: '',
      })
    }
  }

  const closeAccount = async () => {
    if (!requireToken(setCloseAccountState)) {
      return
    }

    const accountID = selectedAccountIDNumber()
    if (!accountID) {
      setCloseAccountState({ loading: false, error: 'Выбери счет.', success: '' })
      return
    }

    const confirmed = window.confirm(
      'Закрыть выбранный счет? Закрытие возможно только при нулевом балансе и без активного кредита.',
    )
    if (!confirmed) {
      return
    }

    setCloseAccountState({ loading: true, error: '', success: '' })
    setCloseResult(null)

    try {
      const data = await apiRequest<CloseAccountResponse>(
        `/api/accounts/${accountID}/close`,
        {
          method: 'POST',
          token,
          headers: { 'Idempotency-Key': createIdempotencyKey() },
        },
      )

      setCloseResult(data)
      applyClosedAccount(data)
      setCloseAccountState({ loading: false, error: '', success: 'Счет закрыт.' })
    } catch (error) {
      setCloseAccountState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to close account',
        success: '',
      })
    }
  }

  return (
    <section className="panel">
      <div className="panelHeader">
        <div>
          <h2>Счета пользователя</h2>
          <p>
            Все основные действия: создание, список, просмотр, deposit, withdraw, close и predict.
          </p>
        </div>

        <div className="actions">
          <button type="button" onClick={loadAccounts} disabled={accountsState.loading || !token}>
            {accountsState.loading ? 'Загружаю...' : 'Загрузить счета'}
          </button>
          <button
            className="secondary"
            type="button"
            onClick={createAccount}
            disabled={createAccountState.loading || !token}
          >
            {createAccountState.loading ? 'Создаю...' : 'Создать счет'}
          </button>
        </div>
      </div>

      <RequestMessage state={accountsState} />
      <RequestMessage state={createAccountState} />

      <div className="accountsLayout">
        <section className="subPanel">
          <div className="subPanelHeader">
            <h3>Мои счета</h3>
            <span>{accounts.length}</span>
          </div>

          {accounts.length === 0 && (
            <div className="empty">Список пуст. Нажми “Загрузить счета” или “Создать счет”.</div>
          )}

          {accounts.length > 0 && (
            <div className="accountList">
              {accounts.map((account) => (
                <button
                  key={account.id}
                  className={selectedAccountId === String(account.id) ? 'accountItem active' : 'accountItem'}
                  type="button"
                  onClick={() => selectAccount(account)}
                >
                  <span className="accountNumber">{account.account_number}</span>
                  <span className="accountMeta">
                    <span>{account.balance} {account.currency}</span>
                    <span className={getAccountBadgeClass(account)}>{getAccountStatusText(account)}</span>
                  </span>
                </button>
              ))}
            </div>
          )}
        </section>

        <section className="subPanel">
          <div className="subPanelHeader">
            <h3>Выбранный счет</h3>
            {selectedAccount && (
              <span className={getAccountBadgeClass(selectedAccount)}>
                {getAccountStatusText(selectedAccount)}
              </span>
            )}
          </div>

          {!selectedAccount && <div className="empty">Выбери счет из списка слева.</div>}

          {selectedAccount && (
            <>
              <div className="detailsGrid">
                <div><span>ID</span><strong>{selectedAccount.id}</strong></div>
                <div><span>Номер</span><strong>{selectedAccount.account_number}</strong></div>
                <div><span>Баланс</span><strong>{selectedAccount.balance} {selectedAccount.currency}</strong></div>
                <div><span>Создан</span><strong>{formatDate(selectedAccount.created_at)}</strong></div>
                <div><span>Blocked</span><strong>{selectedAccount.is_blocked ? 'yes' : 'no'}</strong></div>
                <div><span>Closed at</span><strong>{formatDate(selectedAccount.closed_at)}</strong></div>
              </div>

              <div className="actions topGap">
                <button className="secondary" type="button" onClick={loadAccountDetails} disabled={accountDetailsState.loading}>
                  {accountDetailsState.loading ? 'Обновляю...' : 'Обновить данные'}
                </button>
              </div>

              <RequestMessage state={accountDetailsState} />

              <div className="accountActionsGrid">
                <form className="actionBox" onSubmit={handleDeposit}>
                  <h4>Пополнение</h4>
                  <p>Запрос к <code>POST /accounts/{'{accountId}'}/deposit</code>.</p>
                  <label>
                    <span>Amount</span>
                    <input value={depositAmount} onChange={(event) => setDepositAmount(event.target.value)} disabled={isAccountClosed(selectedAccount)} />
                  </label>
                  <button type="submit" disabled={depositState.loading || isAccountClosed(selectedAccount)}>
                    {depositState.loading ? 'Пополняю...' : 'Пополнить'}
                  </button>
                  <RequestMessage state={depositState} />
                </form>

                <form className="actionBox" onSubmit={handleWithdraw}>
                  <h4>Списание с MFA</h4>
                  <p>Сначала запроси MFA-код, потом выполни withdraw.</p>
                  <label>
                    <span>Amount</span>
                    <input value={withdrawAmount} onChange={(event) => setWithdrawAmount(event.target.value)} disabled={isAccountClosed(selectedAccount)} />
                  </label>
                  <button className="secondary" type="button" onClick={requestWithdrawMFA} disabled={withdrawMfaState.loading || isAccountClosed(selectedAccount)}>
                    {withdrawMfaState.loading ? 'Отправляю...' : 'Запросить MFA'}
                  </button>
                  <RequestMessage state={withdrawMfaState} />
                  <label>
                    <span>MFA code</span>
                    <input value={withdrawMfaCode} onChange={(event) => setWithdrawMfaCode(event.target.value)} disabled={isAccountClosed(selectedAccount)} />
                  </label>
                  <button type="submit" disabled={withdrawState.loading || isAccountClosed(selectedAccount)}>
                    {withdrawState.loading ? 'Списываю...' : 'Списать'}
                  </button>
                  <RequestMessage state={withdrawState} />
                </form>

                <form className="actionBox" onSubmit={loadPrediction}>
                  <h4>Прогноз баланса</h4>
                  <p>Запрос к <code>GET /accounts/{'{accountId}'}/predict</code>.</p>
                  <label>
                    <span>Days</span>
                    <input value={predictDays} onChange={(event) => setPredictDays(event.target.value)} />
                  </label>
                  <button type="submit" disabled={predictState.loading}>
                    {predictState.loading ? 'Считаю...' : 'Получить прогноз'}
                  </button>
                  <RequestMessage state={predictState} />
                </form>

                <div className="actionBox dangerZone">
                  <h4>Закрытие счета</h4>
                  <p>Закрытие возможно только при нулевом балансе и без активного кредита.</p>
                  <button className="danger" type="button" onClick={closeAccount} disabled={closeAccountState.loading || isAccountClosed(selectedAccount)}>
                    {closeAccountState.loading ? 'Закрываю...' : 'Закрыть счет'}
                  </button>
                  <RequestMessage state={closeAccountState} />
                </div>
              </div>

              {predictResult && (
                <div className="result success">
                  <strong>Прогноз баланса</strong>
                  <pre>{JSON.stringify(predictResult, null, 2)}</pre>
                </div>
              )}

              {closeResult && (
                <div className="result success">
                  <strong>Результат закрытия счета</strong>
                  <pre>{JSON.stringify(closeResult, null, 2)}</pre>
                </div>
              )}
            </>
          )}
        </section>
      </div>
    </section>
  )
}
