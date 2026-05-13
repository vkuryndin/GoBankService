import type { MenuKey } from '../types/common'
import { getPageTitle } from '../config/menu'

export function PlaceholderPage({ activeMenu }: { activeMenu: MenuKey }) {
  return (
    <section className="panel">
      <h2>{getPageTitle(activeMenu)}</h2>
      <div className="empty">
        Раздел добавлен в меню по структуре API. Формы и запросы сделаем следующим шагом.
      </div>
    </section>
  )
}
