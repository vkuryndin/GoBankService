import { Button } from '../../components/ui/Button'
import type { AccountResponse } from '../../types/account'
import { getAccountBadgeClass, getAccountStatusText } from '../../utils/format'

type CreditAccountListPanelProps = {
  accounts: AccountResponse[]
  selectedAccountId: string
  onSelect: (account: AccountResponse) => void
}

export function CreditAccountListPanel({ accounts, selectedAccountId, onSelect }: CreditAccountListPanelProps) {
  return (
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
              className={selectedAccountId === String(account.id) ? 'creditAccountCard selected' : 'creditAccountCard'}
              type="button"
              onClick={() => onSelect(account)}
              aria-pressed={selectedAccountId === String(account.id)}
            >
              <span className="creditAccountNumber">{account.account_number}</span>
              <span className="creditAccountMeta">
                <span>ID {account.id}</span>
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
