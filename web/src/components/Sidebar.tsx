import type { MouseEvent as ReactMouseEvent } from 'react'
import { menuItems } from '../config/menu'
import type { MenuKey } from '../types/common'

type SidebarProps = {
  activeMenu: MenuKey
  collapsed: boolean
  width: number
  onMenuChange: (menu: MenuKey) => void
  onCollapsedChange: (collapsed: boolean) => void
  onWidthChange: (width: number) => void
}

const minSidebarWidth = 220
const maxSidebarWidth = 420

export function Sidebar({
  activeMenu,
  collapsed,
  width,
  onMenuChange,
  onCollapsedChange,
  onWidthChange,
}: SidebarProps) {
  const startSidebarResize = (event: ReactMouseEvent<HTMLDivElement>) => {
    if (collapsed) {
      return
    }

    event.preventDefault()

    const startX = event.clientX
    const startWidth = width

    const handleMouseMove = (moveEvent: MouseEvent) => {
      const nextWidth = startWidth + moveEvent.clientX - startX
      onWidthChange(Math.min(maxSidebarWidth, Math.max(minSidebarWidth, nextWidth)))
    }

    const handleMouseUp = () => {
      window.removeEventListener('mousemove', handleMouseMove)
      window.removeEventListener('mouseup', handleMouseUp)
    }

    window.addEventListener('mousemove', handleMouseMove)
    window.addEventListener('mouseup', handleMouseUp)
  }

  return (
    <aside className="sidebar">
      <div className="brand">
        <div className="brandMark">GB</div>
        <div className="brandText">
          <strong>Go Bank</strong>
          <span>REST API frontend</span>
        </div>
        <button
          className="sidebarToggle"
          type="button"
          onClick={() => onCollapsedChange(!collapsed)}
          title={collapsed ? 'Развернуть меню' : 'Свернуть меню'}
          aria-label={collapsed ? 'Развернуть меню' : 'Свернуть меню'}
        >
          {collapsed ? '›' : '‹'}
        </button>
      </div>

      <nav className="menu" aria-label="Основное меню">
        {menuItems.map((item) => (
          <button
            key={item.key}
            className={activeMenu === item.key ? 'menuItem active' : 'menuItem'}
            type="button"
            onClick={() => onMenuChange(item.key)}
            data-tooltip={item.title}
            title={collapsed ? item.title : undefined}
          >
            <span className="menuIcon" aria-hidden="true">
              {item.icon}
            </span>
            <span className="menuText">{item.title}</span>
            {!item.implemented && <small>скоро</small>}
          </button>
        ))}
      </nav>

      <div
        className="sidebarResizer"
        role="separator"
        aria-orientation="vertical"
        aria-label="Изменить ширину меню"
        onMouseDown={startSidebarResize}
      />
    </aside>
  )
}
