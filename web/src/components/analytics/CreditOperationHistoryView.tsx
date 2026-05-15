import type { CreditOperationResponse } from '../../types/credit'
import { formatDate } from '../../utils/format'

type CreditOperationHistoryViewProps = {
  operations: CreditOperationResponse[]
}

export function CreditOperationHistoryView({ operations }: CreditOperationHistoryViewProps) {
  if (operations.length === 0) {
    return null
  }

  return (
    <div className="tableWrap creditOperationHistoryTable topGap">
      <table>
        <thead>
          <tr>
            <th>Source</th>
            <th>Event</th>
            <th>Amount</th>
            <th>Penalty</th>
            <th>Status</th>
            <th>Transaction</th>
            <th>Schedule</th>
            <th>Payment date</th>
            <th>Occurred at</th>
            <th>Description</th>
          </tr>
        </thead>
        <tbody>
          {operations.map((operation, index) => (
            <tr key={`${operation.source}-${operation.event_type}-${operation.transaction_id || operation.schedule_id || index}`}>
              <td>{operation.source}</td>
              <td>{operation.event_type}</td>
              <td>{operation.amount || '-'}</td>
              <td>{operation.penalty_amount || '-'}</td>
              <td>
                {operation.status ? (
                  <span className={getCreditOperationBadgeClass(operation)}>{operation.status}</span>
                ) : '-'}
              </td>
              <td>{operation.transaction_id || '-'}</td>
              <td>{operation.schedule_id || '-'}</td>
              <td>{formatDate(operation.payment_date)}</td>
              <td>{formatDate(operation.occurred_at)}</td>
              <td>{operation.description || '-'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function getCreditOperationBadgeClass(operation: CreditOperationResponse): string {
  if (operation.status === 'completed' || operation.status === 'paid') {
    return 'badge successBadge'
  }

  if (operation.status === 'failed' || operation.status === 'overdue') {
    return 'badge dangerBadge'
  }

  return 'badge mutedBadge'
}
