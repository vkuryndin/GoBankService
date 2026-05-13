import type { MenuKey } from '../types/common'
import { getPageTitle } from '../config/menu'
import { Card } from '../components/ui/Card'

export function PlaceholderPage({ activeMenu }: { activeMenu: MenuKey }) {
  return (
    <Card variant="plain" className="panel">
      <h2>{getPageTitle(activeMenu)}</h2>
      <div className="empty">
        Раздел добавлен в меню по структуре API. Формы и запросы сделаем следующим шагом.
      </div>
    </Card>
  )
}
