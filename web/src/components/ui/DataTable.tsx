import type { ReactNode } from 'react'

type DataTableColumn<T> = {
  key: string
  title: ReactNode
  render: (row: T) => ReactNode
}

type DataTableProps<T> = {
  columns: DataTableColumn<T>[]
  rows: T[]
  getRowKey: (row: T) => string | number
  emptyText?: ReactNode
  className?: string
}

export function DataTable<T>({
  columns,
  rows,
  getRowKey,
  emptyText = 'Нет данных.',
  className = '',
}: DataTableProps<T>) {
  if (rows.length === 0) {
    return <div className="empty compactEmpty">{emptyText}</div>
  }

  const tableClassName = ['tableWrap', className].filter(Boolean).join(' ')

  return (
    <div className={tableClassName}>
      <table>
        <thead>
          <tr>
            {columns.map((column) => (
              <th key={column.key}>{column.title}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={getRowKey(row)}>
              {columns.map((column) => (
                <td key={column.key}>{column.render(row)}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
