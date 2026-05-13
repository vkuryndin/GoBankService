import { RequestStatus } from '../../components/RequestStatus'
import { Button } from '../../components/ui/Button'
import { Card } from '../../components/ui/Card'
import type { RequestState } from '../../types/common'
import type { KeyRateResponse } from '../../types/rate'
import { formatDate } from '../../utils/format'

type CachedKeyRateResponse = KeyRateResponse & {
  fetched_at?: string
}

type RatesViewProps = {
  token: string
  rate: CachedKeyRateResponse | null
  rateState: RequestState
  onLoadRate: () => void
  onClearRate: () => void
}

export function RatesView({ token, rate, rateState, onLoadRate, onClearRate }: RatesViewProps) {
  return (
    <Card variant="plain" className="panel ratesPage">
      <div className="panelHeader">
        <div>
          <h2>Ставки</h2>
          <p>
            Получение ключевой ставки ЦБ РФ и расчет банковской ставки для кредитов.
          </p>
        </div>

        <div className="actions">
          <Button type="button" onClick={onLoadRate} disabled={rateState.loading || !token}>
            {rateState.loading ? 'Загружаю...' : rate ? 'Обновить ставку' : 'Получить ставку'}
          </Button>
          {rate && (
            <Button className="secondary" type="button" onClick={onClearRate}>
              Очистить
            </Button>
          )}
        </div>
      </div>

      <RequestStatus state={rateState} />

      {!rate && !rateState.error && (
        <div className="empty">
          Нажми “Получить ставку”, чтобы запросить данные через backend-интеграцию с ЦБ РФ.
        </div>
      )}

      {rate && (
        <>
          <div className="rateHero">
            <div>
              <span>Ключевая ставка</span>
              <strong>{rate.key_rate}%</strong>
            </div>

            <div>
              <span>Банковская ставка</span>
              <strong>{rate.bank_rate}%</strong>
            </div>

            <div>
              <span>Маржа банка</span>
              <strong>{rate.bank_margin}%</strong>
            </div>
          </div>

          <div className="rateMetaGrid">
            <div>
              <span>Дата ставки</span>
              <strong>{formatDate(rate.date)}</strong>
            </div>
            <div>
              <span>Источник</span>
              <strong>{rate.source}</strong>
            </div>
            <div>
              <span>Загружено во frontend</span>
              <strong>{formatDate(rate.fetched_at)}</strong>
            </div>
          </div>

          <div className="result success">
            <strong>Raw response</strong>
            <pre>{JSON.stringify(rate, null, 2)}</pre>
          </div>
        </>
      )}
    </Card>
  )
}
