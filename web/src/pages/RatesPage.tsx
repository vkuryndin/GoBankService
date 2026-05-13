import { useEffect, useState } from 'react'
import { apiRequest } from '../api/http'
import { RequestMessage } from '../components/RequestMessage'
import { emptyState, type RequestState } from '../types/common'
import type { KeyRateResponse } from '../types/rate'
import { formatDate } from '../utils/format'

type RatesPageProps = {
  token: string
}

type CachedKeyRateResponse = KeyRateResponse & {
  fetched_at?: string
}

const rateStorageKey = 'bank_service_key_rate_cache'

function readCachedRate(): CachedKeyRateResponse | null {
  const rawValue = localStorage.getItem(rateStorageKey)
  if (!rawValue) {
    return null
  }

  try {
    return JSON.parse(rawValue) as CachedKeyRateResponse
  } catch {
    localStorage.removeItem(rateStorageKey)
    return null
  }
}

function saveCachedRate(rate: CachedKeyRateResponse) {
  localStorage.setItem(rateStorageKey, JSON.stringify(rate))
}

export function RatesPage({ token }: RatesPageProps) {
  const [rateState, setRateState] = useState<RequestState>(emptyState)
  const [rate, setRate] = useState<CachedKeyRateResponse | null>(() => readCachedRate())

  useEffect(() => {
    setRate(readCachedRate())
  }, [])

  const loadRate = async () => {
    if (!token) {
      setRateState({
        loading: false,
        error: 'Сначала нужно войти в систему.',
        success: '',
      })
      return
    }

    setRateState({
      loading: true,
      error: '',
      success: '',
    })

    try {
      const data = await apiRequest<KeyRateResponse>('/api/rates/key', { token })
      const cachedRate: CachedKeyRateResponse = {
        ...data,
        fetched_at: new Date().toISOString(),
      }

      setRate(cachedRate)
      saveCachedRate(cachedRate)
      setRateState({
        loading: false,
        error: '',
        success: 'Ключевая ставка загружена.',
      })
    } catch (error) {
      setRateState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load key rate',
        success: '',
      })
    }
  }

  const clearRate = () => {
    localStorage.removeItem(rateStorageKey)
    setRate(null)
    setRateState(emptyState)
  }

  return (
    <section className="panel ratesPage">
      <div className="panelHeader">
        <div>
          <h2>Ставки</h2>
          <p>
            Получение ключевой ставки ЦБ РФ и расчет банковской ставки для кредитов.
          </p>
        </div>

        <div className="actions">
          <button type="button" onClick={loadRate} disabled={rateState.loading || !token}>
            {rateState.loading ? 'Загружаю...' : rate ? 'Обновить ставку' : 'Получить ставку'}
          </button>
          {rate && (
            <button className="secondary" type="button" onClick={clearRate}>
              Очистить
            </button>
          )}
        </div>
      </div>

      <RequestMessage state={rateState} />

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
    </section>
  )
}
