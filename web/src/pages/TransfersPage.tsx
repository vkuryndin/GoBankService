import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { apiRequest } from '../api/http'
import { RequestMessage } from '../components/RequestMessage'
import type { AccountResponse } from '../types/account'
import { emptyState, type RequestState } from '../types/common'
import type { TransferResponse } from '../types/transfer'
import {
  createIdempotencyKey,
  getAccountBadgeClass,
  getAccountStatusText,
} from '../utils/format'

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
  const [accountsState, setAccountsState] = useState<RequestState>(emptyState)
  const [transferMfaState, setTransferMfaState] = useState<RequestState>(emptyState)
  const [transferState, setTransferState] = useState<RequestState>(emptyState)

  const [accounts, setAccounts] = useState<AccountResponse[]>([])
  const [transferFromAccountId, setTransferFromAccountId] = useState(sharedAccountId)
  const [transferToAccountId, setTransferToAccountId] = useState('')
  const [transferAmount, setTransferAmount] = useState('100.00')
  const [transferDescription, setTransferDescription] = useState('Account transfer')
  const [transferMfaCode, setTransferMfaCode] = useState('')
  const [transferResult, setTransferResult] = useState<TransferResponse | null>(null)

  useEffect(() => {
    if (sharedAccountId && !transferFromAccountId) {
      setTransferFromAccountId(sharedAccountId)
    }
  }, [sharedAccountId, transferFromAccountId])

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
      const data = await apiRequest<AccountResponse[]>('/api/accounts', { token })
      setAccounts(Array.isArray(data) ? data : [])
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
    const toAccountID = Number(transferToAccountId)

    if (!Number.isInteger(fromAccountID) || fromAccountID <= 0) {
      setState({ loading: false, error: 'Укажи корректный from_account_id.', success: '' })
      return null
    }

    if (!Number.isInteger(toAccountID) || toAccountID <= 0) {
      setState({ loading: false, error: 'Укажи корректный to_account_id.', success: '' })
      return null
    }

    return { fromAccountID, toAccountID }
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
      await apiRequest<{ message: string }>('/api/mfa/request', {
        method: 'POST',
        token,
        body: {
          purpose: 'transfer',
          from_account_id: ids.fromAccountID,
          to_account_id: ids.toAccountID,
          amount: transferAmount,
        },
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
      const data = await apiRequest<TransferResponse>('/api/transfer', {
        method: 'POST',
        token,
        headers: { 'Idempotency-Key': createIdempotencyKey() },
        body: {
          from_account_id: ids.fromAccountID,
          to_account_id: ids.toAccountID,
          amount: transferAmount,
          description: transferDescription,
          mfa_code: transferMfaCode,
        },
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
    <section className="panel">
      <div className="panelHeader">
        <div>
          <h2>Переводы между счетами</h2>
          <p>
            Перевод выполняется в два шага: сначала MFA-код, потом операция <code>POST /transfer</code>.
          </p>
        </div>

        <div className="actions">
          <button className="secondary" type="button" onClick={loadAccounts} disabled={accountsState.loading || !token}>
            {accountsState.loading ? 'Загружаю...' : 'Загрузить мои счета'}
          </button>
        </div>
      </div>

      <RequestMessage state={accountsState} />

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
                <button
                  key={account.id}
                  className={transferFromAccountId === String(account.id) ? 'accountItem active' : 'accountItem'}
                  type="button"
                  onClick={() => selectFromAccount(account)}
                >
                  <span className="accountNumber">{account.account_number}</span>
                  <span className="accountMeta">
                    <span>ID {account.id}</span>
                    <span>{account.balance} {account.currency}</span>
                    <span className={getAccountBadgeClass(account)}>{getAccountStatusText(account)}</span>
                  </span>
                </button>
              ))}
            </div>
          )}
        </section>

        <section className="subPanel transferPanel">
          <div className="subPanelHeader"><h3>Новый перевод</h3></div>

          <form className="form transferForm" onSubmit={handleTransfer}>
            <label>
              <span>From account ID</span>
              <input value={transferFromAccountId} onChange={(event) => setTransferFromAccountId(event.target.value)} placeholder="ID счета отправителя" />
            </label>

            <label>
              <span>To account ID</span>
              <input value={transferToAccountId} onChange={(event) => setTransferToAccountId(event.target.value)} placeholder="ID счета получателя" />
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
              <button className="secondary" type="button" onClick={requestTransferMFA} disabled={transferMfaState.loading || !token}>
                {transferMfaState.loading ? 'Отправляю...' : 'Запросить MFA'}
              </button>
              <RequestMessage state={transferMfaState} />
            </div>

            <label>
              <span>MFA code</span>
              <input value={transferMfaCode} onChange={(event) => setTransferMfaCode(event.target.value)} placeholder="6 цифр" />
            </label>

            <button type="submit" disabled={transferState.loading || !token}>
              {transferState.loading ? 'Перевожу...' : 'Выполнить перевод'}
            </button>

            <RequestMessage state={transferState} />
          </form>

          {transferResult && (
            <div className="result success">
              <strong>Результат перевода</strong>
              <div className="predictionGrid">
                <div><span>Transaction</span><strong>{transferResult.transaction_id}</strong></div>
                <div><span>From</span><strong>{transferResult.from_account_id}</strong></div>
                <div><span>To</span><strong>{transferResult.to_account_id}</strong></div>
                <div><span>Amount</span><strong>{transferResult.amount}</strong></div>
                <div><span>Status</span><strong>{transferResult.status}</strong></div>
              </div>
              <pre>{JSON.stringify(transferResult, null, 2)}</pre>
            </div>
          )}
        </section>
      </div>
    </section>
  )
}
