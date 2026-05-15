import { Button } from '../../components/ui/Button'
import type { AccountResponse } from '../../types/account'
import type { CreditResponse } from '../../types/credit'
import { formatDate } from '../../utils/format'

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
                <span className="creditLabel">Кредит</span>
                <span className={getCreditBadgeClass(credit)}>{credit.status}</span>
              </span>
              <span className="creditLoanTitle">credit_id {credit.id}</span>
              <span className="creditLoanMetaGrid">
                <span>
                  <small>Account ID</small>
                  <strong>{credit.account_id}</strong>
                </span>
                <span>
                  <small>Principal</small>
                  <strong>{credit.principal_amount}</strong>
                </span>
                <span>
                  <small>Rate</small>
                  <strong>{credit.interest_rate}</strong>
                </span>
                <span>
                  <small>Term</small>
                  <strong>{credit.term_months}</strong>
                </span>
                <span>
                  <small>Monthly</small>
                  <strong>{credit.monthly_payment}</strong>
                </span>
                <span>
                  <small>Created</small>
                  <strong>{formatDate(credit.created_at)}</strong>
                </span>
              </span>
            </Button>
          ))}
        </div>
      )}
    </section>
  )
}
