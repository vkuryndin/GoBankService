import { Button } from '../../components/ui/Button'
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
            <Button
              key={account.id}
              className={selectedAccountId === String(account.id) ? 'accountItem active' : 'accountItem'}
              type="button"
              onClick={() => onSelect(account)}
              aria-pressed={selectedAccountId === String(account.id)}
            >
              <span className="accountNumber">{account.account_number}</span>
              <span className="accountMeta">
                <span>{account.balance} {account.currency}</span>
                <span className={getAccountBadgeClass(account)}>{getAccountStatusText(account)}</span>
              </span>
            </Button>
          ))}
        </div>
      )}
    </section>
  )
}
