import { DataTable } from '../../components/ui/DataTable'
import { EmptyState } from '../../components/ui/EmptyState'
import { StatusBadge } from '../../components/ui/StatusBadge'
import type { AdminUser } from '../../types/admin'
import { formatDate } from '../../utils/format'

type AdminUsersTableProps = {
  users: AdminUser[]
}

export function AdminUsersTable({ users }: AdminUsersTableProps) {
  if (users.length === 0) {
    return <EmptyState>Нажми “Загрузить пользователей”.</EmptyState>
  }

  return (
    <DataTable
      className="compactTable"
      rows={users}
      getRowKey={(user) => user.id}
      columns={[
        { key: 'id', title: 'ID', render: (user) => user.id },
        { key: 'email', title: 'Email', render: (user) => user.email },
        { key: 'username', title: 'Username', render: (user) => user.username },
        {
          key: 'role',
          title: 'Role',
          render: (user) => (
            <StatusBadge tone={user.is_admin ? 'success' : 'muted'}>
              {user.is_admin ? 'admin' : 'user'}
            </StatusBadge>
          ),
        },
        { key: 'accounts', title: 'Accounts', render: (user) => user.accounts_count },
        { key: 'blocked', title: 'Blocked', render: (user) => user.blocked_accounts_count },
        { key: 'created', title: 'Created', render: (user) => formatDate(user.created_at) },
      ]}
    />
  )
}
