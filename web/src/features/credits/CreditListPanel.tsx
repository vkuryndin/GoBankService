import { Button } from '../../components/ui/Button'
import type { AccountResponse } from '../../types/account'
import type { CreditResponse } from '../../types/credit'

type CreditListPanelProps = {
  selectedAccount: AccountResponse | null
  credits: CreditResponse[]
  selectedCreditId: string
  loading: boolean
  getCreditBadgeClass: (credit: CreditResponse) => string
  onSelect: (credit: CreditResponse) => void
}

export function CreditListPanel({
  selectedAccount,
  credits,
  selectedCreditId,
  loading,
  getCreditBadgeClass,
  onSelect,
}: CreditListPanelProps) {
  return (
    <section className="subPanel creditListPanelV3">
      <div className="subPanelHeader">
        <div>
          <h3>Кредиты выбранного счета</h3>
          <p className="mutedText">
            {selectedAccount ? `account_id ${selectedAccount.id}` : 'Сначала выбери счет.'}
          </p>
        </div>
        <span>{credits.length}</span>
      </div>

      {!selectedAccount && <div className="empty compactEmpty">Выбери счет слева.</div>}

      {selectedAccount && credits.length === 0 && !loading && (
        <div className="empty compactEmpty">У выбранного счета пока нет кредитов.</div>
      )}

      {credits.length > 0 && (
        <div className="creditCardsListV3">
          {credits.map((credit) => (
            <Button
              key={credit.id}
              className={selectedCreditId === String(credit.id) ? 'creditLoanCard selected' : 'creditLoanCard'}
              type="button"
              onClick={() => onSelect(credit)}
              aria-pressed={selectedCreditId === String(credit.id)}
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
  )
}
