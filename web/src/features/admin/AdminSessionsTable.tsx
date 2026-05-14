import { DataTable } from '../../components/ui/DataTable'
import { EmptyState } from '../../components/ui/EmptyState'
import type { AdminSession } from '../../types/admin'
import { formatDate } from '../../utils/format'

type AdminSessionsTableProps = {
  sessions: AdminSession[]
}

export function AdminSessionsTable({ sessions }: AdminSessionsTableProps) {
  if (sessions.length === 0) {
    return <EmptyState>Нажми “Активные сессии”.</EmptyState>
  }

  return (
    <DataTable
      className="compactTable"
      rows={sessions}
      getRowKey={(session) => session.session_id}
      columns={[
        { key: 'session', title: 'Session', render: (session) => session.session_id },
        { key: 'user', title: 'User', render: (session) => session.user_id },
        { key: 'email', title: 'Email', render: (session) => session.email },
        { key: 'username', title: 'Username', render: (session) => session.username },
        { key: 'created', title: 'Created', render: (session) => formatDate(session.created_at) },
        { key: 'expires', title: 'Expires', render: (session) => formatDate(session.expires_at) },
      ]}
    />
  )
}
