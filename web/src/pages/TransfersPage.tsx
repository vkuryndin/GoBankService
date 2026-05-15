import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { RequestStatus } from '../components/RequestStatus'
import type { AccountResponse } from '../types/account'
import { emptyState, type RequestState } from '../types/common'
import type { TransferResponse } from '../types/transfer'
import {
  getAccountBadgeClass,
  getAccountStatusText,
} from '../utils/format'
import { Button } from '../components/ui/Button'
import { useAccounts } from '../hooks/useAccounts'
import { useMfaFlow } from '../hooks/useMfaFlow'
import { useTransfers } from '../hooks/useTransfers'
import { Card } from '../components/ui/Card'

type TransfersPageProps = {
  token: string
  sharedAccountId: string
  onSharedAccountIdChange: (accountId: string) => void
}

export function TransfersPage({
  token,
  sharedAccountId,
  onSharedAccountIdChange,
}: TransfersPageProps) {
  const accountsDomain = useAccounts(token)
  const transfersDomain = useTransfers(token)
  const transferMfaFlow = useMfaFlow(token)
  const [accountsState, setAccountsState] = useState<RequestState>(emptyState)
  const [transferMfaState, setTransferMfaState] = useState<RequestState>(emptyState)
  const [transferState, setTransferState] = useState<RequestState>(emptyState)

  const [accounts, setAccounts] = useState<AccountResponse[]>([])
  const [transferFromAccountId, setTransferFromAccountId] = useState(sharedAccountId)
  const [transferToAccountNumber, setTransferToAccountNumber] = useState('')
  const [transferAmount, setTransferAmount] = useState('100.00')
  const [transferDescription, setTransferDescription] = useState('Account transfer')
  const [transferMfaCode, setTransferMfaCode] = useState('')
  const [transferResult, setTransferResult] = useState<TransferResponse | null>(null)

  useEffect(() => {
    if (sharedAccountId && !transferFromAccountId) {
      setTransferFromAccountId(sharedAccountId)
    }
  }, [sharedAccountId, transferFromAccountId])

  useEffect(() => {
    const cachedAccounts = accountsDomain.listQuery.data
    if (!cachedAccounts) {
      return
    }

    setAccounts(cachedAccounts)

    if (!transferFromAccountId && cachedAccounts.length > 0) {
      const accountToSelect =
        cachedAccounts.find((item) => String(item.id) === sharedAccountId) ||
        cachedAccounts[0]

      setTransferFromAccountId(String(accountToSelect.id))
    }
  }, [accountsDomain.listQuery.data])

  const requireToken = (setState: (state: RequestState) => void): boolean => {
    if (token) {
      return true
    }

    setState({ loading: false, error: 'Сначала нужно войти в систему.', success: '' })
    return false
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
      setAccounts(Array.isArray(result.data) ? result.data : [])
      setAccountsState({ loading: false, error: '', success: 'Список счетов загружен.' })
    } catch (error) {
      setAccountsState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load accounts',
        success: '',
      })
    }
  }

  const selectFromAccount = (account: AccountResponse) => {
    setTransferFromAccountId(String(account.id))
    onSharedAccountIdChange(String(account.id))
  }

  const validateTransfer = (setState: (state: RequestState) => void) => {
    const fromAccountID = Number(transferFromAccountId)
    const toAccountNumber = transferToAccountNumber.trim().replace(/\s+/g, '')

    if (!Number.isInteger(fromAccountID) || fromAccountID <= 0) {
      setState({ loading: false, error: 'Выбери счет отправителя.', success: '' })
      return null
    }

    if (toAccountNumber === '') {
      setState({ loading: false, error: 'Укажи номер счета получателя.', success: '' })
      return null
    }

    const fromAccount = accounts.find((account) => account.id === fromAccountID)
    if (fromAccount && fromAccount.account_number === toAccountNumber) {
      setState({ loading: false, error: 'Нельзя переводить на тот же счет.', success: '' })
      return null
    }

    return { fromAccountID, toAccountNumber }
  }

  const requestTransferMFA = async () => {
    if (!requireToken(setTransferMfaState)) {
      return
    }

    const ids = validateTransfer(setTransferMfaState)
    if (!ids) {
      return
    }

    setTransferMfaState({ loading: true, error: '', success: '' })

    try {
      await transferMfaFlow.requestMutation.mutateAsync({
        purpose: 'transfer',
        from_account_id: ids.fromAccountID,
        to_account_number: ids.toAccountNumber,
        amount: transferAmount,
      })

      setTransferMfaState({ loading: false, error: '', success: 'MFA-код для перевода отправлен.' })
    } catch (error) {
      setTransferMfaState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to request MFA code',
        success: '',
      })
    }
  }

  const handleTransfer = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!requireToken(setTransferState)) {
      return
    }

    const ids = validateTransfer(setTransferState)
    if (!ids) {
      return
    }

    setTransferState({ loading: true, error: '', success: '' })
    setTransferResult(null)

    try {
      const data = await transfersDomain.transferMutation.mutateAsync({
        from_account_id: ids.fromAccountID,
        to_account_number: ids.toAccountNumber,
        amount: transferAmount,
        description: transferDescription,
        mfa_code: transferMfaCode,
      })

      setTransferResult(data)
      setTransferMfaCode('')
      setTransferState({ loading: false, error: '', success: 'Перевод выполнен.' })
    } catch (error) {
      setTransferState({
        loading: false,
        error: error instanceof Error ? error.message : 'Transfer failed',
        success: '',
      })
    }
  }

  return (
    <Card variant="plain" className="panel">
      <div className="panelHeader">
        <div>
          <h2>Переводы по номеру счета</h2>
          <p>
            Перевод выполняется по номеру счета получателя: сначала MFA-код, потом операция <code>POST /transfer</code>.
          </p>
        </div>

        <div className="actions">
          <Button className="secondary" type="button" onClick={loadAccounts} disabled={accountsState.loading || !token}>
            {accountsState.loading ? 'Загружаю...' : 'Загрузить мои счета'}
          </Button>
        </div>
      </div>

      <RequestStatus state={accountsState} />

      <div className="transfersLayout">
        <section className="subPanel">
          <div className="subPanelHeader">
            <h3>Мои счета</h3>
            <span>{accounts.length}</span>
          </div>

          {accounts.length === 0 && (
            <div className="empty">Нажми “Загрузить мои счета”, чтобы быстро выбрать счет отправителя.</div>
          )}

          {accounts.length > 0 && (
            <div className="accountList">
              {accounts.map((account) => (
                <Button
                  key={account.id}
                  className={transferFromAccountId === String(account.id) ? 'accountItem active' : 'accountItem'}
                  type="button"
                  onClick={() => selectFromAccount(account)}
                >
                  <span className="accountItemTop">
                    <span className="accountLabel">Счет</span>
                    <span className={getAccountBadgeClass(account)}>{getAccountStatusText(account)}</span>
                  </span>
                  <span className="accountNumber">{account.account_number}</span>
                  <span className="accountMetaGrid">
                    <span>
                      <small>ID</small>
                      <strong>{account.id}</strong>
                    </span>
                    <span>
                      <small>Баланс</small>
                      <strong>{account.balance} {account.currency}</strong>
                    </span>
                    <span>
                      <small>Blocked</small>
                      <strong>{account.is_blocked ? 'yes' : 'no'}</strong>
                    </span>
                    <span>
                      <small>Closed</small>
                      <strong>{account.closed_at ? 'yes' : 'no'}</strong>
                    </span>
                  </span>
                </Button>
              ))}
            </div>
          )}
        </section>

        <section className="subPanel transferPanel">
          <div className="subPanelHeader"><h3>Новый перевод</h3></div>

          <form className="form transferForm" onSubmit={handleTransfer}>
            <label>
              <span>Счет отправителя</span>
              <select value={transferFromAccountId} onChange={(event) => setTransferFromAccountId(event.target.value)}>
                {accounts.length === 0 && <option value="">Нет счетов</option>}
                {accounts.map((account) => (
                  <option key={account.id} value={account.id}>
                    {account.account_number} · {account.balance} {account.currency}
                  </option>
                ))}
              </select>
            </label>

            <label>
              <span>Номер счета получателя</span>
              <input
                value={transferToAccountNumber}
                onChange={(event) => setTransferToAccountNumber(event.target.value)}
                placeholder="Например: 2200327146406047"
              />
            </label>

            <label>
              <span>Amount</span>
              <input value={transferAmount} onChange={(event) => setTransferAmount(event.target.value)} placeholder="100.00" />
            </label>

            <label>
              <span>Description</span>
              <input value={transferDescription} onChange={(event) => setTransferDescription(event.target.value)} placeholder="Account transfer" />
            </label>

            <div className="transferMfaBox">
              <Button className="secondary" type="button" onClick={requestTransferMFA} disabled={transferMfaState.loading || !token}>
                {transferMfaState.loading ? 'Отправляю...' : 'Запросить MFA'}
              </Button>
              <RequestStatus state={transferMfaState} />
            </div>

            <label>
              <span>MFA code</span>
              <input value={transferMfaCode} onChange={(event) => setTransferMfaCode(event.target.value)} placeholder="6 цифр" />
            </label>

            <Button type="submit" disabled={transferState.loading || !token}>
              {transferState.loading ? 'Перевожу...' : 'Выполнить перевод'}
            </Button>

            <RequestStatus state={transferState} />
          </form>

          {transferResult && (
            <div className="result success">
              <strong>Результат перевода</strong>
              <div className="predictionGrid">
                <div><span>Transaction</span><strong>{transferResult.transaction_id}</strong></div>
                <div><span>From account ID</span><strong>{transferResult.from_account_id}</strong></div>
                <div><span>To account ID</span><strong>{transferResult.to_account_id}</strong></div>
                <div><span>Amount</span><strong>{transferResult.amount}</strong></div>
                <div><span>Status</span><strong>{transferResult.status}</strong></div>
              </div>
              <pre>{JSON.stringify(transferResult, null, 2)}</pre>
            </div>
          )}
        </section>
      </div>
    </Card>
  )
}
