import { useEffect, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import type { FormEvent } from 'react'
import { RequestStatus } from '../components/RequestStatus'
import { Button } from '../components/ui/Button'
import { CreditAccountListPanel } from '../features/credits/CreditAccountListPanel'
import { CreditListPanel } from '../features/credits/CreditListPanel'
import { queryKeys } from '../api/queryKeys'
import { useAccounts } from '../hooks/useAccounts'
import { useCredits } from '../hooks/useCredits'
import { useMfaFlow } from '../hooks/useMfaFlow'
import { Card } from '../components/ui/Card'
import type { AccountResponse } from '../types/account'
import { emptyState, type RequestState } from '../types/common'
import type {
  CreditCheckResponse,
  CreditOperationResponse,
  CreditPrepaymentMode,
  CreditPrepaymentResponse,
  CreditResponse,
  PaymentScheduleResponse,
} from '../types/credit'
import { formatDate } from '../utils/format'
import { validateAmount } from '../utils/validation'

type CreditsPageProps = {
  token: string
  sharedAccountId: string
  onSharedAccountIdChange: (accountId: string) => void
}

function getCreditBadgeClass(credit: CreditResponse): string {
  if (credit.status === 'active') {
    return 'badge successBadge'
  }

  if (credit.status === 'overdue') {
    return 'badge dangerBadge'
  }

  return 'badge mutedBadge'
}

function getScheduleBadgeClass(payment: PaymentScheduleResponse): string {
  if (payment.status === 'paid') {
    return 'badge successBadge'
  }

  if (payment.status === 'overdue') {
    return 'badge dangerBadge'
  }

  return 'badge mutedBadge'
}

function getCreditOperationBadgeClass(operation: CreditOperationResponse): string {
  if (operation.status === 'completed' || operation.status === 'paid') {
    return 'badge successBadge'
  }

  if (operation.status === 'failed' || operation.status === 'overdue') {
    return 'badge dangerBadge'
  }

  return 'badge mutedBadge'
}

function moneyToCents(amount: string): number {
  const normalized = amount.trim()
  if (!/^\d+(\.\d{1,2})?$/.test(normalized)) {
    return 0
  }

  const [whole, fraction = ''] = normalized.split('.')
  return Number(whole) * 100 + Number(fraction.padEnd(2, '0'))
}

function centsToMoney(cents: number): string {
  return `${Math.floor(cents / 100)}.${String(cents % 100).padStart(2, '0')}`
}

function calculatePendingScheduleAmount(schedule: PaymentScheduleResponse[]): string {
  const totalCents = schedule
    .filter((payment) => payment.status === 'pending')
    .reduce((sum, payment) => sum + moneyToCents(payment.amount), 0)

  return centsToMoney(totalCents)
}

export function CreditsPage({
  token,
  sharedAccountId,
  onSharedAccountIdChange,
}: CreditsPageProps) {
  const selectedAccountID = Number(sharedAccountId)
  const accountsDomain = useAccounts(token)
  const queryClient = useQueryClient()
  const creditsDomain = useCredits(token, Number.isInteger(selectedAccountID) && selectedAccountID > 0 ? selectedAccountID : undefined)
  const creditMfaFlow = useMfaFlow(token)
  const creditPrepaymentMfaFlow = useMfaFlow(token)
  const [accountsState, setAccountsState] = useState<RequestState>(emptyState)
  const [creditsState, setCreditsState] = useState<RequestState>(emptyState)
  const [checkCreditState, setCheckCreditState] = useState<RequestState>(emptyState)
  const [creditMfaState, setCreditMfaState] = useState<RequestState>(emptyState)
  const [createCreditState, setCreateCreditState] = useState<RequestState>(emptyState)
  const [creditDetailsState, setCreditDetailsState] = useState<RequestState>(emptyState)
  const [scheduleState, setScheduleState] = useState<RequestState>(emptyState)
  const [operationHistoryState, setOperationHistoryState] = useState<RequestState>(emptyState)
  const [prepaymentMfaState, setPrepaymentMfaState] = useState<RequestState>(emptyState)
  const [prepaymentState, setPrepaymentState] = useState<RequestState>(emptyState)

  const [accounts, setAccounts] = useState<AccountResponse[]>([])
  const [selectedAccountId, setSelectedAccountId] = useState(sharedAccountId)
  const [selectedAccount, setSelectedAccount] = useState<AccountResponse | null>(null)

  const [principalAmount, setPrincipalAmount] = useState('100000.00')
  const [termMonths, setTermMonths] = useState('12')
  const [creditMfaCode, setCreditMfaCode] = useState('')
  const [prepaymentAmount, setPrepaymentAmount] = useState('10000.00')
  const [prepaymentMode, setPrepaymentMode] = useState<CreditPrepaymentMode>('reduce_term')
  const [prepaymentMfaCode, setPrepaymentMfaCode] = useState('')
  const [prepaymentAutofillState, setPrepaymentAutofillState] = useState<RequestState>(emptyState)

  const [creditCheck, setCreditCheck] = useState<CreditCheckResponse | null>(null)
  const [createdCredit, setCreatedCredit] = useState<CreditResponse | null>(null)
  const [prepaymentResult, setPrepaymentResult] = useState<CreditPrepaymentResponse | null>(null)
  const [credits, setCredits] = useState<CreditResponse[]>([])
  const [selectedCreditId, setSelectedCreditId] = useState('')
  const [selectedCredit, setSelectedCredit] = useState<CreditResponse | null>(null)
  const [schedule, setSchedule] = useState<PaymentScheduleResponse[]>([])
  const [creditOperations, setCreditOperations] = useState<CreditOperationResponse[]>([])

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

  useEffect(() => {
    const cachedCredits = creditsDomain.accountCreditsQuery.data
    if (!cachedCredits) {
      return
    }

    setCredits(cachedCredits)

    if (cachedCredits.length === 0) {
      setSelectedCreditId('')
      setSelectedCredit(null)
      setSchedule([])
      setCreditOperations([])
      return
    }

    const creditToSelect =
      cachedCredits.find((item) => String(item.id) === selectedCreditId) ||
      (selectedCredit
        ? cachedCredits.find((item) => item.id === selectedCredit.id)
        : undefined) ||
      cachedCredits[0]

    setSelectedCreditId(String(creditToSelect.id))
    setSelectedCredit(creditToSelect)

    const cachedSchedule = queryClient.getQueryData<PaymentScheduleResponse[]>(
      queryKeys.credits.schedule(creditToSelect.id),
    )
    setSchedule(cachedSchedule || [])

    const cachedOperations = queryClient.getQueryData<CreditOperationResponse[]>(
      queryKeys.credits.operations(creditToSelect.id),
    )
    setCreditOperations(cachedOperations || [])
  }, [creditsDomain.accountCreditsQuery.data])

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

  const resetCreditSelection = () => {
    setCredits([])
    setSelectedCreditId('')
    setSelectedCredit(null)
    setSchedule([])
    setCreditOperations([])
    setCreditCheck(null)
    setCreatedCredit(null)
    setPrepaymentResult(null)
  }

  const getCreditRequest = (setState: (state: RequestState) => void) => {
    const accountID = Number(selectedAccountId)
    const months = Number(termMonths)

    if (!Number.isInteger(accountID) || accountID <= 0) {
      setState({
        loading: false,
        error: 'Выбери счет или укажи корректный account_id.',
        success: '',
      })
      return null
    }

    if (!Number.isInteger(months) || months <= 0) {
      setState({
        loading: false,
        error: 'Укажи корректный term_months.',
        success: '',
      })
      return null
    }

    return {
      account_id: accountID,
      principal_amount: principalAmount,
      term_months: months,
    }
  }

  const selectCredit = (credit: CreditResponse) => {
    setSelectedCreditId(String(credit.id))
    setSelectedCredit(credit)

    const cachedSchedule = queryClient.getQueryData<PaymentScheduleResponse[]>(
      queryKeys.credits.schedule(credit.id),
    )
    setSchedule(cachedSchedule || [])

    const cachedOperations = queryClient.getQueryData<CreditOperationResponse[]>(
      queryKeys.credits.operations(credit.id),
    )
    setCreditOperations(cachedOperations || [])

    setCreditDetailsState(emptyState)
    setScheduleState(emptyState)
    setOperationHistoryState(emptyState)
    setPrepaymentMfaState(emptyState)
    setPrepaymentState(emptyState)
    setPrepaymentAutofillState(emptyState)
    setPrepaymentResult(null)
    setPrepaymentMfaCode('')
  }

  const loadCreditsForAccount = async (accountID: number) => {
    if (!requireToken(setCreditsState)) {
      return
    }

    setCreditsState({ loading: true, error: '', success: '' })
    setCredits([])
    setSelectedCreditId('')
    setSelectedCredit(null)
    setSchedule([])
    setCreditOperations([])

    try {
      const list = await creditsDomain.listByAccountMutation.mutateAsync(accountID)
      setCredits(list)

      if (list.length > 0) {
        selectCredit(list[0])
      }

      setCreditsState({
        loading: false,
        error: '',
        success: list.length > 0
          ? 'Кредиты выбранного счета загружены.'
          : 'У выбранного счета пока нет кредитов.',
      })
    } catch (error) {
      setCreditsState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load account credits',
        success: '',
      })
    }
  }

  const selectAccount = (account: AccountResponse) => {
    setSelectedAccountId(String(account.id))
    setSelectedAccount(account)
    onSharedAccountIdChange(String(account.id))
    setAccountsState(emptyState)
    setCheckCreditState(emptyState)
    setCreditMfaState(emptyState)
    setCreateCreditState(emptyState)
    setPrepaymentMfaState(emptyState)
    setPrepaymentState(emptyState)
    setCreditCheck(null)
    setCreatedCredit(null)
    setPrepaymentResult(null)
    void loadCreditsForAccount(account.id)
  }

  const upsertCredit = (credit: CreditResponse) => {
    setCredits((current) => {
      const exists = current.some((item) => item.id === credit.id)
      return exists
        ? current.map((item) => (item.id === credit.id ? credit : item))
        : [credit, ...current]
    })
    selectCredit(credit)
  }

  const loadAccounts = async () => {
    if (!requireToken(setAccountsState)) {
      return
    }

    setAccountsState({ loading: true, error: '', success: '' })
    resetCreditSelection()

    try {
      const result = await accountsDomain.listQuery.refetch()
      if (result.error) {
        throw result.error
      }
      const list = Array.isArray(result.data) ? result.data : []
      setAccounts(list)

      setAccountsState({ loading: false, error: '', success: 'Список счетов загружен.' })

      if (list.length === 0) {
        setSelectedAccountId('')
        setSelectedAccount(null)
        return
      }

      const accountToSelect =
        list.find((item) => String(item.id) === selectedAccountId) || list[0]
      selectAccount(accountToSelect)
    } catch (error) {
      setAccountsState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load accounts',
        success: '',
      })
    }
  }

  const refreshAccountsAfterCreditBalanceChange = async (accountID: number) => {
    try {
      const result = await accountsDomain.listQuery.refetch()
      const list = Array.isArray(result.data) ? result.data : []

      if (list.length === 0) {
        return
      }

      setAccounts(list)

      const updatedAccount = list.find((account) => account.id === accountID)
      if (updatedAccount) {
        setSelectedAccount(updatedAccount)
        setSelectedAccountId(String(updatedAccount.id))
        onSharedAccountIdChange(String(updatedAccount.id))
      }
    } catch {
      // Credit operation has already completed successfully. Balance refresh can be retried
      // by pressing "Загрузить счета", so we do not replace the operation success message here.
    }
  }

  const checkCredit = async (event?: FormEvent<HTMLFormElement>) => {
    event?.preventDefault()

    if (!requireToken(setCheckCreditState)) {
      return
    }

    const request = getCreditRequest(setCheckCreditState)
    if (!request) {
      return
    }

    setCheckCreditState({ loading: true, error: '', success: '' })
    setCreditCheck(null)

    try {
      const data = await creditsDomain.checkMutation.mutateAsync(request)

      setCreditCheck(data)
      setCheckCreditState({
        loading: false,
        error: '',
        success: data.eligible
          ? 'Проверка пройдена. Можно запросить MFA и оформить кредит.'
          : 'Проверка выполнена. Кредит не одобрен по политике.',
      })
    } catch (error) {
      setCheckCreditState({
        loading: false,
        error: error instanceof Error ? error.message : 'Credit check failed',
        success: '',
      })
    }
  }

  const requestCreditMFA = async () => {
    if (!requireToken(setCreditMfaState)) {
      return
    }

    const request = getCreditRequest(setCreditMfaState)
    if (!request) {
      return
    }

    setCreditMfaState({ loading: true, error: '', success: '' })

    try {
      await creditMfaFlow.requestMutation.mutateAsync({
        purpose: 'credit_create',
        account_id: request.account_id,
        principal_amount: request.principal_amount,
        term_months: request.term_months,
      })

      setCreditMfaState({
        loading: false,
        error: '',
        success: 'MFA-код для оформления кредита отправлен.',
      })
    } catch (error) {
      setCreditMfaState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to request MFA code',
        success: '',
      })
    }
  }

  const createCredit = async () => {
    if (!requireToken(setCreateCreditState)) {
      return
    }

    const request = getCreditRequest(setCreateCreditState)
    if (!request) {
      return
    }

    setCreateCreditState({ loading: true, error: '', success: '' })
    setCreatedCredit(null)
    setPrepaymentResult(null)

    try {
      const data = await creditsDomain.createMutation.mutateAsync({
        ...request,
        mfa_code: creditMfaCode,
      })

      setCreatedCredit(data)
      upsertCredit(data)
      await refreshAccountsAfterCreditBalanceChange(data.account_id)
      setCreditMfaCode('')
      setCreateCreditState({ loading: false, error: '', success: 'Кредит оформлен.' })
    } catch (error) {
      setCreateCreditState({
        loading: false,
        error: error instanceof Error ? error.message : 'Create credit failed',
        success: '',
      })
    }
  }

  const loadCreditDetails = async () => {
    if (!requireToken(setCreditDetailsState)) {
      return
    }

    const creditID = Number(selectedCreditId)
    if (!Number.isInteger(creditID) || creditID <= 0) {
      setCreditDetailsState({ loading: false, error: 'Выбери кредит.', success: '' })
      return
    }

    setCreditDetailsState({ loading: true, error: '', success: '' })

    try {
      const data = await creditsDomain.detailMutation.mutateAsync(creditID)
      upsertCredit(data)
      setCreditDetailsState({ loading: false, error: '', success: 'Данные кредита обновлены.' })
    } catch (error) {
      setCreditDetailsState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load credit',
        success: '',
      })
    }
  }

  const loadSchedule = async () => {
    if (!requireToken(setScheduleState)) {
      return
    }

    const creditID = Number(selectedCreditId)
    if (!Number.isInteger(creditID) || creditID <= 0) {
      setScheduleState({ loading: false, error: 'Выбери кредит.', success: '' })
      return
    }

    setScheduleState({ loading: true, error: '', success: '' })
    setSchedule([])

    try {
      const data = await creditsDomain.scheduleMutation.mutateAsync(creditID)

      setSchedule(Array.isArray(data) ? data : [])
      setScheduleState({ loading: false, error: '', success: 'График платежей загружен.' })
    } catch (error) {
      setScheduleState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load schedule',
        success: '',
      })
    }
  }


  const loadCreditOperations = async () => {
    if (!requireToken(setOperationHistoryState)) {
      return
    }

    const creditID = Number(selectedCreditId)
    if (!Number.isInteger(creditID) || creditID <= 0) {
      setOperationHistoryState({ loading: false, error: 'Выбери кредит.', success: '' })
      return
    }

    setOperationHistoryState({ loading: true, error: '', success: '' })
    setCreditOperations([])

    try {
      const data = await creditsDomain.operationsMutation.mutateAsync(creditID)

      setCreditOperations(Array.isArray(data) ? data : [])
      setOperationHistoryState({
        loading: false,
        error: '',
        success: 'История операций по кредиту загружена.',
      })
    } catch (error) {
      setOperationHistoryState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load credit operation history',
        success: '',
      })
    }
  }


  const getSelectedCreditID = (setState: (state: RequestState) => void): number | null => {
    const creditID = Number(selectedCreditId)
    if (!Number.isInteger(creditID) || creditID <= 0) {
      setState({ loading: false, error: 'Выбери кредит.', success: '' })
      return null
    }

    return creditID
  }


  const loadFullCloseAmount = async () => {
    const creditID = getSelectedCreditID(setPrepaymentAutofillState)
    if (!creditID) {
      return
    }

    setPrepaymentAutofillState({ loading: true, error: '', success: '' })

    try {
      const data = await creditsDomain.scheduleMutation.mutateAsync(creditID)
      const nextSchedule = Array.isArray(data) ? data : []
      setSchedule(nextSchedule)

      const amount = calculatePendingScheduleAmount(nextSchedule)
      if (moneyToCents(amount) <= 0) {
        setPrepaymentAutofillState({
          loading: false,
          error: 'Не удалось рассчитать остаток по будущим платежам.',
          success: '',
        })
        return
      }

      setPrepaymentAmount(amount)
      setPrepaymentAutofillState({
        loading: false,
        error: '',
        success: 'Сумма полного погашения подставлена из актуального графика.',
      })
    } catch (error) {
      setPrepaymentAutofillState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load credit schedule',
        success: '',
      })
    }
  }

  const changePrepaymentMode = (mode: CreditPrepaymentMode) => {
    setPrepaymentMode(mode)
    setPrepaymentMfaCode('')
    setPrepaymentMfaState(emptyState)
    setPrepaymentState(emptyState)
    setPrepaymentResult(null)

    if (mode === 'full_close') {
      void loadFullCloseAmount()
    }
  }
  const getPrepaymentRequest = (setState: (state: RequestState) => void) => {
    const creditID = getSelectedCreditID(setState)
    if (!creditID) {
      return null
    }

    const amountError = validateAmount(prepaymentAmount)
    if (amountError) {
      setState({ loading: false, error: amountError, success: '' })
      return null
    }

    return {
      creditID,
      amount: prepaymentAmount,
      mode: prepaymentMode,
    }
  }

  const requestPrepaymentMFA = async () => {
    if (!requireToken(setPrepaymentMfaState)) {
      return
    }

    const request = getPrepaymentRequest(setPrepaymentMfaState)
    if (!request) {
      return
    }

    setPrepaymentMfaState({ loading: true, error: '', success: '' })

    try {
      await creditPrepaymentMfaFlow.requestMutation.mutateAsync({
        purpose: 'credit_prepayment',
        credit_id: request.creditID,
        amount: request.amount,
        prepayment_mode: request.mode,
        mode: request.mode,
      })

      setPrepaymentMfaState({
        loading: false,
        error: '',
        success: 'MFA-код для досрочного погашения отправлен.',
      })
    } catch (error) {
      setPrepaymentMfaState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to request MFA code',
        success: '',
      })
    }
  }

  const prepayCredit = async () => {
    if (!requireToken(setPrepaymentState)) {
      return
    }

    const request = getPrepaymentRequest(setPrepaymentState)
    if (!request) {
      return
    }

    setPrepaymentState({ loading: true, error: '', success: '' })
    setPrepaymentResult(null)

    try {
      const data = await creditsDomain.prepayMutation.mutateAsync({
        creditID: request.creditID,
        body: {
          amount: request.amount,
          mode: request.mode,
          mfa_code: prepaymentMfaCode,
        },
      })

      setPrepaymentResult(data)
      upsertCredit(data.credit)
      await refreshAccountsAfterCreditBalanceChange(data.credit.account_id)
      setSchedule([])
      setCreditOperations([])
      setPrepaymentMfaCode('')
      setPrepaymentState({ loading: false, error: '', success: 'Досрочное погашение выполнено.' })
    } catch (error) {
      setPrepaymentState({
        loading: false,
        error: error instanceof Error ? error.message : 'Credit prepayment failed',
        success: '',
      })
    }
  }

  return (
    <Card variant="plain" className="panel creditsPage cardsLikeCreditsPage">
      <div className="panelHeader creditsHeader">
        <div>
          <h2>Кредиты</h2>
          <p>
            Сначала загрузи счета. При выборе счета frontend подгружает и показывает кредиты только этого счета.
          </p>
        </div>

        <div className="actions">
          <Button
            type="button"
            onClick={loadAccounts}
            disabled={accountsState.loading || !token}
          >
            {accountsState.loading ? 'Загружаю...' : 'Загрузить счета'}
          </Button>
        </div>
      </div>

      <RequestStatus state={accountsState} />
      <RequestStatus state={creditsState} />

      <div className="creditCardsTopGrid">
        <CreditAccountListPanel
          accounts={accounts}
          selectedAccountId={selectedAccountId}
          onSelect={selectAccount}
        />

        <CreditListPanel
          selectedAccount={selectedAccount}
          credits={credits}
          selectedCreditId={selectedCreditId}
          loading={creditsState.loading}
          getCreditBadgeClass={getCreditBadgeClass}
          onSelect={selectCredit}
        />
      </div>

      <div className="creditWorkGridV3">
        <section className="subPanel creditApplicationPanelV3">
          <div className="subPanelHeader">
            <div>
              <h3>Проверка и оформление</h3>
              <p className="mutedText">Выбранный счет уже подставлен в Account ID.</p>
            </div>
          </div>

          <form className="creditApplicationFormV3" onSubmit={checkCredit}>
            <label>
              <span>Account ID</span>
              <input
                value={selectedAccountId}
                onChange={(event) => {
                  setSelectedAccountId(event.target.value)
                  setSelectedAccount(null)
                  setCredits([])
                  setSelectedCredit(null)
                  setSelectedCreditId('')
                }}
                placeholder="ID счета"
              />
            </label>

            <label>
              <span>Principal amount</span>
              <input
                value={principalAmount}
                onChange={(event) => setPrincipalAmount(event.target.value)}
                placeholder="100000.00"
              />
            </label>

            <label>
              <span>Term months</span>
              <input
                value={termMonths}
                onChange={(event) => setTermMonths(event.target.value)}
                placeholder="12"
              />
            </label>

            <Button type="submit" disabled={checkCreditState.loading || !token}>
              {checkCreditState.loading ? 'Проверяю...' : 'Проверить кредит'}
            </Button>
          </form>

          <RequestStatus state={checkCreditState} />

          {creditCheck && (
            <div className={creditCheck.eligible ? 'creditDecision successDecision' : 'creditDecision errorDecision'}>
              <div className="creditDecisionHeader">
                <strong>{creditCheck.eligible ? 'Кредит может быть одобрен' : 'Кредит не одобрен'}</strong>
                <span>{creditCheck.monthly_payment} / мес.</span>
              </div>

              {creditCheck.reason && <p>{creditCheck.reason}</p>}

              {creditCheck.reasons && creditCheck.reasons.length > 0 && (
                <ul>
                  {creditCheck.reasons.map((reason) => (
                    <li key={reason}>{reason}</li>
                  ))}
                </ul>
              )}

              <div className="creditCheckGrid compactCreditCheck">
                <div><span>Rate</span><strong>{creditCheck.interest_rate}</strong></div>
                <div><span>Debt load</span><strong>{creditCheck.debt_load_percent}%</strong></div>
                <div><span>Limit</span><strong>{creditCheck.max_debt_load_percent}%</strong></div>
                <div><span>Income</span><strong>{creditCheck.monthly_income}</strong></div>
              </div>
            </div>
          )}

          <div className="creditMfaRowV3">
            <Button
              className="secondary"
              type="button"
              onClick={requestCreditMFA}
              disabled={creditMfaState.loading || !token}
            >
              {creditMfaState.loading ? 'Отправляю...' : 'Запросить MFA'}
            </Button>

            <label>
              <span>MFA code</span>
              <input
                value={creditMfaCode}
                onChange={(event) => setCreditMfaCode(event.target.value)}
                placeholder="6 цифр"
              />
            </label>

            <Button
              type="button"
              onClick={createCredit}
              disabled={createCreditState.loading || !token}
            >
              {createCreditState.loading ? 'Оформляю...' : 'Оформить кредит'}
            </Button>
          </div>

          <RequestStatus state={creditMfaState} />
          <RequestStatus state={createCreditState} />

          {createdCredit && (
            <div className="result success compactResult">
              <strong>Кредит оформлен</strong>
              <pre>{JSON.stringify(createdCredit, null, 2)}</pre>
            </div>
          )}
        </section>

        <section className="subPanel creditDetailsPanelV3">
          <div className="subPanelHeader">
            <div>
              <h3>График платежей</h3>
              {selectedCredit ? (
                <p className="mutedText">Выбран credit_id <code>{selectedCredit.id}</code>. Детали кредита показаны в карточке выше.</p>
              ) : (
                <p className="mutedText">Выбери кредит из списка выше.</p>
              )}
            </div>
            {selectedCredit && (
              <span className={getCreditBadgeClass(selectedCredit)}>{selectedCredit.status}</span>
            )}
          </div>

          {!selectedCredit && (
            <div className="empty">Выбери кредит из списка “Кредиты выбранного счета”.</div>
          )}

          {selectedCredit && (
            <>
              <div className="actions creditScheduleActions">
                <Button
                  className="secondary"
                  type="button"
                  onClick={loadCreditDetails}
                  disabled={creditDetailsState.loading}
                >
                  {creditDetailsState.loading ? 'Обновляю...' : 'Обновить кредит'}
                </Button>
                <Button type="button" onClick={loadSchedule} disabled={scheduleState.loading}>
                  {scheduleState.loading ? 'Загружаю...' : 'Показать график'}
                </Button>
                <Button
                  className="secondary"
                  type="button"
                  onClick={loadCreditOperations}
                  disabled={operationHistoryState.loading}
                >
                  {operationHistoryState.loading ? 'Загружаю...' : 'Показать историю операций'}
                </Button>
              </div>

              <RequestStatus state={creditDetailsState} />
              <RequestStatus state={scheduleState} />
              <RequestStatus state={operationHistoryState} />

              <section className="creditPrepaymentBox">
                <div className="subPanelHeader">
                  <div>
                    <h4>Досрочное погашение</h4>
                    <p className="mutedText">
                      Уменьшить срок, уменьшить будущий платеж или полностью закрыть кредит.
                    </p>
                  </div>
                </div>

                <div className="creditPrepaymentForm">
                  <label>
                    <span>Amount</span>
                    <input
                      value={prepaymentAmount}
                      onChange={(event) => setPrepaymentAmount(event.target.value)}
                      placeholder="10000.00"
                      disabled={selectedCredit.status !== 'active' || prepaymentAutofillState.loading}
                    />
                  </label>

                  <label>
                    <span>Mode</span>
                    <select
                      value={prepaymentMode}
                      onChange={(event) => changePrepaymentMode(event.target.value as CreditPrepaymentMode)}
                      disabled={selectedCredit.status !== 'active'}
                    >
                      <option value="reduce_term">Уменьшить срок</option>
                      <option value="reduce_payment">Уменьшить платеж</option>
                      <option value="full_close">Погасить полностью</option>
                    </select>
                  </label>

                  <Button
                    className="secondary"
                    type="button"
                    onClick={requestPrepaymentMFA}
                    disabled={prepaymentMfaState.loading || selectedCredit.status !== 'active'}
                  >
                    {prepaymentMfaState.loading ? 'Отправляю...' : 'Запросить MFA'}
                  </Button>

                  <label>
                    <span>MFA code</span>
                    <input
                      value={prepaymentMfaCode}
                      onChange={(event) => setPrepaymentMfaCode(event.target.value)}
                      placeholder="6 цифр"
                      disabled={selectedCredit.status !== 'active'}
                    />
                  </label>

                  <Button
                    type="button"
                    onClick={prepayCredit}
                    disabled={prepaymentState.loading || selectedCredit.status !== 'active'}
                  >
                    {prepaymentState.loading ? 'Погашаю...' : 'Погасить досрочно'}
                  </Button>
                </div>

                <RequestStatus state={prepaymentAutofillState} />
                <RequestStatus state={prepaymentMfaState} />
                <RequestStatus state={prepaymentState} />

                {prepaymentResult && (
                  <div className="result success compactResult">
                    <strong>Досрочное погашение выполнено</strong>
                    <pre>{JSON.stringify(prepaymentResult, null, 2)}</pre>
                  </div>
                )}
              </section>

              {schedule.length === 0 && !scheduleState.error && (
                <div className="empty compactEmpty">Нажми “Показать график”, чтобы загрузить платежи выбранного кредита.</div>
              )}

              {creditOperations.length > 0 && (
                <div className="tableWrap creditOperationHistoryTable topGap">
                  <table>
                    <thead>
                      <tr>
                        <th>Source</th>
                        <th>Event</th>
                        <th>Amount</th>
                        <th>Penalty</th>
                        <th>Status</th>
                        <th>Transaction</th>
                        <th>Schedule</th>
                        <th>Payment date</th>
                        <th>Occurred at</th>
                        <th>Description</th>
                      </tr>
                    </thead>
                    <tbody>
                      {creditOperations.map((operation, index) => (
                        <tr key={`${operation.source}-${operation.event_type}-${operation.transaction_id || operation.schedule_id || index}`}>
                          <td>{operation.source}</td>
                          <td>{operation.event_type}</td>
                          <td>{operation.amount || '-'}</td>
                          <td>{operation.penalty_amount || '-'}</td>
                          <td>
                            {operation.status ? (
                              <span className={getCreditOperationBadgeClass(operation)}>{operation.status}</span>
                            ) : '-'}
                          </td>
                          <td>{operation.transaction_id || '-'}</td>
                          <td>{operation.schedule_id || '-'}</td>
                          <td>{formatDate(operation.payment_date)}</td>
                          <td>{formatDate(operation.occurred_at)}</td>
                          <td>{operation.description || '-'}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}

              {schedule.length > 0 && (
                <div className="tableWrap scheduleTable topGap">
                  <table>
                    <thead>
                      <tr>
                        <th>ID</th>
                        <th>Payment date</th>
                        <th>Amount</th>
                        <th>Penalty</th>
                        <th>Status</th>
                        <th>Paid at</th>
                      </tr>
                    </thead>
                    <tbody>
                      {schedule.map((payment) => (
                        <tr key={payment.id}>
                          <td>{payment.id}</td>
                          <td>{formatDate(payment.payment_date)}</td>
                          <td>{payment.amount}</td>
                          <td>{payment.penalty_amount}</td>
                          <td><span className={getScheduleBadgeClass(payment)}>{payment.status}</span></td>
                          <td>{formatDate(payment.paid_at)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </>
          )}
        </section>
      </div>
    </Card>
  )
}
