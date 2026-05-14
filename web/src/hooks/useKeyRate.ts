import { useEffect, useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { ratesApi } from '../api/ratesApi'
import { emptyState, type RequestState } from '../types/common'
import type { KeyRateResponse } from '../types/rate'

export type CachedKeyRateResponse = KeyRateResponse & {
  fetched_at?: string
}

const rateStorageKey = 'bank_service_key_rate_cache'
const keyRateCacheTtlMs = 12 * 60 * 60 * 1000

function isFreshCache(rate: CachedKeyRateResponse) {
  if (!rate.fetched_at) {
    return false
  }

  const fetchedAt = new Date(rate.fetched_at).getTime()
  if (Number.isNaN(fetchedAt)) {
    return false
  }

  return Date.now() - fetchedAt <= keyRateCacheTtlMs
}

function readCachedRate(): CachedKeyRateResponse | null {
  const rawValue = localStorage.getItem(rateStorageKey)
  if (!rawValue) {
    return null
  }

  try {
    const cachedRate = JSON.parse(rawValue) as CachedKeyRateResponse

    if (!isFreshCache(cachedRate)) {
      localStorage.removeItem(rateStorageKey)
      return null
    }

    return cachedRate
  } catch {
    localStorage.removeItem(rateStorageKey)
    return null
  }
}

function saveCachedRate(rate: CachedKeyRateResponse) {
  localStorage.setItem(rateStorageKey, JSON.stringify(rate))
}

export function useKeyRate(token: string) {
  const [rateState, setRateState] = useState<RequestState>(emptyState)
  const [rate, setRate] = useState<CachedKeyRateResponse | null>(() => readCachedRate())

  useEffect(() => {
    setRate(readCachedRate())
  }, [])

  const mutation = useMutation({
    mutationFn: () => ratesApi.keyRate(token),
    onMutate: () => {
      setRateState({
        loading: true,
        error: '',
        success: '',
      })
    },
    onSuccess: (data) => {
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
    },
    onError: (error) => {
      setRateState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load key rate',
        success: '',
      })
    },
  })

  const loadRate = () => {
    if (!token) {
      setRateState({
        loading: false,
        error: 'Сначала нужно войти в систему.',
        success: '',
      })
      return
    }

    mutation.mutate()
  }

  const clearRate = () => {
    localStorage.removeItem(rateStorageKey)
    setRate(null)
    setRateState(emptyState)
  }

  return {
    rate,
    rateState,
    loadRate,
    clearRate,
    cacheTtlMs: keyRateCacheTtlMs,
  }
}
