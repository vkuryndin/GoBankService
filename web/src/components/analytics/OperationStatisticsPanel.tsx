import type { FormEvent } from 'react'
import { Button } from '../ui/Button'
import { OperationStatisticsView } from '../OperationStatisticsView'
import { RequestStatus } from '../RequestStatus'
import type { OperationStatisticsResponse } from '../../types/analytics'
import type { RequestState } from '../../types/common'

type OperationStatisticsPanelProps = {
  title: string
  description: string
  endpointLabel: string
  limit: string
  state: RequestState
  statistics: OperationStatisticsResponse | null
  disabled?: boolean
  emptyText: string
  onLimitChange: (value: string) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
}

export function OperationStatisticsPanel({
  title,
  description,
  endpointLabel,
  limit,
  state,
  statistics,
  disabled = false,
  emptyText,
  onLimitChange,
  onSubmit,
}: OperationStatisticsPanelProps) {
  return (
    <section className="operationStatisticsSection">
      <div className="subPanelHeader">
        <div>
          <h3>{title}</h3>
          <p className="mutedText">{description}</p>
          <p className="mutedText">Endpoint: <code>{endpointLabel}</code>.</p>
        </div>
      </div>

      <form className="operationStatisticsForm" onSubmit={onSubmit}>
        <label>
          <span>Limit</span>
          <input
            value={limit}
            onChange={(event) => onLimitChange(event.target.value)}
            placeholder="1-500"
            disabled={disabled}
          />
        </label>
        <Button type="submit" disabled={state.loading || disabled}>
          {state.loading ? 'Загружаю...' : 'Получить статистику'}
        </Button>
      </form>

      <RequestStatus state={state} />

      {!statistics && !state.error && (
        <div className="empty compactEmpty">{emptyText}</div>
      )}

      {statistics && <OperationStatisticsView statistics={statistics} />}
    </section>
  )
}
