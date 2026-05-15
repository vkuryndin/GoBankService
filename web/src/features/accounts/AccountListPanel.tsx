import type { AccountResponse } from '../../types/account'
import { getAccountBadgeClass, getAccountStatusText } from '../../utils/format'

type AccountListPanelProps = {
  accounts: AccountResponse[]
  selectedAccountId: string
  onSelect: (account: AccountResponse) => void
}

export function AccountListPanel({ accounts, selectedAccountId, onSelect }: AccountListPanelProps) {
  return (
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
            <div
              key={account.id}
              className={selectedAccountId === String(account.id) ? 'accountItem active' : 'accountItem'}
              role="button"
              tabIndex={0}
              onClick={() => onSelect(account)}
              onKeyDown={(event) => {
                if (event.key === 'Enter' || event.key === ' ') {
                  event.preventDefault()
                  onSelect(account)
                }
              }}
              aria-pressed={selectedAccountId === String(account.id)}
            >
              <span className="accountItemTop">
                <span className="accountLabel">Счет</span>
                <span className={getAccountBadgeClass(account)}>{getAccountStatusText(account)}</span>
              </span>
              <span className="accountNumber" onMouseDown={(event) => event.stopPropagation()} onClick={(event) => event.stopPropagation()}>{account.account_number}</span>
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
            </div>
          ))}
        </div>
      )}
    </section>
  )
}
