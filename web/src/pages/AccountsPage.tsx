import { useEffect, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import type { FormEvent } from 'react'
import { queryKeys } from '../api/queryKeys'
import { OperationStatisticsPanel } from '../components/analytics/OperationStatisticsPanel'
import { RequestStatus } from '../components/RequestStatus'
import type { AccountResponse, CloseAccountResponse } from '../types/account'
import type { OperationStatisticsResponse } from '../types/analytics'
import { emptyState, type RequestState } from '../types/common'
import {
  getAccountBadgeClass,
  getAccountStatusText,
  isAccountClosed,
} from '../utils/format'
import { Button } from '../components/ui/Button'
import { AccountListPanel } from '../features/accounts/AccountListPanel'
import { Card } from '../components/ui/Card'
import { ConfirmDialog } from '../components/ui/ConfirmDialog'
import { useAccounts } from '../hooks/useAccounts'
import { useMfaFlow } from '../hooks/useMfaFlow'
import { useToast } from '../hooks/useToast'
import { validateAmount } from '../utils/validation'

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
  const { showToast } = useToast()
  const queryClient = useQueryClient()
  const accountsDomain = useAccounts(token)
  const withdrawMfaFlow = useMfaFlow(token)
  const [accountsState, setAccountsState] = useState<RequestState>(emptyState)
  const [createAccountState, setCreateAccountState] = useState<RequestState>(emptyState)
  const [accountDetailsState, setAccountDetailsState] = useState<RequestState>(emptyState)
  const [depositState, setDepositState] = useState<RequestState>(emptyState)
  const [withdrawMfaState, setWithdrawMfaState] = useState<RequestState>(emptyState)
  const [withdrawState, setWithdrawState] = useState<RequestState>(emptyState)
  const [accountStatisticsState, setAccountStatisticsState] = useState<RequestState>(emptyState)
  const [closeAccountState, setCloseAccountState] = useState<RequestState>(emptyState)

  const [accounts, setAccounts] = useState<AccountResponse[]>([])
  const [selectedAccountId, setSelectedAccountId] = useState(sharedAccountId)
  const [selectedAccount, setSelectedAccount] = useState<AccountResponse | null>(null)
  const [depositAmount, setDepositAmount] = useState('100.00')
  const [withdrawAmount, setWithdrawAmount] = useState('50.00')
  const [withdrawMfaCode, setWithdrawMfaCode] = useState('')
  const [accountStatisticsLimit, setAccountStatisticsLimit] = useState('100')
  const [accountOperationStatistics, setAccountOperationStatistics] = useState<OperationStatisticsResponse | null>(null)
  const [closeResult, setCloseResult] = useState<CloseAccountResponse | null>(null)
  const [closeConfirmOpen, setCloseConfirmOpen] = useState(false)

  useEffect(() => {
    if (sharedAccountId && !selectedAccountId) {
      setSelectedAccountId(sharedAccountId)
    }
  }, [sharedAccountId, selectedAccountId])

  useEffect(() => {
    const cachedAccounts = accountsDomain.listQuery.data
    if (!cachedAccounts) {
      return
    }

    setAccounts(cachedAccounts)

    if (cachedAccounts.length === 0) {
      setSelectedAccountId('')
      setSelectedAccount(null)
      return
    }

    const accountToSelect =
      cachedAccounts.find((item) => String(item.id) === selectedAccountId) ||
      cachedAccounts.find((item) => String(item.id) === sharedAccountId) ||
      (selectedAccount
        ? cachedAccounts.find((item) => item.id === selectedAccount.id)
        : undefined) ||
      cachedAccounts[0]

    setSelectedAccountId(String(accountToSelect.id))
    setSelectedAccount(accountToSelect)
    onSharedAccountIdChange(String(accountToSelect.id))
  }, [accountsDomain.listQuery.data])

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
    const statisticsLimit = Number(accountStatisticsLimit)
    const cachedStatistics = Number.isInteger(statisticsLimit)
      ? queryClient.getQueryData<OperationStatisticsResponse>(
        queryKeys.accounts.operationStatistics(account.id, statisticsLimit),
      )
      : undefined
    setAccountOperationStatistics(cachedStatistics || null)

    setAccountStatisticsState(emptyState)
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
      const result = await accountsDomain.listQuery.refetch()
      if (result.error) {
        throw result.error
      }
      const list = Array.isArray(result.data) ? result.data : []
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
      const account = await accountsDomain.createMutation.mutateAsync()

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
      const account = await accountsDomain.detailMutation.mutateAsync(accountID)
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

    const amountError = validateAmount(depositAmount)
    if (amountError) {
      setDepositState({ loading: false, error: amountError, success: '' })
      return
    }

    setDepositState({ loading: true, error: '', success: '' })

    try {
      const account = await accountsDomain.depositMutation.mutateAsync({ accountID, amount: depositAmount })

      upsertAccount(account)
      setAccountOperationStatistics(null)
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

    const amountError = validateAmount(withdrawAmount)
    if (amountError) {
      setWithdrawMfaState({ loading: false, error: amountError, success: '' })
      return
    }

    setWithdrawMfaState({ loading: true, error: '', success: '' })

    try {
      await withdrawMfaFlow.requestMutation.mutateAsync({
        purpose: 'withdraw',
        account_id: accountID,
        amount: withdrawAmount,
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
      const account = await accountsDomain.withdrawMutation.mutateAsync({
        accountID,
        amount: withdrawAmount,
        mfaCode: withdrawMfaCode,
      })

      upsertAccount(account)
      setAccountOperationStatistics(null)
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

  const loadAccountOperationStatistics = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!requireToken(setAccountStatisticsState)) {
      return
    }

    const accountID = selectedAccountIDNumber()
    if (!accountID) {
      setAccountStatisticsState({ loading: false, error: 'Выбери счет.', success: '' })
      return
    }

    const limit = Number(accountStatisticsLimit)
    if (!Number.isInteger(limit) || limit <= 0 || limit > 500) {
      setAccountStatisticsState({ loading: false, error: 'Limit должен быть от 1 до 500.', success: '' })
      return
    }

    setAccountStatisticsState({ loading: true, error: '', success: '' })
    setAccountOperationStatistics(null)

    try {
      const data = await accountsDomain.operationStatisticsMutation.mutateAsync({ accountID, limit })

      setAccountOperationStatistics(data)
      setAccountStatisticsState({ loading: false, error: '', success: 'Статистика операций по счету загружена.' })
    } catch (error) {
      setAccountStatisticsState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load account operation statistics',
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

    setCloseConfirmOpen(true)
  }

  const confirmCloseAccount = async () => {
    const accountID = selectedAccountIDNumber()
    if (!accountID) {
      setCloseConfirmOpen(false)
      setCloseAccountState({ loading: false, error: 'Выбери счет.', success: '' })
      return
    }

    setCloseAccountState({ loading: true, error: '', success: '' })
    setCloseResult(null)

    try {
      const data = await accountsDomain.closeMutation.mutateAsync(accountID)

      setCloseResult(data)
      setAccountOperationStatistics(null)
      applyClosedAccount(data)
      setCloseConfirmOpen(false)
      setCloseAccountState({ loading: false, error: '', success: 'Счет закрыт.' })
      showToast('Счет закрыт.', 'success')
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to close account'
      setCloseAccountState({
        loading: false,
        error: message,
        success: '',
      })
      showToast(message, 'error')
    }
  }

  return (
    <Card variant="plain" className="panel">
      <div className="panelHeader">
        <div>
          <h2>Счета пользователя</h2>
          <p>
            Все основные действия: создание, список, просмотр, deposit, withdraw, close и полная статистика операций.
          </p>
        </div>

        <div className="actions">
          <Button type="button" onClick={loadAccounts} disabled={accountsState.loading || !token}>
            {accountsState.loading ? 'Загружаю...' : 'Загрузить счета'}
          </Button>
          <Button
            className="secondary"
            type="button"
            onClick={createAccount}
            disabled={createAccountState.loading || !token}
          >
            {createAccountState.loading ? 'Создаю...' : 'Создать счет'}
          </Button>
        </div>
      </div>

      <RequestStatus state={accountsState} />
      <RequestStatus state={createAccountState} />

      <div className="accountsLayout">
        <AccountListPanel
          accounts={accounts}
          selectedAccountId={selectedAccountId}
          onSelect={selectAccount}
        />

        <section className="subPanel">
          <div className="subPanelHeader accountOperationsHeader">
            <div>
              <h3>Операции по счету</h3>
              {selectedAccount && (
                <p className="mutedText">
                  Выбран счет <code>{selectedAccount.account_number}</code>, ID {selectedAccount.id}.
                </p>
              )}
            </div>

            {selectedAccount && (
              <div className="accountOperationsHeaderActions">
                <span className={getAccountBadgeClass(selectedAccount)}>
                  {getAccountStatusText(selectedAccount)}
                </span>
                <Button
                  className="secondary compactButton"
                  type="button"
                  onClick={loadAccountDetails}
                  disabled={accountDetailsState.loading}
                >
                  {accountDetailsState.loading ? 'Обновляю...' : 'Обновить данные'}
                </Button>
              </div>
            )}
          </div>

          {!selectedAccount && <div className="empty">Выбери счет из списка слева.</div>}

          {selectedAccount && (
            <>
              <RequestStatus state={accountDetailsState} />

              <div className="accountActionsGrid">
                <form className="actionBox" onSubmit={handleDeposit}>
                  <h4>Пополнение</h4>
                  <p>Запрос к <code>POST /accounts/{'{accountId}'}/deposit</code>.</p>
                  <label>
                    <span>Amount</span>
                    <input value={depositAmount} onChange={(event) => setDepositAmount(event.target.value)} disabled={isAccountClosed(selectedAccount)} />
                  </label>
                  <Button type="submit" disabled={depositState.loading || isAccountClosed(selectedAccount)}>
                    {depositState.loading ? 'Пополняю...' : 'Пополнить'}
                  </Button>
                  <RequestStatus state={depositState} />
                </form>

                <form className="actionBox" onSubmit={handleWithdraw}>
                  <h4>Списание с MFA</h4>
                  <p>Сначала запроси MFA-код, потом выполни withdraw.</p>
                  <label>
                    <span>Amount</span>
                    <input value={withdrawAmount} onChange={(event) => setWithdrawAmount(event.target.value)} disabled={isAccountClosed(selectedAccount)} />
                  </label>
                  <Button className="secondary" type="button" onClick={requestWithdrawMFA} disabled={withdrawMfaState.loading || isAccountClosed(selectedAccount)}>
                    {withdrawMfaState.loading ? 'Отправляю...' : 'Запросить MFA'}
                  </Button>
                  <RequestStatus state={withdrawMfaState} />
                  <label>
                    <span>MFA code</span>
                    <input value={withdrawMfaCode} onChange={(event) => setWithdrawMfaCode(event.target.value)} disabled={isAccountClosed(selectedAccount)} />
                  </label>
                  <Button type="submit" disabled={withdrawState.loading || isAccountClosed(selectedAccount)}>
                    {withdrawState.loading ? 'Списываю...' : 'Списать'}
                  </Button>
                  <RequestStatus state={withdrawState} />
                </form>
              </div>

              <OperationStatisticsPanel
                title="Полная статистика операций по счету"
                description="История и суммы по выбранному счету."
                endpointLabel="GET /accounts/{accountId}/operations/statistics"
                limit={accountStatisticsLimit}
                state={accountStatisticsState}
                statistics={accountOperationStatistics}
                disabled={isAccountClosed(selectedAccount)}
                emptyText="Нажми “Получить статистику”, чтобы увидеть историю и суммы по выбранному счету."
                onLimitChange={setAccountStatisticsLimit}
                onSubmit={loadAccountOperationStatistics}
              />
            </>
          )}
        </section>

        <aside className="accountsSideColumn">
          {selectedAccount ? (
            <section className="subPanel accountClosePanel dangerZone">
              <div className="subPanelHeader">
                <div>
                  <h3>Закрытие счета</h3>
                  <p className="mutedText">Закрытие возможно только при нулевом балансе и без активного кредита.</p>
                </div>
                <span className={getAccountBadgeClass(selectedAccount)}>
                  {getAccountStatusText(selectedAccount)}
                </span>
              </div>

              <Button
                className="danger"
                type="button"
                onClick={closeAccount}
                disabled={closeAccountState.loading || isAccountClosed(selectedAccount)}
              >
                {closeAccountState.loading ? 'Закрываю...' : 'Закрыть счет'}
              </Button>
              <RequestStatus state={closeAccountState} />

              {closeResult && (
                <div className="result success compactResult">
                  <strong>Результат закрытия счета</strong>
                  <pre>{JSON.stringify(closeResult, null, 2)}</pre>
                </div>
              )}
            </section>
          ) : (
            <section className="subPanel accountClosePanel">
              <h3>Закрытие счета</h3>
              <div className="empty compactEmpty">Выбери счет, чтобы открыть действия справа.</div>
            </section>
          )}
        </aside>
      </div>
      <ConfirmDialog
        open={closeConfirmOpen}
        title="Закрыть счет"
        message="Закрыть выбранный счет? Закрытие возможно только при нулевом балансе и без активного кредита."
        confirmText="Закрыть счет"
        danger
        loading={closeAccountState.loading}
        onConfirm={() => void confirmCloseAccount()}
        onCancel={() => setCloseConfirmOpen(false)}
      />
    </Card>
  )
}
