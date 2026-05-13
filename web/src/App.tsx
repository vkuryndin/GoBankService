import { useEffect, useMemo, useState } from 'react'
import type { FormEvent } from 'react'
import './App.css'

type MenuKey =
  | 'health'
  | 'register'
  | 'auth'
  | 'admin'
  | 'accounts'
  | 'cards'
  | 'transfers'
  | 'credits'
  | 'analytics'
  | 'rates'
  | 'notifications'

type RequestState = {
  loading: boolean
  error: string
  success: string
}

type HealthResult = {
  statusCode: number
  body: unknown
}

type RegisterResponse = {
  id: number
  email: string
  username: string
}

type LoginResponse = {
  token: string
}

type AuthCheckResponse = {
  authenticated: boolean
  user_id: number
  email: string
  username: string
  is_admin: boolean
}

type CurrentUser = {
  authenticated: boolean
  user_id: number
  email: string
  username: string
  is_admin: boolean
}

type AdminUser = {
  id: number
  email: string
  username: string
  is_admin: boolean
  accounts_count: number
  blocked_accounts_count: number
  created_at: string
}

type AdminSession = {
  session_id: number
  user_id: number
  email: string
  username: string
  created_at: string
  expires_at: string
}

type AdminAccountStatusResponse = {
  id: number
  user_id: number
  account_number: string
  is_blocked: boolean
  message: string
}

type AccountResponse = {
  id: number
  account_number: string
  balance: string
  currency: string
  is_blocked: boolean
  status: string
  closed_at?: string
  created_at: string
}

type CloseAccountResponse = {
  id: number
  account_number: string
  status: string
  closed_at: string
  message: string
}

type PredictBalanceResponse = {
  account_id: number
  days: number
  current_balance: string
  expected_income: string
  expected_expense: string
  scheduled_credit_payments: string
  predicted_balance: string
  average_daily_income: string
  average_daily_expense: string
  statistics_period_days: number
}

const tokenStorageKey = 'bank_service_token'

const emptyState: RequestState = {
  loading: false,
  error: '',
  success: '',
}

const menuItems: Array<{
  key: MenuKey
  title: string
  description: string
  implemented: boolean
}> = [
  {
    key: 'health',
    title: 'Health',
    description: 'Проверка доступности backend',
    implemented: true,
  },
  {
    key: 'register',
    title: 'Register',
    description: 'Регистрация пользователя',
    implemented: true,
  },
  {
    key: 'auth',
    title: 'Auth',
    description: 'Login, logout и текущий пользователь',
    implemented: true,
  },
  {
    key: 'admin',
    title: 'Admin',
    description: 'Пользователи, сессии, блокировка счетов',
    implemented: true,
  },
  {
    key: 'accounts',
    title: 'Accounts',
    description: 'Счета, deposit, withdraw, close',
    implemented: true,
  },
  {
    key: 'cards',
    title: 'Cards',
    description: 'Карты, оплата, перевод, закрытие',
    implemented: false,
  },
  {
    key: 'transfers',
    title: 'Transfers',
    description: 'Переводы между счетами',
    implemented: false,
  },
  {
    key: 'credits',
    title: 'Credits',
    description: 'Проверка, оформление, график',
    implemented: false,
  },
  {
    key: 'analytics',
    title: 'Analytics',
    description: 'Аналитика и прогноз баланса',
    implemented: false,
  },
  {
    key: 'rates',
    title: 'Rates',
    description: 'Ключевая ставка ЦБ РФ',
    implemented: false,
  },
  {
    key: 'notifications',
    title: 'Notifications',
    description: 'SMTP test email',
    implemented: false,
  },
]

async function readResponseBody(response: Response): Promise<unknown> {
  const contentType = response.headers.get('content-type') || ''

  if (contentType.includes('application/json')) {
    return response.json()
  }

  return response.text()
}

function getErrorMessage(body: unknown): string {
  if (typeof body === 'string') {
    return body
  }

  if (body && typeof body === 'object') {
    const record = body as Record<string, unknown>

    if (typeof record.error === 'string') {
      return record.error
    }

    if (typeof record.message === 'string') {
      return record.message
    }
  }

  return ''
}

async function parseResponse<T>(response: Response): Promise<T> {
  const body = await readResponseBody(response)

  if (!response.ok) {
    throw new Error(getErrorMessage(body) || `HTTP ${response.status}`)
  }

  return body as T
}

function formatDate(value?: string): string {
  if (!value) {
    return '-'
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return date.toLocaleString('ru-RU')
}

function createIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }

  return `${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function isAccountClosed(account?: AccountResponse | null): boolean {
  return account?.status === 'closed'
}

function getAccountBadgeClass(account: AccountResponse): string {
  if (account.status === 'closed') {
    return 'badge mutedBadge'
  }

  if (account.is_blocked) {
    return 'badge dangerBadge'
  }

  return 'badge successBadge'
}

function getAccountStatusText(account: AccountResponse): string {
  if (account.status === 'closed') {
    return 'closed'
  }

  if (account.is_blocked) {
    return 'blocked'
  }

  return 'active'
}

function App() {
  const [activeMenu, setActiveMenu] = useState<MenuKey>('health')

  const [token, setToken] = useState(() => localStorage.getItem(tokenStorageKey) || '')
  const [currentUser, setCurrentUser] = useState<CurrentUser | null>(null)

  const [registerEmail, setRegisterEmail] = useState('')
  const [registerUsername, setRegisterUsername] = useState('')
  const [registerPassword, setRegisterPassword] = useState('')

  const [login, setLogin] = useState('test@example.com')
  const [password, setPassword] = useState('password123')

  const [healthState, setHealthState] = useState<RequestState>(emptyState)
  const [registerState, setRegisterState] = useState<RequestState>(emptyState)
  const [loginState, setLoginState] = useState<RequestState>(emptyState)
  const [authCheckState, setAuthCheckState] = useState<RequestState>(emptyState)
  const [logoutState, setLogoutState] = useState<RequestState>(emptyState)

  const [adminUsersState, setAdminUsersState] = useState<RequestState>(emptyState)
  const [adminSessionsState, setAdminSessionsState] = useState<RequestState>(emptyState)
  const [adminAccountState, setAdminAccountState] = useState<RequestState>(emptyState)

  const [accountsState, setAccountsState] = useState<RequestState>(emptyState)
  const [createAccountState, setCreateAccountState] = useState<RequestState>(emptyState)
  const [accountDetailsState, setAccountDetailsState] = useState<RequestState>(emptyState)
  const [depositState, setDepositState] = useState<RequestState>(emptyState)
  const [withdrawMfaState, setWithdrawMfaState] = useState<RequestState>(emptyState)
  const [withdrawState, setWithdrawState] = useState<RequestState>(emptyState)
  const [predictState, setPredictState] = useState<RequestState>(emptyState)
  const [closeAccountState, setCloseAccountState] = useState<RequestState>(emptyState)

  const [healthResult, setHealthResult] = useState<HealthResult | null>(null)
  const [registeredUser, setRegisteredUser] = useState<RegisterResponse | null>(null)

  const [adminUsers, setAdminUsers] = useState<AdminUser[]>([])
  const [adminSessions, setAdminSessions] = useState<AdminSession[]>([])
  const [adminAccountId, setAdminAccountId] = useState('')
  const [adminAccountResult, setAdminAccountResult] = useState<AdminAccountStatusResponse | null>(null)

  const [accounts, setAccounts] = useState<AccountResponse[]>([])
  const [selectedAccountId, setSelectedAccountId] = useState('')
  const [selectedAccount, setSelectedAccount] = useState<AccountResponse | null>(null)
  const [depositAmount, setDepositAmount] = useState('100.00')
  const [withdrawAmount, setWithdrawAmount] = useState('50.00')
  const [withdrawMfaCode, setWithdrawMfaCode] = useState('')
  const [predictDays, setPredictDays] = useState('30')
  const [predictResult, setPredictResult] = useState<PredictBalanceResponse | null>(null)
  const [closeResult, setCloseResult] = useState<CloseAccountResponse | null>(null)

  const isAuthenticated = token.trim() !== ''

  const maskedToken = useMemo(() => {
    if (!token) {
      return ''
    }

    if (token.length <= 24) {
      return token
    }

    return `${token.slice(0, 14)}...${token.slice(-10)}`
  }, [token])

  useEffect(() => {
    if (token) {
      localStorage.setItem(tokenStorageKey, token)
      void checkCurrentUser(token)
      return
    }

    localStorage.removeItem(tokenStorageKey)
    setCurrentUser(null)
  }, [token])

  const authHeaders = (withJSON = false): Record<string, string> => {
    const headers: Record<string, string> = {
      Accept: 'application/json',
      Authorization: `Bearer ${token}`,
    }

    if (withJSON) {
      headers['Content-Type'] = 'application/json'
    }

    return headers
  }

  const requireToken = (setState: (state: RequestState) => void): boolean => {
    if (token) {
      return true
    }

    setState({
      loading: false,
      error: 'Сначала нужно войти в систему.',
      success: '',
    })

    return false
  }

  const requireAdmin = (setState: (state: RequestState) => void): boolean => {
    if (!requireToken(setState)) {
      return false
    }

    if (currentUser?.is_admin) {
      return true
    }

    setState({
      loading: false,
      error: 'Доступ разрешен только администратору.',
      success: '',
    })

    return false
  }

  const selectedAccountIDNumber = (): number | null => {
    const id = Number(selectedAccountId)
    if (!Number.isInteger(id) || id <= 0) {
      return null
    }

    return id
  }

  const upsertAccount = (account: AccountResponse) => {
    setAccounts((current) => {
      const exists = current.some((item) => item.id === account.id)
      if (!exists) {
        return [account, ...current]
      }

      return current.map((item) => (item.id === account.id ? account : item))
    })

    setSelectedAccount(account)
    setSelectedAccountId(String(account.id))
  }

  const applyClosedAccount = (response: CloseAccountResponse) => {
    setAccounts((current) =>
      current.map((account) =>
        account.id === response.id
          ? {
              ...account,
              status: response.status,
              closed_at: response.closed_at,
            }
          : account,
      ),
    )

    setSelectedAccount((account) =>
      account && account.id === response.id
        ? {
            ...account,
            status: response.status,
            closed_at: response.closed_at,
          }
        : account,
    )
  }

  const applyAdminAccountStatus = (response: AdminAccountStatusResponse) => {
    setAccounts((current) =>
      current.map((account) =>
        account.id === response.id
          ? {
              ...account,
              is_blocked: response.is_blocked,
            }
          : account,
      ),
    )

    setSelectedAccount((account) =>
      account && account.id === response.id
        ? {
            ...account,
            is_blocked: response.is_blocked,
          }
        : account,
    )
  }

  const resetUserData = () => {
    setToken('')
    setCurrentUser(null)
    setAdminUsers([])
    setAdminSessions([])
    setAdminAccountResult(null)
    setAccounts([])
    setSelectedAccount(null)
    setSelectedAccountId('')
    setPredictResult(null)
    setCloseResult(null)
    setLoginState(emptyState)
    setAuthCheckState(emptyState)
    setAdminUsersState(emptyState)
    setAdminSessionsState(emptyState)
    setAdminAccountState(emptyState)
    setAccountsState(emptyState)
    setCreateAccountState(emptyState)
    setAccountDetailsState(emptyState)
    setDepositState(emptyState)
    setWithdrawMfaState(emptyState)
    setWithdrawState(emptyState)
    setPredictState(emptyState)
    setCloseAccountState(emptyState)
  }

  const checkHealth = async () => {
    setHealthState({
      loading: true,
      error: '',
      success: '',
    })
    setHealthResult(null)

    try {
      const response = await fetch('/api/health', {
        method: 'GET',
        headers: {
          Accept: 'application/json',
        },
      })

      const body = await readResponseBody(response)

      setHealthResult({
        statusCode: response.status,
        body,
      })

      setHealthState({
        loading: false,
        error: response.ok ? '' : getErrorMessage(body) || `HTTP ${response.status}`,
        success: response.ok ? 'Backend отвечает.' : '',
      })
    } catch (error) {
      setHealthState({
        loading: false,
        error: error instanceof Error ? error.message : 'Health check failed',
        success: '',
      })
    }
  }

  const handleRegister = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    setRegisterState({
      loading: true,
      error: '',
      success: '',
    })
    setRegisteredUser(null)

    try {
      const response = await fetch('/api/register', {
        method: 'POST',
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          email: registerEmail,
          username: registerUsername,
          password: registerPassword,
        }),
      })

      const data = await parseResponse<RegisterResponse>(response)

      setRegisteredUser(data)
      setLogin(data.email)
      setPassword('')
      setRegisterPassword('')
      setRegisterState({
        loading: false,
        error: '',
        success: 'Пользователь зарегистрирован. Теперь можно войти через Login.',
      })
    } catch (error) {
      setRegisterState({
        loading: false,
        error: error instanceof Error ? error.message : 'Registration failed',
        success: '',
      })
    }
  }

  const handleLogin = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    setLoginState({
      loading: true,
      error: '',
      success: '',
    })
    setAuthCheckState(emptyState)
    setLogoutState(emptyState)

    try {
      const response = await fetch('/api/login', {
        method: 'POST',
        headers: {
          Accept: 'application/json',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          login,
          password,
        }),
      })

      const data = await parseResponse<LoginResponse>(response)

      setToken(data.token)
      setLoginState({
        loading: false,
        error: '',
        success: 'Вход выполнен.',
      })
    } catch (error) {
      resetUserData()
      setLoginState({
        loading: false,
        error: error instanceof Error ? error.message : 'Login failed',
        success: '',
      })
    }
  }

  const checkCurrentUser = async (tokenToCheck = token) => {
    if (!tokenToCheck) {
      setAuthCheckState({
        loading: false,
        error: 'Токен отсутствует.',
        success: '',
      })
      setCurrentUser(null)
      return
    }

    setAuthCheckState({
      loading: true,
      error: '',
      success: '',
    })

    try {
      const response = await fetch('/api/auth/check', {
        method: 'GET',
        headers: {
          Accept: 'application/json',
          Authorization: `Bearer ${tokenToCheck}`,
        },
      })

      const data = await parseResponse<AuthCheckResponse>(response)

      setCurrentUser({
        authenticated: data.authenticated,
        user_id: data.user_id,
        email: data.email,
        username: data.username,
        is_admin: data.is_admin,
      })

      setAuthCheckState({
        loading: false,
        error: '',
        success: `Вы вошли как ${data.email}. Роль: ${
          data.is_admin ? 'администратор' : 'пользователь'
        }.`,
      })
    } catch (error) {
      setToken('')
      setCurrentUser(null)
      setAuthCheckState({
        loading: false,
        error:
          error instanceof Error
            ? error.message
            : 'Failed to check current user',
        success: '',
      })
    }
  }

  const handleLogout = async () => {
    if (!token) {
      resetUserData()
      return
    }

    setLogoutState({
      loading: true,
      error: '',
      success: '',
    })

    try {
      const response = await fetch('/api/logout', {
        method: 'POST',
        headers: authHeaders(),
      })

      await parseResponse<{ message: string }>(response)

      resetUserData()
      setLogoutState({
        loading: false,
        error: '',
        success: 'Вы вышли из системы.',
      })
    } catch (error) {
      setLogoutState({
        loading: false,
        error: error instanceof Error ? error.message : 'Logout failed',
        success: '',
      })
    }
  }

  const loadAdminUsers = async () => {
    if (!requireAdmin(setAdminUsersState)) {
      return
    }

    setAdminUsersState({
      loading: true,
      error: '',
      success: '',
    })

    try {
      const response = await fetch('/api/admin/users', {
        method: 'GET',
        headers: authHeaders(),
      })

      const data = await parseResponse<AdminUser[]>(response)

      setAdminUsers(Array.isArray(data) ? data : [])
      setAdminUsersState({
        loading: false,
        error: '',
        success: 'Список пользователей загружен.',
      })
    } catch (error) {
      setAdminUsersState({
        loading: false,
        error:
          error instanceof Error ? error.message : 'Failed to load users',
        success: '',
      })
    }
  }

  const loadAdminSessions = async () => {
    if (!requireAdmin(setAdminSessionsState)) {
      return
    }

    setAdminSessionsState({
      loading: true,
      error: '',
      success: '',
    })

    try {
      const response = await fetch('/api/admin/logged-in-users', {
        method: 'GET',
        headers: authHeaders(),
      })

      const data = await parseResponse<AdminSession[]>(response)

      setAdminSessions(Array.isArray(data) ? data : [])
      setAdminSessionsState({
        loading: false,
        error: '',
        success: 'Список активных сессий загружен.',
      })
    } catch (error) {
      setAdminSessionsState({
        loading: false,
        error:
          error instanceof Error
            ? error.message
            : 'Failed to load logged in users',
        success: '',
      })
    }
  }

  const changeAdminAccountStatus = async (action: 'block' | 'unblock') => {
    if (!requireAdmin(setAdminAccountState)) {
      return
    }

    const accountID = Number(adminAccountId)
    if (!Number.isInteger(accountID) || accountID <= 0) {
      setAdminAccountState({
        loading: false,
        error: 'Укажи корректный account_id.',
        success: '',
      })
      return
    }

    setAdminAccountState({
      loading: true,
      error: '',
      success: '',
    })
    setAdminAccountResult(null)

    try {
      const response = await fetch(`/api/admin/accounts/${accountID}/${action}`, {
        method: 'POST',
        headers: authHeaders(),
      })

      const data = await parseResponse<AdminAccountStatusResponse>(response)

      setAdminAccountResult(data)
      applyAdminAccountStatus(data)
      setAdminAccountState({
        loading: false,
        error: '',
        success: action === 'block' ? 'Счет заблокирован.' : 'Счет разблокирован.',
      })
    } catch (error) {
      setAdminAccountState({
        loading: false,
        error:
          error instanceof Error ? error.message : `Failed to ${action} account`,
        success: '',
      })
    }
  }

  const loadAccounts = async () => {
    if (!requireToken(setAccountsState)) {
      return
    }

    setAccountsState({
      loading: true,
      error: '',
      success: '',
    })

    try {
      const response = await fetch('/api/accounts', {
        method: 'GET',
        headers: authHeaders(),
      })

      const data = await parseResponse<AccountResponse[]>(response)

      setAccounts(Array.isArray(data) ? data : [])
      setAccountsState({
        loading: false,
        error: '',
        success: 'Список счетов загружен.',
      })

      if (Array.isArray(data) && data.length > 0) {
        const selectedExists = data.some((account) => String(account.id) === selectedAccountId)
        const account = selectedExists
          ? data.find((item) => String(item.id) === selectedAccountId) || data[0]
          : data[0]

        setSelectedAccountId(String(account.id))
        setSelectedAccount(account)
      } else {
        setSelectedAccountId('')
        setSelectedAccount(null)
      }
    } catch (error) {
      setAccountsState({
        loading: false,
        error:
          error instanceof Error ? error.message : 'Failed to load accounts',
        success: '',
      })
    }
  }

  const createAccount = async () => {
    if (!requireToken(setCreateAccountState)) {
      return
    }

    setCreateAccountState({
      loading: true,
      error: '',
      success: '',
    })

    try {
      const response = await fetch('/api/accounts', {
        method: 'POST',
        headers: authHeaders(),
      })

      const account = await parseResponse<AccountResponse>(response)

      upsertAccount(account)
      setCreateAccountState({
        loading: false,
        error: '',
        success: `Счет создан: ${account.account_number}.`,
      })
    } catch (error) {
      setCreateAccountState({
        loading: false,
        error:
          error instanceof Error ? error.message : 'Failed to create account',
        success: '',
      })
    }
  }

  const loadAccountDetails = async () => {
    if (!requireToken(setAccountDetailsState)) {
      return
    }

    const accountID = selectedAccountIDNumber()
    if (!accountID) {
      setAccountDetailsState({
        loading: false,
        error: 'Выбери счет.',
        success: '',
      })
      return
    }

    setAccountDetailsState({
      loading: true,
      error: '',
      success: '',
    })

    try {
      const response = await fetch(`/api/accounts/${accountID}`, {
        method: 'GET',
        headers: authHeaders(),
      })

      const account = await parseResponse<AccountResponse>(response)

      upsertAccount(account)
      setAccountDetailsState({
        loading: false,
        error: '',
        success: 'Данные счета обновлены.',
      })
    } catch (error) {
      setAccountDetailsState({
        loading: false,
        error:
          error instanceof Error ? error.message : 'Failed to load account',
        success: '',
      })
    }
  }

  const handleDeposit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!requireToken(setDepositState)) {
      return
    }

    const accountID = selectedAccountIDNumber()
    if (!accountID) {
      setDepositState({
        loading: false,
        error: 'Выбери счет.',
        success: '',
      })
      return
    }

    setDepositState({
      loading: true,
      error: '',
      success: '',
    })

    try {
      const response = await fetch(`/api/accounts/${accountID}/deposit`, {
        method: 'POST',
        headers: {
          ...authHeaders(true),
          'Idempotency-Key': createIdempotencyKey(),
        },
        body: JSON.stringify({
          amount: depositAmount,
        }),
      })

      const account = await parseResponse<AccountResponse>(response)

      upsertAccount(account)
      setDepositState({
        loading: false,
        error: '',
        success: `Счет пополнен на ${depositAmount} RUB.`,
      })
    } catch (error) {
      setDepositState({
        loading: false,
        error: error instanceof Error ? error.message : 'Deposit failed',
        success: '',
      })
    }
  }

  const requestWithdrawMFA = async () => {
    if (!requireToken(setWithdrawMfaState)) {
      return
    }

    const accountID = selectedAccountIDNumber()
    if (!accountID) {
      setWithdrawMfaState({
        loading: false,
        error: 'Выбери счет.',
        success: '',
      })
      return
    }

    setWithdrawMfaState({
      loading: true,
      error: '',
      success: '',
    })

    try {
      const response = await fetch('/api/mfa/request', {
        method: 'POST',
        headers: authHeaders(true),
        body: JSON.stringify({
          purpose: 'withdraw',
          account_id: accountID,
          amount: withdrawAmount,
        }),
      })

      await parseResponse<{ message: string }>(response)

      setWithdrawMfaState({
        loading: false,
        error: '',
        success: 'MFA-код для списания отправлен.',
      })
    } catch (error) {
      setWithdrawMfaState({
        loading: false,
        error:
          error instanceof Error ? error.message : 'Failed to request MFA code',
        success: '',
      })
    }
  }

  const handleWithdraw = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!requireToken(setWithdrawState)) {
      return
    }

    const accountID = selectedAccountIDNumber()
    if (!accountID) {
      setWithdrawState({
        loading: false,
        error: 'Выбери счет.',
        success: '',
      })
      return
    }

    setWithdrawState({
      loading: true,
      error: '',
      success: '',
    })

    try {
      const response = await fetch(`/api/accounts/${accountID}/withdraw`, {
        method: 'POST',
        headers: {
          ...authHeaders(true),
          'Idempotency-Key': createIdempotencyKey(),
        },
        body: JSON.stringify({
          amount: withdrawAmount,
          mfa_code: withdrawMfaCode,
        }),
      })

      const account = await parseResponse<AccountResponse>(response)

      upsertAccount(account)
      setWithdrawMfaCode('')
      setWithdrawState({
        loading: false,
        error: '',
        success: `Со счета списано ${withdrawAmount} RUB.`,
      })
    } catch (error) {
      setWithdrawState({
        loading: false,
        error: error instanceof Error ? error.message : 'Withdraw failed',
        success: '',
      })
    }
  }

  const loadPrediction = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!requireToken(setPredictState)) {
      return
    }

    const accountID = selectedAccountIDNumber()
    if (!accountID) {
      setPredictState({
        loading: false,
        error: 'Выбери счет.',
        success: '',
      })
      return
    }

    setPredictState({
      loading: true,
      error: '',
      success: '',
    })
    setPredictResult(null)

    try {
      const days = Number(predictDays)
      const query = Number.isInteger(days) && days > 0 ? `?days=${days}` : ''
      const response = await fetch(`/api/accounts/${accountID}/predict${query}`, {
        method: 'GET',
        headers: authHeaders(),
      })

      const data = await parseResponse<PredictBalanceResponse>(response)

      setPredictResult(data)
      setPredictState({
        loading: false,
        error: '',
        success: 'Прогноз баланса получен.',
      })
    } catch (error) {
      setPredictState({
        loading: false,
        error:
          error instanceof Error ? error.message : 'Failed to load prediction',
        success: '',
      })
    }
  }

  const closeAccount = async () => {
    if (!requireToken(setCloseAccountState)) {
      return
    }

    const accountID = selectedAccountIDNumber()
    if (!accountID) {
      setCloseAccountState({
        loading: false,
        error: 'Выбери счет.',
        success: '',
      })
      return
    }

    const confirmed = window.confirm(
      'Закрыть выбранный счет? Закрытие возможно только при нулевом балансе и без активного кредита.',
    )

    if (!confirmed) {
      return
    }

    setCloseAccountState({
      loading: true,
      error: '',
      success: '',
    })
    setCloseResult(null)

    try {
      const response = await fetch(`/api/accounts/${accountID}/close`, {
        method: 'POST',
        headers: {
          ...authHeaders(),
          'Idempotency-Key': createIdempotencyKey(),
        },
      })

      const data = await parseResponse<CloseAccountResponse>(response)

      setCloseResult(data)
      applyClosedAccount(data)
      setCloseAccountState({
        loading: false,
        error: '',
        success: 'Счет закрыт.',
      })
    } catch (error) {
      setCloseAccountState({
        loading: false,
        error:
          error instanceof Error ? error.message : 'Failed to close account',
        success: '',
      })
    }
  }

  const topbarUserText = currentUser
    ? `Вы вошли как ${currentUser.email}. Роль: ${
        currentUser.is_admin ? 'администратор' : 'пользователь'
      }.`
    : authCheckState.loading
      ? 'Проверяю текущего пользователя...'
      : authCheckState.error
        ? 'Сессия не подтверждена. Войдите заново.'
        : isAuthenticated
          ? 'Токен сохранён, пользователь ещё не проверен.'
          : 'Вы не вошли в систему.'

  return (
    <main className="app">
      <aside className="sidebar">
        <div className="brand">
          <div className="brandMark">GB</div>
          <div>
            <strong>Go Bank</strong>
            <span>REST API frontend</span>
          </div>
        </div>

        <nav className="menu">
          {menuItems.map((item) => (
            <button
              key={item.key}
              className={activeMenu === item.key ? 'menuItem active' : 'menuItem'}
              type="button"
              onClick={() => setActiveMenu(item.key)}
            >
              <span>{item.title}</span>
              {!item.implemented && <small>скоро</small>}
            </button>
          ))}
        </nav>
      </aside>

      <section className="content">
        <header className="topbar">
          <div>
            <p className="eyebrow">Bank Service</p>
            <h1>{getPageTitle(activeMenu)}</h1>
            <p className="currentUser">{topbarUserText}</p>
          </div>

          {isAuthenticated && (
            <button
              className="logoutButton"
              type="button"
              onClick={handleLogout}
              disabled={logoutState.loading}
            >
              {logoutState.loading ? 'Выходим...' : 'Выйти'}
            </button>
          )}
        </header>

        {(logoutState.error || logoutState.success) && (
          <div className="panel slimPanel">
            <RequestMessage state={logoutState} />
          </div>
        )}

        {activeMenu === 'health' && (
          <section className="panel">
            <div className="panelHeader">
              <div>
                <h2>Проверка backend</h2>
                <p>
                  Запрос к <code>GET /health</code>.
                </p>
              </div>

              <button type="button" onClick={checkHealth} disabled={healthState.loading}>
                {healthState.loading ? 'Проверяю...' : 'Проверить'}
              </button>
            </div>

            <RequestMessage state={healthState} />

            {healthResult && (
              <div className={healthState.error ? 'result error' : 'result success'}>
                <strong>
                  HTTP status: <code>{healthResult.statusCode}</code>
                </strong>
                <pre>{JSON.stringify(healthResult.body, null, 2)}</pre>
              </div>
            )}
          </section>
        )}

        {activeMenu === 'register' && (
          <section className="panel">
            <div className="panelHeader">
              <div>
                <h2>Регистрация</h2>
                <p>
                  Запрос к <code>POST /register</code>. После регистрации нужно отдельно выполнить login.
                </p>
              </div>
            </div>

            <form className="form" onSubmit={handleRegister}>
              <label>
                <span>Email</span>
                <input
                  value={registerEmail}
                  onChange={(event) => setRegisterEmail(event.target.value)}
                  placeholder="user@example.com"
                  autoComplete="email"
                />
              </label>

              <label>
                <span>Username</span>
                <input
                  value={registerUsername}
                  onChange={(event) => setRegisterUsername(event.target.value)}
                  placeholder="username"
                  autoComplete="username"
                />
              </label>

              <label>
                <span>Password</span>
                <input
                  value={registerPassword}
                  onChange={(event) => setRegisterPassword(event.target.value)}
                  placeholder="минимум 8 символов"
                  type="password"
                  autoComplete="new-password"
                />
              </label>

              <div className="actions">
                <button type="submit" disabled={registerState.loading}>
                  {registerState.loading ? 'Регистрирую...' : 'Зарегистрировать'}
                </button>
              </div>
            </form>

            <RequestMessage state={registerState} />

            {registeredUser && (
              <div className="result success">
                <strong>Пользователь создан</strong>
                <pre>{JSON.stringify(registeredUser, null, 2)}</pre>
                <button
                  className="secondary topGap"
                  type="button"
                  onClick={() => setActiveMenu('auth')}
                >
                  Перейти к Login
                </button>
              </div>
            )}
          </section>
        )}

        {activeMenu === 'auth' && (
          <section className="panel">
            <div className="panelHeader">
              <div>
                <h2>Login и текущий пользователь</h2>
                <p>
                  Запросы к <code>POST /login</code> и <code>GET /auth/check</code>.
                </p>
              </div>

              <button
                className="secondary"
                type="button"
                onClick={() => void checkCurrentUser()}
                disabled={authCheckState.loading || !isAuthenticated}
              >
                {authCheckState.loading ? 'Проверяю...' : 'Кто я сейчас?'}
              </button>
            </div>

            <form className="form" onSubmit={handleLogin}>
              <label>
                <span>Login</span>
                <input
                  value={login}
                  onChange={(event) => setLogin(event.target.value)}
                  placeholder="email или username"
                  autoComplete="username"
                />
              </label>

              <label>
                <span>Password</span>
                <input
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  placeholder="password"
                  type="password"
                  autoComplete="current-password"
                />
              </label>

              <div className="actions">
                <button type="submit" disabled={loginState.loading}>
                  {loginState.loading ? 'Вхожу...' : 'Войти'}
                </button>
              </div>
            </form>

            <RequestMessage state={loginState} />
            <RequestMessage state={authCheckState} />

            {isAuthenticated && (
              <div className="tokenBox">
                <span>Текущий токен</span>
                <code>{maskedToken}</code>
              </div>
            )}

            {currentUser && (
              <div className="currentUserBox">
                <strong>Текущая сессия</strong>
                <p>
                  Вы вошли как <code>{currentUser.email}</code>.
                </p>
                <p>
                  Username: <code>{currentUser.username}</code>
                </p>
                <p>
                  Роль:{' '}
                  <code>
                    {currentUser.is_admin ? 'администратор' : 'пользователь'}
                  </code>
                </p>
                <p>
                  User ID: <code>{currentUser.user_id}</code>
                </p>
              </div>
            )}
          </section>
        )}

        {activeMenu === 'admin' && (
          <section className="panel">
            <div className="panelHeader">
              <div>
                <h2>Администрирование</h2>
                <p>
                  Все admin-действия API: пользователи, активные сессии, блокировка и разблокировка счетов.
                </p>
              </div>

              <div className="actions">
                <button
                  type="button"
                  onClick={loadAdminUsers}
                  disabled={adminUsersState.loading || !isAuthenticated}
                >
                  {adminUsersState.loading ? 'Загружаю...' : 'Загрузить пользователей'}
                </button>
                <button
                  className="secondary"
                  type="button"
                  onClick={loadAdminSessions}
                  disabled={adminSessionsState.loading || !isAuthenticated}
                >
                  {adminSessionsState.loading ? 'Загружаю...' : 'Активные сессии'}
                </button>
              </div>
            </div>

            {!currentUser?.is_admin && (
              <div className="empty">
                Этот раздел доступен только администратору. Войдите под admin-пользователем.
              </div>
            )}

            <div className="adminGrid">
              <section className="subPanel">
                <div className="subPanelHeader">
                  <h3>Пользователи</h3>
                  <span>{adminUsers.length}</span>
                </div>

                <RequestMessage state={adminUsersState} />

                {adminUsers.length === 0 && (
                  <div className="empty">
                    Нажми “Загрузить пользователей”.
                  </div>
                )}

                {adminUsers.length > 0 && (
                  <div className="tableWrap compactTable">
                    <table>
                      <thead>
                        <tr>
                          <th>ID</th>
                          <th>Email</th>
                          <th>Username</th>
                          <th>Role</th>
                          <th>Accounts</th>
                          <th>Blocked</th>
                          <th>Created</th>
                        </tr>
                      </thead>
                      <tbody>
                        {adminUsers.map((user) => (
                          <tr
                            key={user.id}
                            className={currentUser?.user_id === user.id ? 'currentRow' : ''}
                          >
                            <td>{user.id}</td>
                            <td>{user.email}</td>
                            <td>{user.username}</td>
                            <td>
                              <span className={user.is_admin ? 'badge successBadge' : 'badge mutedBadge'}>
                                {user.is_admin ? 'admin' : 'user'}
                              </span>
                            </td>
                            <td>{user.accounts_count}</td>
                            <td>{user.blocked_accounts_count}</td>
                            <td>{formatDate(user.created_at)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </section>

              <section className="subPanel">
                <div className="subPanelHeader">
                  <h3>Активные сессии</h3>
                  <span>{adminSessions.length}</span>
                </div>

                <RequestMessage state={adminSessionsState} />

                {adminSessions.length === 0 && (
                  <div className="empty">
                    Нажми “Активные сессии”.
                  </div>
                )}

                {adminSessions.length > 0 && (
                  <div className="tableWrap compactTable">
                    <table>
                      <thead>
                        <tr>
                          <th>Session</th>
                          <th>User</th>
                          <th>Email</th>
                          <th>Username</th>
                          <th>Created</th>
                          <th>Expires</th>
                        </tr>
                      </thead>
                      <tbody>
                        {adminSessions.map((session) => (
                          <tr
                            key={session.session_id}
                            className={
                              currentUser?.user_id === session.user_id ? 'currentRow' : ''
                            }
                          >
                            <td>{session.session_id}</td>
                            <td>{session.user_id}</td>
                            <td>{session.email}</td>
                            <td>{session.username}</td>
                            <td>{formatDate(session.created_at)}</td>
                            <td>{formatDate(session.expires_at)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </section>

              <section className="subPanel adminAccountPanel">
                <div className="subPanelHeader">
                  <h3>Блокировка счета</h3>
                </div>

                <p className="mutedText">
                  Используй ID счета. Можно взять ID из раздела Accounts или из test.http.
                </p>

                <div className="form adminAccountForm">
                  <label>
                    <span>Account ID</span>
                    <input
                      value={adminAccountId}
                      onChange={(event) => setAdminAccountId(event.target.value)}
                      placeholder="например, 16"
                    />
                  </label>

                  <div className="actions">
                    <button
                      className="danger"
                      type="button"
                      onClick={() => void changeAdminAccountStatus('block')}
                      disabled={adminAccountState.loading || !isAuthenticated}
                    >
                      {adminAccountState.loading ? 'Выполняю...' : 'Заблокировать'}
                    </button>
                    <button
                      className="secondary"
                      type="button"
                      onClick={() => void changeAdminAccountStatus('unblock')}
                      disabled={adminAccountState.loading || !isAuthenticated}
                    >
                      {adminAccountState.loading ? 'Выполняю...' : 'Разблокировать'}
                    </button>
                  </div>
                </div>

                <RequestMessage state={adminAccountState} />

                {adminAccountResult && (
                  <div className="result success">
                    <strong>{adminAccountResult.message}</strong>
                    <pre>{JSON.stringify(adminAccountResult, null, 2)}</pre>
                  </div>
                )}
              </section>
            </div>
          </section>
        )}

        {activeMenu === 'accounts' && (
          <section className="panel">
            <div className="panelHeader">
              <div>
                <h2>Счета пользователя</h2>
                <p>
                  Все основные действия: создание, список, просмотр, deposit, withdraw, close и predict.
                </p>
              </div>

              <div className="actions">
                <button type="button" onClick={loadAccounts} disabled={accountsState.loading || !isAuthenticated}>
                  {accountsState.loading ? 'Загружаю...' : 'Загрузить счета'}
                </button>
                <button
                  className="secondary"
                  type="button"
                  onClick={createAccount}
                  disabled={createAccountState.loading || !isAuthenticated}
                >
                  {createAccountState.loading ? 'Создаю...' : 'Создать счет'}
                </button>
              </div>
            </div>

            <RequestMessage state={accountsState} />
            <RequestMessage state={createAccountState} />

            <div className="accountsLayout">
              <section className="subPanel">
                <div className="subPanelHeader">
                  <h3>Мои счета</h3>
                  <span>{accounts.length}</span>
                </div>

                {accounts.length === 0 && (
                  <div className="empty">
                    Список пуст. Нажми “Загрузить счета” или “Создать счет”.
                  </div>
                )}

                {accounts.length > 0 && (
                  <div className="accountList">
                    {accounts.map((account) => (
                      <button
                        key={account.id}
                        className={
                          selectedAccountId === String(account.id)
                            ? 'accountItem active'
                            : 'accountItem'
                        }
                        type="button"
                        onClick={() => {
                          setSelectedAccountId(String(account.id))
                          setSelectedAccount(account)
                          setPredictResult(null)
                          setCloseResult(null)
                          setAdminAccountId(String(account.id))
                        }}
                      >
                        <span className="accountNumber">{account.account_number}</span>
                        <span className="accountMeta">
                          <span>{account.balance} {account.currency}</span>
                          <span className={getAccountBadgeClass(account)}>
                            {getAccountStatusText(account)}
                          </span>
                        </span>
                      </button>
                    ))}
                  </div>
                )}
              </section>

              <section className="subPanel">
                <div className="subPanelHeader">
                  <h3>Выбранный счет</h3>
                  {selectedAccount && (
                    <span className={getAccountBadgeClass(selectedAccount)}>
                      {getAccountStatusText(selectedAccount)}
                    </span>
                  )}
                </div>

                {!selectedAccount && (
                  <div className="empty">
                    Выбери счет из списка слева.
                  </div>
                )}

                {selectedAccount && (
                  <>
                    <div className="detailsGrid">
                      <div>
                        <span>ID</span>
                        <strong>{selectedAccount.id}</strong>
                      </div>
                      <div>
                        <span>Номер</span>
                        <strong>{selectedAccount.account_number}</strong>
                      </div>
                      <div>
                        <span>Баланс</span>
                        <strong>{selectedAccount.balance} {selectedAccount.currency}</strong>
                      </div>
                      <div>
                        <span>Создан</span>
                        <strong>{formatDate(selectedAccount.created_at)}</strong>
                      </div>
                      <div>
                        <span>Blocked</span>
                        <strong>{selectedAccount.is_blocked ? 'yes' : 'no'}</strong>
                      </div>
                      <div>
                        <span>Closed at</span>
                        <strong>{formatDate(selectedAccount.closed_at)}</strong>
                      </div>
                    </div>

                    <div className="actions topGap">
                      <button
                        className="secondary"
                        type="button"
                        onClick={loadAccountDetails}
                        disabled={accountDetailsState.loading}
                      >
                        {accountDetailsState.loading ? 'Обновляю...' : 'Обновить данные'}
                      </button>
                    </div>

                    <RequestMessage state={accountDetailsState} />

                    <div className="accountActionsGrid">
                      <form className="actionBox" onSubmit={handleDeposit}>
                        <h4>Пополнение</h4>
                        <p>Запрос к <code>POST /accounts/{'{accountId}'}/deposit</code>.</p>
                        <label>
                          <span>Amount</span>
                          <input
                            value={depositAmount}
                            onChange={(event) => setDepositAmount(event.target.value)}
                            placeholder="100.00"
                            disabled={isAccountClosed(selectedAccount)}
                          />
                        </label>
                        <button type="submit" disabled={depositState.loading || isAccountClosed(selectedAccount)}>
                          {depositState.loading ? 'Пополняю...' : 'Пополнить'}
                        </button>
                        <RequestMessage state={depositState} />
                      </form>

                      <form className="actionBox" onSubmit={handleWithdraw}>
                        <h4>Списание с MFA</h4>
                        <p>Сначала запроси MFA-код, потом выполни withdraw.</p>
                        <label>
                          <span>Amount</span>
                          <input
                            value={withdrawAmount}
                            onChange={(event) => setWithdrawAmount(event.target.value)}
                            placeholder="50.00"
                            disabled={isAccountClosed(selectedAccount)}
                          />
                        </label>
                        <button
                          className="secondary"
                          type="button"
                          onClick={requestWithdrawMFA}
                          disabled={withdrawMfaState.loading || isAccountClosed(selectedAccount)}
                        >
                          {withdrawMfaState.loading ? 'Отправляю...' : 'Запросить MFA'}
                        </button>
                        <RequestMessage state={withdrawMfaState} />
                        <label>
                          <span>MFA code</span>
                          <input
                            value={withdrawMfaCode}
                            onChange={(event) => setWithdrawMfaCode(event.target.value)}
                            placeholder="6 цифр"
                            disabled={isAccountClosed(selectedAccount)}
                          />
                        </label>
                        <button type="submit" disabled={withdrawState.loading || isAccountClosed(selectedAccount)}>
                          {withdrawState.loading ? 'Списываю...' : 'Списать'}
                        </button>
                        <RequestMessage state={withdrawState} />
                      </form>

                      <form className="actionBox" onSubmit={loadPrediction}>
                        <h4>Прогноз баланса</h4>
                        <p>Запрос к <code>GET /accounts/{'{accountId}'}/predict</code>.</p>
                        <label>
                          <span>Days</span>
                          <input
                            value={predictDays}
                            onChange={(event) => setPredictDays(event.target.value)}
                            placeholder="30"
                          />
                        </label>
                        <button type="submit" disabled={predictState.loading}>
                          {predictState.loading ? 'Считаю...' : 'Получить прогноз'}
                        </button>
                        <RequestMessage state={predictState} />
                      </form>

                      <div className="actionBox dangerZone">
                        <h4>Закрытие счета</h4>
                        <p>Закрытие возможно только при нулевом балансе и без активного кредита.</p>
                        <button
                          className="danger"
                          type="button"
                          onClick={closeAccount}
                          disabled={closeAccountState.loading || isAccountClosed(selectedAccount)}
                        >
                          {closeAccountState.loading ? 'Закрываю...' : 'Закрыть счет'}
                        </button>
                        <RequestMessage state={closeAccountState} />
                      </div>
                    </div>

                    {predictResult && (
                      <div className="result success">
                        <strong>Прогноз баланса</strong>
                        <div className="predictionGrid">
                          <div>
                            <span>Current</span>
                            <strong>{predictResult.current_balance}</strong>
                          </div>
                          <div>
                            <span>Income</span>
                            <strong>{predictResult.expected_income}</strong>
                          </div>
                          <div>
                            <span>Expense</span>
                            <strong>{predictResult.expected_expense}</strong>
                          </div>
                          <div>
                            <span>Credit payments</span>
                            <strong>{predictResult.scheduled_credit_payments}</strong>
                          </div>
                          <div>
                            <span>Predicted</span>
                            <strong>{predictResult.predicted_balance}</strong>
                          </div>
                          <div>
                            <span>Days</span>
                            <strong>{predictResult.days}</strong>
                          </div>
                        </div>
                        <pre>{JSON.stringify(predictResult, null, 2)}</pre>
                      </div>
                    )}

                    {closeResult && (
                      <div className="result success">
                        <strong>Результат закрытия счета</strong>
                        <pre>{JSON.stringify(closeResult, null, 2)}</pre>
                      </div>
                    )}
                  </>
                )}
              </section>
            </div>
          </section>
        )}

        {!menuItems.find((item) => item.key === activeMenu)?.implemented && (
          <section className="panel">
            <h2>{getPageTitle(activeMenu)}</h2>
            <div className="empty">
              Раздел добавлен в меню по структуре API. Формы и запросы сделаем следующим шагом.
            </div>
          </section>
        )}
      </section>
    </main>
  )
}

function RequestMessage({ state }: { state: RequestState }) {
  if (state.error) {
    return <div className="alert">{state.error}</div>
  }

  if (state.success) {
    return <div className="notice">{state.success}</div>
  }

  return null
}

function getPageTitle(activeMenu: MenuKey): string {
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

export default App
