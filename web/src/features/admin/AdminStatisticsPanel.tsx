import { DataTable } from '../../components/ui/DataTable'
import { EmptyState } from '../../components/ui/EmptyState'
import { StatusBadge } from '../../components/ui/StatusBadge'
import type { AdminSystemStatistics } from '../../types/admin'
import { formatDate } from '../../utils/format'

type AdminStatisticsPanelProps = {
  statistics: AdminSystemStatistics | null
}

export function AdminStatisticsPanel({ statistics }: AdminStatisticsPanelProps) {
  if (!statistics) {
    return <EmptyState>Нажми “Системная статистика”, чтобы загрузить admin dashboard.</EmptyState>
  }

  return (
    <div className="adminStatisticsContent">
      <div className="adminStatsCards">
        <MetricCard title="Пользователи" value={statistics.users.total} details={`${statistics.users.admins} admin / ${statistics.users.regular_users} user`} />
        <MetricCard title="Активные сессии" value={statistics.users.active_sessions} details={`Новых за 24ч: ${statistics.users.new_last_24h}`} />
        <MetricCard title="Счета" value={statistics.accounts.total} details={`active ${statistics.accounts.active}, blocked ${statistics.accounts.blocked}, closed ${statistics.accounts.closed}`} />
        <MetricCard title="Баланс счетов" value={`${statistics.accounts.total_balance} ${statistics.accounts.currency}`} details="Суммарный баланс всех счетов" />
        <MetricCard title="Карты" value={statistics.cards.total} details={`active ${statistics.cards.active}, closed ${statistics.cards.closed}`} />
        <MetricCard title="Кредиты" value={statistics.credits.total} details={`active ${statistics.credits.active}, overdue ${statistics.credits.overdue}, closed ${statistics.credits.closed}`} />
        <MetricCard title="Активный principal" value={`${statistics.credits.active_principal_amount} ${statistics.credits.currency}`} details={`monthly ${statistics.credits.active_monthly_payment} ${statistics.credits.currency}`} />
        <MetricCard title="Операции" value={statistics.transactions.total} details={`completed ${statistics.transactions.completed}, failed ${statistics.transactions.failed}, 24ч ${statistics.transactions.last_24h}`} />
        <MetricCard title="Объём операций" value={`${statistics.transactions.completed_amount} ${statistics.transactions.currency}`} details={`месяц ${statistics.transactions.completed_this_month} ${statistics.transactions.currency}`} />
        <MetricCard title="Audit events" value={statistics.audit.total} details={`success ${statistics.audit.success}, failed ${statistics.audit.failed}, blocked ${statistics.audit.blocked}`} />
      </div>

      <div className="adminStatsTables">
        <section>
          <h4>Операции по типам</h4>
          <DataTable
            className="compactTable"
            rows={statistics.transactions.by_type}
            getRowKey={(item) => item.type}
            columns={[
              { key: 'type', title: 'Type', render: (item) => item.type },
              { key: 'count', title: 'Count', render: (item) => item.count },
              { key: 'amount', title: 'Amount', render: (item) => `${item.total_amount} ${statistics.transactions.currency}` },
            ]}
          />
        </section>

        <section>
          <h4>Последние операции</h4>
          <DataTable
            className="compactTable"
            rows={statistics.transactions.recent}
            getRowKey={(item) => item.id}
            columns={[
              { key: 'id', title: 'ID', render: (item) => item.id },
              { key: 'user', title: 'User', render: (item) => item.user_id },
              { key: 'type', title: 'Type', render: (item) => item.type },
              { key: 'status', title: 'Status', render: (item) => <StatusBadge tone={item.status === 'completed' ? 'success' : 'danger'}>{item.status}</StatusBadge> },
              { key: 'amount', title: 'Amount', render: (item) => `${item.amount} ${item.currency}` },
              { key: 'created', title: 'Created', render: (item) => formatDate(item.created_at) },
            ]}
          />
        </section>

        <section>
          <h4>Последние audit events</h4>
          <DataTable
            className="compactTable"
            rows={statistics.audit.recent}
            getRowKey={(item) => item.id}
            columns={[
              { key: 'id', title: 'ID', render: (item) => item.id },
              { key: 'user', title: 'User', render: (item) => item.user_id || '-' },
              { key: 'action', title: 'Action', render: (item) => item.action },
              { key: 'resource', title: 'Resource', render: (item) => formatResource(item.resource_type, item.resource_id) },
              { key: 'status', title: 'Status', render: (item) => <StatusBadge tone={item.status === 'success' ? 'success' : item.status === 'blocked' ? 'muted' : 'danger'}>{item.status}</StatusBadge> },
              { key: 'created', title: 'Created', render: (item) => formatDate(item.created_at) },
            ]}
          />
        </section>
      </div>

      <p className="mutedText">Обновлено: {formatDate(statistics.generated_at)}</p>
    </div>
  )
}

type MetricCardProps = {
  title: string
  value: string | number
  details: string
}

function MetricCard({ title, value, details }: MetricCardProps) {
  return (
    <div className="adminStatsCard">
      <span>{title}</span>
      <strong>{value}</strong>
      <small>{details}</small>
    </div>
  )
}

function formatResource(resourceType?: string, resourceID?: number): string {
  if (!resourceType && !resourceID) {
    return '-'
  }

  return `${resourceType || '-'} ${resourceID || ''}`.trim()
}
