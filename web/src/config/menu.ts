import type { MenuItem, MenuKey } from '../types/common'

export const menuItems: MenuItem[] = [
  {
    key: 'health',
    title: 'Health',
    description: 'Проверка доступности backend',
    icon: '●',
    implemented: true,
    path: '/health',
  },
  {
    key: 'register',
    title: 'Register',
    description: 'Регистрация пользователя',
    icon: '+',
    implemented: true,
    path: '/register',
  },
  {
    key: 'auth',
    title: 'Auth',
    description: 'Login, logout и текущий пользователь',
    icon: '↪',
    implemented: true,
    path: '/auth',
  },
  {
    key: 'accounts',
    title: 'Accounts',
    description: 'Счета, deposit, withdraw, close',
    icon: '₽',
    implemented: true,
    path: '/accounts',
  },
  {
    key: 'cards',
    title: 'Cards',
    description: 'Карты, выпуск, просмотр, оплата, перевод, закрытие',
    icon: '▣',
    implemented: true,
    path: '/cards',
  },
  {
    key: 'transfers',
    title: 'Transfers',
    description: 'Переводы между счетами с MFA',
    icon: '⇄',
    implemented: true,
    path: '/transfers',
  },
  {
    key: 'credits',
    title: 'Credits',
    description: 'Проверка, оформление, график',
    icon: '%',
    implemented: true,
    path: '/credits',
  },
  {
    key: 'analytics',
    title: 'Analytics',
    description: 'Доходы, расходы и кредитная нагрузка',
    icon: '↗',
    implemented: true,
    path: '/analytics',
  },
  {
    key: 'rates',
    title: 'Rates',
    description: 'Ключевая и банковская ставка',
    icon: '⌁',
    implemented: true,
    path: '/rates',
  },
  {
    key: 'notifications',
    title: 'Notifications',
    description: 'SMTP test email',
    icon: '✉',
    implemented: true,
    path: '/notifications',
  },
  {
    key: 'admin',
    title: 'Admin',
    description: 'Пользователи, сессии, блокировка счетов',
    icon: '★',
    implemented: true,
    path: '/admin',
  },
]

export function getPageTitle(activeMenu: string): string {
  switch (activeMenu) {
    case 'health':
      return 'Health check'
    case 'register':
      return 'Регистрация'
    case 'auth':
      return 'Авторизация'
    case 'admin':
      return 'Администрирование'
    case 'accounts':
      return 'Счета'
    case 'cards':
      return 'Карты'
    case 'transfers':
      return 'Переводы'
    case 'credits':
      return 'Кредиты'
    case 'analytics':
      return 'Аналитика'
    case 'rates':
      return 'Ставки'
    case 'notifications':
      return 'Уведомления'
    default:
      return 'Go Bank Service'
  }
}

export function getMenuKeyByPath(pathname: string): MenuKey {
  const normalizedPath = pathname === '/' ? '/health' : pathname
  const menuItem = menuItems.find((item) => normalizedPath.startsWith(item.path))

  return menuItem?.key || 'health'
}
