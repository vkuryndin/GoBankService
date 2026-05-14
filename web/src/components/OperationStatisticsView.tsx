import type { OperationStatisticsResponse } from '../types/analytics'
import { formatDate } from '../utils/format'

type OperationStatisticsViewProps = {
  statistics: OperationStatisticsResponse
}

export function OperationStatisticsView({ statistics }: OperationStatisticsViewProps) {
  return (
    <div className="operationStatisticsContent">
      <div className="operationStatsCards">
        <div className="analyticsCard incomeCard">
          <span>Доходы</span>
          <strong>{statistics.total_income} {statistics.currency}</strong>
          <small>{statistics.income_count} операций</small>
        </div>
        <div className="analyticsCard expenseCard">
          <span>Расходы</span>
          <strong>{statistics.total_expense} {statistics.currency}</strong>
          <small>{statistics.expense_count} операций</small>
        </div>
        <div className="analyticsCard creditLoadCard">
          <span>Итог</span>
          <strong>{statistics.net_amount} {statistics.currency}</strong>
          <small>Всего: {statistics.operation_count}</small>
        </div>
      </div>

      <div className="operationStatsTables">
        <div>
          <h4>По типам операций</h4>
          <div className="tableWrap">
            <table className="simpleTable">
              <thead>
                <tr>
                  <th>Тип</th>
                  <th>Кол-во</th>
                  <th>Доход</th>
                  <th>Расход</th>
                  <th>Итог</th>
                </tr>
              </thead>
              <tbody>
                {statistics.by_type.length === 0 && <tr><td colSpan={5}>Нет операций.</td></tr>}
                {statistics.by_type.map((item) => (
                  <tr key={item.type}>
                    <td>{item.type}</td>
                    <td>{item.count}</td>
                    <td>{item.total_income}</td>
                    <td>{item.total_expense}</td>
                    <td>{item.net_amount}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        <div>
          <h4>По статусам</h4>
          <div className="tableWrap">
            <table className="simpleTable">
              <thead>
                <tr>
                  <th>Статус</th>
                  <th>Кол-во</th>
                  <th>Сумма</th>
                </tr>
              </thead>
              <tbody>
                {statistics.by_status.length === 0 && <tr><td colSpan={3}>Нет операций.</td></tr>}
                {statistics.by_status.map((item) => (
                  <tr key={item.status}>
                    <td>{item.status}</td>
                    <td>{item.count}</td>
                    <td>{item.total_amount}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <div>
        <h4>Последние операции</h4>
        <div className="tableWrap operationsTableWrap">
          <table className="simpleTable operationsTable">
            <thead>
              <tr>
                <th>ID</th>
                <th>Дата</th>
                <th>Направление</th>
                <th>Тип</th>
                <th>Сумма</th>
                <th>Счета</th>
                <th>Карты</th>
                <th>Описание</th>
              </tr>
            </thead>
            <tbody>
              {statistics.operations.length === 0 && <tr><td colSpan={8}>Нет операций.</td></tr>}
              {statistics.operations.map((operation) => (
                <tr key={operation.id}>
                  <td>{operation.id}</td>
                  <td>{formatDate(operation.created_at)}</td>
                  <td><span className={getDirectionClass(operation.direction)}>{getDirectionText(operation.direction)}</span></td>
                  <td>{operation.type}</td>
                  <td>{operation.amount} {operation.currency}</td>
                  <td>{formatLinkPair(operation.from_account_id, operation.to_account_id)}</td>
                  <td>{formatLinkPair(operation.from_card_id, operation.to_card_id)}</td>
                  <td>{operation.description || '-'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <details className="rawDetails">
        <summary>Raw response</summary>
        <pre>{JSON.stringify(statistics, null, 2)}</pre>
      </details>
    </div>
  )
}

function getDirectionClass(direction: string): string {
  if (direction === 'income') {
    return 'badge successBadge'
  }

  if (direction === 'expense') {
    return 'badge dangerBadge'
  }

  return 'badge mutedBadge'
}

function getDirectionText(direction: string): string {
  if (direction === 'income') {
    return 'доход'
  }

  if (direction === 'expense') {
    return 'расход'
  }

  return 'нейтр.'
}

function formatLinkPair(fromID?: number, toID?: number): string {
  if (!fromID && !toID) {
    return '-'
  }

  return `${fromID ? String(fromID) : '-'} → ${toID ? String(toID) : '-'}`
}
