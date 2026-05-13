import type { MenuItem } from '../types/common'

export const menuItems: MenuItem[] = [
  {
    key: 'health',
    title: 'Health',
    description: 'Проверка доступности backend',
    icon: '●',
    implemented: true,
  },
  {
    key: 'register',
    title: 'Register',
    description: 'Регистрация пользователя',
    icon: '+',
    implemented: true,
  },
  {
    key: 'auth',
    title: 'Auth',
    description: 'Login, logout и текущий пользователь',
    icon: '↪',
    implemented: true,
  },
  {
    key: 'admin',
    title: 'Admin',
    description: 'Пользователи, сессии, блокировка счетов',
    icon: '★',
    implemented: true,
  },
  {
    key: 'accounts',
    title: 'Accounts',
    description: 'Счета, deposit, withdraw, close',
    icon: '₽',
    implemented: true,
  },
  {
    key: 'cards',
    title: 'Cards',
    description: 'Карты, выпуск, просмотр, оплата, перевод, закрытие',
    icon: '▣',
    implemented: true,
  },
  {
    key: 'transfers',
    title: 'Transfers',
    description: 'Переводы между счетами с MFA',
    icon: '⇄',
    implemented: true,
  },
  {
    key: 'credits',
    title: 'Credits',
    description: 'Проверка, оформление, график',
    icon: '%',
    implemented: true,
  },
  {
    key: 'analytics',
    title: 'Analytics',
    description: 'Доходы, расходы и кредитная нагрузка',
    icon: '↗',
    implemented: true,
  },
  {
    key: 'rates',
    title: 'Rates',
    description: 'Ключевая и банковская ставка',
    icon: '⌁',
    implemented: true,
  },
  {
    key: 'notifications',
    title: 'Notifications',
    description: 'SMTP test email',
    icon: '✉',
    implemented: true,
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
