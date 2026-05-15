import type { KeyboardEvent as ReactKeyboardEvent, MouseEvent as ReactMouseEvent } from 'react'
import { NavLink } from 'react-router-dom'
import { menuItems } from '../config/menu'
import type { MenuKey } from '../types/common'
import { useAuth } from '../hooks/useAuth'
import { Button } from './ui/Button'

type SidebarProps = {
  activeMenu: MenuKey
  collapsed: boolean
  width: number
  onCollapsedChange: (collapsed: boolean) => void
  onWidthChange: (width: number) => void
}

const minSidebarWidth = 220
const maxSidebarWidth = 420

export function Sidebar({
  activeMenu,
  collapsed,
  width,
  onCollapsedChange,
  onWidthChange,
}: SidebarProps) {
  const { currentUser, isAuthenticated } = useAuth()
  const visibleMenuItems = isAuthenticated
    ? menuItems.filter((item) => {
        if (item.key === 'auth' || item.key === 'register') {
          return false
        }

        if (item.key === 'admin') {
          return Boolean(currentUser?.is_admin)
        }

        return true
      })
    : menuItems

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


  const handleResizeKeyDown = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    if (collapsed) {
      return
    }

    if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') {
      return
    }

    event.preventDefault()
    const direction = event.key === 'ArrowRight' ? 12 : -12
    const nextWidth = Math.min(maxSidebarWidth, Math.max(minSidebarWidth, width + direction))
    onWidthChange(nextWidth)
  }

  return (
    <aside className="sidebar">
      <div className="brand">
        <div className="brandMark">GB</div>
        <div className="brandText">
          <strong>Go Bank</strong>
          <span>REST API frontend</span>
        </div>
        <Button
          className="sidebarToggle"
          type="button"
          onClick={() => onCollapsedChange(!collapsed)}
          title={collapsed ? 'Развернуть меню' : 'Свернуть меню'}
          aria-label={collapsed ? 'Развернуть меню' : 'Свернуть меню'}
        >
          {collapsed ? '›' : '‹'}
        </Button>
      </div>

      <nav className="menu" aria-label="Основное меню">
        {visibleMenuItems.map((item) => (
          <NavLink
            key={item.key}
            className={({ isActive }) =>
              isActive || activeMenu === item.key ? 'menuItem active' : 'menuItem'
            }
            to={item.path}
            data-tooltip={item.title}
            title={collapsed ? item.title : undefined}
          >
            <span className="menuIcon" aria-hidden="true">
              {item.icon}
            </span>
            <span className="menuText">{item.title}</span>
            {!item.implemented && <small>скоро</small>}
          </NavLink>
        ))}
      </nav>

      <div
        className="sidebarResizer"
        role="separator"
        aria-orientation="vertical"
        aria-label="Изменить ширину меню"
        tabIndex={collapsed ? -1 : 0}
        onMouseDown={startSidebarResize}
        onKeyDown={handleResizeKeyDown}
      />
    </aside>
  )
}
