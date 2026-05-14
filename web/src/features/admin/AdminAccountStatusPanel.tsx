import { RequestStatus } from '../../components/RequestStatus'
import { Button } from '../../components/ui/Button'
import { Field } from '../../components/ui/Field'
import { Input } from '../../components/ui/Input'
import type { AdminAccountStatusResponse } from '../../types/admin'
import type { RequestState } from '../../types/common'

type AdminAccountStatusPanelProps = {
  accountId: string
  state: RequestState
  result: AdminAccountStatusResponse | null
  disabled: boolean
  onAccountIdChange: (value: string) => void
  onBlock: () => void
  onUnblock: () => void
}

export function AdminAccountStatusPanel({
  accountId,
  state,
  result,
  disabled,
  onAccountIdChange,
  onBlock,
  onUnblock,
}: AdminAccountStatusPanelProps) {
  return (
    <section className="subPanel adminAccountPanel">
      <div className="subPanelHeader">
        <h3>Блокировка счета</h3>
      </div>

      <p className="mutedText">
        Используй ID счета. Можно взять ID из раздела Accounts или из test.http.
      </p>

      <div className="form adminAccountForm">
        <Field label="Account ID">
          <Input
            value={accountId}
            onChange={(event) => onAccountIdChange(event.target.value)}
            placeholder="например, 16"
            inputMode="numeric"
            aria-label="Account ID для блокировки или разблокировки"
          />
        </Field>

        <div className="actions">
          <Button className="danger" type="button" onClick={onBlock} disabled={state.loading || disabled}>
            {state.loading ? 'Выполняю...' : 'Заблокировать'}
          </Button>
          <Button className="secondary" type="button" onClick={onUnblock} disabled={state.loading || disabled}>
            {state.loading ? 'Выполняю...' : 'Разблокировать'}
          </Button>
        </div>
      </div>

      <RequestStatus state={state} />

      {result && (
        <div className="result success">
          <strong>{result.message}</strong>
          <pre>{JSON.stringify(result, null, 2)}</pre>
        </div>
      )}
    </section>
  )
}
