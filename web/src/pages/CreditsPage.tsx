import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { accountsApi } from '../api/accountsApi'
import { creditsApi } from '../api/creditsApi'
import { mfaApi } from '../api/mfaApi'
import { RequestStatus } from '../components/RequestStatus'
import { Button } from '../components/ui/Button'
import { Card } from '../components/ui/Card'
import type { AccountResponse } from '../types/account'
import { emptyState, type RequestState } from '../types/common'
import type {
  CreditCheckResponse,
  CreditResponse,
  PaymentScheduleResponse,
} from '../types/credit'
import {
  formatDate,
  getAccountBadgeClass,
  getAccountStatusText,
} from '../utils/format'

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

export function CreditsPage({
  token,
  sharedAccountId,
  onSharedAccountIdChange,
}: CreditsPageProps) {
  const [accountsState, setAccountsState] = useState<RequestState>(emptyState)
  const [creditsState, setCreditsState] = useState<RequestState>(emptyState)
  const [checkCreditState, setCheckCreditState] = useState<RequestState>(emptyState)
  const [creditMfaState, setCreditMfaState] = useState<RequestState>(emptyState)
  const [createCreditState, setCreateCreditState] = useState<RequestState>(emptyState)
  const [creditDetailsState, setCreditDetailsState] = useState<RequestState>(emptyState)
  const [scheduleState, setScheduleState] = useState<RequestState>(emptyState)

  const [accounts, setAccounts] = useState<AccountResponse[]>([])
  const [selectedAccountId, setSelectedAccountId] = useState(sharedAccountId)
  const [selectedAccount, setSelectedAccount] = useState<AccountResponse | null>(null)

  const [principalAmount, setPrincipalAmount] = useState('100000.00')
  const [termMonths, setTermMonths] = useState('12')
  const [creditMfaCode, setCreditMfaCode] = useState('')

  const [creditCheck, setCreditCheck] = useState<CreditCheckResponse | null>(null)
  const [createdCredit, setCreatedCredit] = useState<CreditResponse | null>(null)
  const [credits, setCredits] = useState<CreditResponse[]>([])
  const [selectedCreditId, setSelectedCreditId] = useState('')
  const [selectedCredit, setSelectedCredit] = useState<CreditResponse | null>(null)
  const [schedule, setSchedule] = useState<PaymentScheduleResponse[]>([])

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

  const resetCreditSelection = () => {
    setCredits([])
    setSelectedCreditId('')
    setSelectedCredit(null)
    setSchedule([])
    setCreditCheck(null)
    setCreatedCredit(null)
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
    setSchedule([])
    setCreditDetailsState(emptyState)
    setScheduleState(emptyState)
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

    try {
      const list = await creditsApi.listByAccount(token, accountID)
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
    setCreditCheck(null)
    setCreatedCredit(null)
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
      const data = await accountsApi.list(token)
      const list = Array.isArray(data) ? data : []
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
      const data = await creditsApi.check(token, request)

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
      await mfaApi.request(token, {
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

    try {
      const data = await creditsApi.create(token, {
        ...request,
        mfa_code: creditMfaCode,
      })

      setCreatedCredit(data)
      upsertCredit(data)
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
      const data = await creditsApi.get(token, creditID)
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
      const data = await creditsApi.schedule(token, creditID)

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
        <section className="subPanel creditAccountPanelV3">
          <div className="subPanelHeader">
            <div>
              <h3>Счета</h3>
              <p className="mutedText">Выбери счет, чтобы загрузить его кредиты.</p>
            </div>
            <span>{accounts.length}</span>
          </div>

          {accounts.length === 0 && (
            <div className="empty compactEmpty">Нажми “Загрузить счета”.</div>
          )}

          {accounts.length > 0 && (
            <div className="creditAccountCardsList">
              {accounts.map((account) => (
                <Button
                  key={account.id}
                  className={
                    selectedAccountId === String(account.id)
                      ? 'creditAccountCard selected'
                      : 'creditAccountCard'
                  }
                  type="button"
                  onClick={() => selectAccount(account)}
                >
                  <span className="creditAccountNumber">{account.account_number}</span>
                  <span className="creditAccountMeta">
                    <span>ID {account.id}</span>
                    <span>{account.balance} {account.currency}</span>
                    <span className={getAccountBadgeClass(account)}>
                      {getAccountStatusText(account)}
                    </span>
                  </span>
                </Button>
              ))}
            </div>
          )}
        </section>

        <section className="subPanel creditListPanelV3">
          <div className="subPanelHeader">
            <div>
              <h3>Кредиты выбранного счета</h3>
              <p className="mutedText">
                {selectedAccount
                  ? `account_id ${selectedAccount.id}`
                  : 'Сначала выбери счет.'}
              </p>
            </div>
            <span>{credits.length}</span>
          </div>

          {!selectedAccount && (
            <div className="empty compactEmpty">Выбери счет слева.</div>
          )}

          {selectedAccount && credits.length === 0 && !creditsState.loading && (
            <div className="empty compactEmpty">У выбранного счета пока нет кредитов.</div>
          )}

          {credits.length > 0 && (
            <div className="creditCardsListV3">
              {credits.map((credit) => (
                <Button
                  key={credit.id}
                  className={
                    selectedCreditId === String(credit.id)
                      ? 'creditLoanCard selected'
                      : 'creditLoanCard'
                  }
                  type="button"
                  onClick={() => selectCredit(credit)}
                >
                  <span className="creditLoanMain">
                    <span className="creditLoanTitle">credit_id {credit.id}</span>
                    <span className={getCreditBadgeClass(credit)}>{credit.status}</span>
                  </span>
                  <span className="creditLoanMeta">
                    <span>{credit.principal_amount}</span>
                    <span>{credit.monthly_payment} / мес.</span>
                  </span>
                </Button>
              ))}
            </div>
          )}
        </section>
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
              <h3>Выбранный кредит</h3>
              <p className="mutedText">Детали кредита и график платежей.</p>
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
              <div className="creditSummaryGridV3">
                <div><span>ID</span><strong>{selectedCredit.id}</strong></div>
                <div><span>Account ID</span><strong>{selectedCredit.account_id}</strong></div>
                <div><span>Principal</span><strong>{selectedCredit.principal_amount}</strong></div>
                <div><span>Rate</span><strong>{selectedCredit.interest_rate}</strong></div>
                <div><span>Term</span><strong>{selectedCredit.term_months}</strong></div>
                <div><span>Monthly payment</span><strong>{selectedCredit.monthly_payment}</strong></div>
                <div><span>Status</span><strong>{selectedCredit.status}</strong></div>
                <div><span>Created</span><strong>{formatDate(selectedCredit.created_at)}</strong></div>
              </div>

              <div className="actions topGap">
                <Button
                  className="secondary"
                  type="button"
                  onClick={loadCreditDetails}
                  disabled={creditDetailsState.loading}
                >
                  {creditDetailsState.loading ? 'Обновляю...' : 'Обновить кредит'}
                </Button>
                <Button type="button" onClick={loadSchedule} disabled={scheduleState.loading}>
                  {scheduleState.loading ? 'Загружаю...' : 'График платежей'}
                </Button>
              </div>

              <RequestStatus state={creditDetailsState} />
              <RequestStatus state={scheduleState} />

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
