# Bank Service REST API

Сессионное задание по дисциплине "Язык программирования GO".

REST API банковского сервиса на Go. Проект реализует регистрацию пользователей, JWT-аутентификацию, счета, карты, переводы, кредиты, аналитику, интеграцию с ЦБ РФ и SMTP-уведомления.

Сервис также доступен на тестовом сервере:

```text
Landing page: http://18.185.7.63/
Frontend:     http://18.185.7.63/app/
API base URL: http://18.185.7.63/api
```

Для проверки API через test.http на сервере нужно указывать:

```
@baseUrl = http://18.185.7.63/api
```

Для локального запуска backend без nginx:

```
@baseUrl = http://localhost:8080
```

Сервер будет доступен временно.

## Стек

- Go 1.26.3
- gorilla/mux
- PostgreSQL + lib/pq
- pgcrypto
- JWT
- bcrypt
- HMAC-SHA256
- logrus
- gomail
- beevik/etree
- React
- TypeScript
- Vite
- Tailwind CSS
- nginx

## Основные возможности

- Регистрация и вход пользователя.
- JWT с временем жизни 24 часа.
- Повторный login инвалидирует старые активные токены пользователя.
- Logout и revoked tokens.
- Создание и просмотр счетов.
- Закрытие счета через soft close.
- Пополнение счета.
- Списание со счета с MFA.
- Переводы между счетами с MFA.
- Выпуск виртуальных карт.
- Просмотр своих карт.
- Оплата картой с CVV и MFA.
- Перевод с карты на карту с CVV и MFA.
- Закрытие карты через soft close.
- Оформление кредита с MFA.
- Предварительная проверка кредита с понятными причинами отказа.
- Расчет аннуитетных платежей.
- График платежей по кредиту.
- Автоматическое списание кредитных платежей.
- Просрочка и штраф +10%.
- Автоматическое закрытие кредита после оплаты всех платежей.
- Финансовая аналитика.
- Прогноз баланса.
- Получение ключевой ставки ЦБ РФ через SOAP.
- SMTP-уведомления.
- Admin API для просмотра пользователей, активных сессий и блокировки счетов.
- React frontend для ручной проверки всех основных сценариев.
- Frontend доступен по `/app/`.
- На сервере API доступен через nginx по `/api/...`.
- Стартовая страница API доступна по `/`.

## Безопасность

В проекте реализованы:

- bcrypt-хеширование паролей;
- bcrypt-хеширование CVV;
- PGP-шифрование номера и срока карты через pgcrypto;
- HMAC-SHA256 для проверки целостности данных карты;
- HMAC карты используется только внутри сервиса и не возвращается в API-ответах;
- сравнение HMAC выполняется через constant-time compare;
- JWT-аутентификация;
- JWT `jti`;
- single-session login;
- revoked tokens;
- проверка владельца счетов, карт и кредитов;
- MFA для критических операций: withdraw, transfer, card payment, card transfer, credit create;
- MFA-код атомарно помечается использованным, чтобы исключить повторное применение одного кода при параллельных запросах;
- ограничение попыток MFA и CVV;
- Idempotency-Key для финансовых POST-операций;
- request_hash для защиты от переиспользования Idempotency-Key с другим body;
- rate limiting;
- ограничение размера request body;
- HTTP server timeouts;
- security headers;
- request id;
- strict JSON decoding;
- strict parsing конфигурации: ошибки в bool/int env-переменных не подменяются молча default-значениями;
- audit logs;
- DB CHECK constraints;
- graceful shutdown.

## Структура проекта

```text
cmd/server              точка входа, инициализация зависимостей, запуск HTTP-сервера
internal/config         конфигурация приложения
internal/db             подключение к PostgreSQL
internal/models         модели БД
internal/dto            DTO запросов и ответов
internal/repositories   SQL-запросы и транзакции
internal/services       бизнес-логика и правила приложения
internal/handlers       HTTP handlers
internal/middleware     middleware
internal/router         маршрутизация и регистрация маршрутов
internal/security       JWT, bcrypt, карты, HMAC
internal/integrations   внешние интеграции: ЦБ РФ и SMTP
internal/scheduler      фоновые задачи
internal/audit          audit events
migrations              SQL-миграции
Dockerfile              сборка Go API
docker-compose.yml      запуск API, PostgreSQL и nginx
nginx                   nginx reverse proxy и стартовая страница
web                     React frontend
test.http               ручные сценарии проверки API
```

### Полная структура проекта

```text
.
├── .dockerignore
├── .env.example
├── .gitignore
├── .golangci.yml
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
├── cmd
│   └── server
│       └── main.go
├── internal
│   ├── audit
│   │   └── event.go
│   ├── config
│   │   └── config.go
│   ├── db
│   │   └── postgres.go
│   ├── dto
│   │   ├── account.go
│   │   ├── admin.go
│   │   ├── analytics.go
│   │   ├── auth.go
│   │   ├── card.go
│   │   ├── common.go
│   │   ├── credit.go
│   │   ├── mfa.go
│   │   ├── rate.go
│   │   └── transfer.go
│   ├── handlers
│   │   ├── account_handler.go
│   │   ├── admin_handler.go
│   │   ├── analytics_handler.go
│   │   ├── audit.go
│   │   ├── auth_context.go
│   │   ├── auth_handler.go
│   │   ├── card_handler.go
│   │   ├── credit_handler.go
│   │   ├── endpoint.go
│   │   ├── error_mapping.go
│   │   ├── health_handler.go
│   │   ├── mfa_handler.go
│   │   ├── notification_handler.go
│   │   ├── path_params.go
│   │   ├── rate_handler.go
│   │   ├── request.go
│   │   ├── response.go
│   │   └── transfer_handler.go
│   ├── integrations
│   │   ├── cbr
│   │   │   └── client.go
│   │   └── smtp
│   │       └── client.go
│   ├── middleware
│   │   ├── admin_middleware.go
│   │   ├── auth_middleware.go
│   │   ├── body_limit.go
│   │   ├── error_response.go
│   │   ├── idempotency.go
│   │   ├── logging_middleware.go
│   │   ├── rate_limiter.go
│   │   ├── request_id.go
│   │   └── security_headers.go
│   ├── models
│   │   ├── account.go
│   │   ├── card.go
│   │   ├── credit.go
│   │   ├── payment_schedule.go
│   │   ├── transaction.go
│   │   └── user.go
│   ├── repositories
│   │   ├── account_repository.go
│   │   ├── admin_repository.go
│   │   ├── analytics_repository.go
│   │   ├── audit_repository.go
│   │   ├── card_repository.go
│   │   ├── credit_payment_repository.go
│   │   ├── credit_repository.go
│   │   ├── idempotency_repository.go
│   │   ├── mfa_repository.go
│   │   ├── token_repository.go
│   │   ├── user_repository.go
│   │   └── user_session_repository.go
│   ├── router
│   │   └── router.go
│   ├── scheduler
│   │   ├── credit_payment_scheduler.go
│   │   ├── idempotency_cleanup_scheduler.go
│   │   ├── mfa_cleanup_scheduler.go
│   │   └── token_cleanup_scheduler.go
│   ├── security
│   │   ├── httpauth
│   │   │   └── bearer.go
│   │   ├── card.go
│   │   ├── jwt.go
│   │   ├── password.go
│   │   └── token.go
│   └── services
│       ├── account_service.go
│       ├── admin_service.go
│       ├── analytics_service.go
│       ├── attempt_limiter.go
│       ├── audit_service.go
│       ├── auth_service.go
│       ├── card_processing_service.go
│       ├── card_service.go
│       ├── credit_payment_service.go
│       ├── credit_service.go
│       ├── limits.go
│       ├── mfa_service.go
│       ├── money.go
│       ├── notification_service.go
│       ├── rate_service.go
│       └── transfer_service.go
├── migrations
│   └── 001_init.sql
├── nginx
│   ├── bank-service.conf
│   └── html
│       └── index.html
├── web
│   ├── package.json
│   ├── package-lock.json
│   ├── vite.config.ts
│   ├── tsconfig.json
│   ├── tsconfig.app.json
│   ├── tsconfig.node.json
│   ├── index.html
│   └── src
│       ├── api
│       │   ├── adminApi.ts
│       │   ├── analyticsApi.ts
│       │   ├── accountsApi.ts
│       │   ├── authApi.ts
│       │   ├── cardsApi.ts
│       │   ├── client.ts
│       │   ├── creditsApi.ts
│       │   ├── http.ts
│       │   ├── mfaApi.ts
│       │   ├── notificationsApi.ts
│       │   ├── ratesApi.ts
│       │   └── transfersApi.ts
│       ├── components
│       │   ├── RequestMessage.tsx
│       │   ├── RequestStatus.tsx
│       │   ├── Sidebar.tsx
│       │   ├── Topbar.tsx
│       │   └── ui
│       │       ├── Button.tsx
│       │       ├── Card.tsx
│       │       └── StatusBadge.tsx
│       ├── config
│       │   └── menu.ts
│       ├── pages
│       │   ├── AccountsPage.tsx
│       │   ├── AdminPage.tsx
│       │   ├── AnalyticsPage.tsx
│       │   ├── AuthPage.tsx
│       │   ├── CardsPage.tsx
│       │   ├── CreditsPage.tsx
│       │   ├── HealthPage.tsx
│       │   ├── NotificationsPage.tsx
│       │   ├── PlaceholderPage.tsx
│       │   ├── RatesPage.tsx
│       │   ├── RegisterPage.tsx
│       │   └── TransfersPage.tsx
│       ├── types
│       │   ├── account.ts
│       │   ├── admin.ts
│       │   ├── analytics.ts
│       │   ├── auth.ts
│       │   ├── card.ts
│       │   ├── common.ts
│       │   ├── credit.ts
│       │   ├── notification.ts
│       │   ├── rate.ts
│       │   └── transfer.ts
│       ├── utils
│       │   └── format.ts
│       ├── App.css
│       ├── App.tsx
│       ├── index.css
│       ├── main.tsx
│       └── vite-env.d.ts
├── README.md
└── test.http
```

## Переменные окружения

Создайте `.env` по примеру `.env.example`.

```env
SERVER_PORT=8080

NGINX_HTTP_PORT=80
POSTGRES_PASSWORD=change_me

DATABASE_URL=postgres://bank_user:change_me@localhost:5432/bank_service?sslmode=disable

JWT_SECRET=change_me_long_random_secret

CARD_PGP_KEY=change_me_card_pgp_key
CARD_HMAC_SECRET=change_me_card_hmac_secret

CBR_URL=https://www.cbr.ru/DailyInfoWebServ/DailyInfo.asmx

SMTP_ENABLED=false
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=noreply@example.com
SMTP_PASSWORD=change_me
SMTP_FROM=noreply@example.com

RATE_LIMIT_ENABLED=true
IDEMPOTENCY_REQUIRED=false

IDEMPOTENCY_RETENTION_SECONDS=86400
IDEMPOTENCY_CLEANUP_INTERVAL_SECONDS=3600

CREDIT_POLICY_ENABLED=true
CREDIT_MAX_ACTIVE_CREDITS=3
CREDIT_MAX_PRINCIPAL_AMOUNT=1000000.00
CREDIT_MAX_TOTAL_PRINCIPAL_AMOUNT=3000000.00
CREDIT_MAX_DEBT_LOAD_PERCENT=50
```

`.env` нельзя коммитить в репозиторий.

## Подготовка базы данных

Создайте базу данных PostgreSQL и пользователя.

```sql
CREATE DATABASE bank_service;
CREATE USER bank_user WITH PASSWORD 'change_me';
GRANT ALL PRIVILEGES ON DATABASE bank_service TO bank_user;
```

Подключитесь к базе и примените миграцию:

```sql
\i migrations/001_init.sql
```

Или выполните содержимое файла `migrations/001_init.sql` через DBeaver.

Миграция создает:

- `pgcrypto`;
- `users`;
- `accounts`;
- `cards`;
- `transactions`;
- `credits`;
- `payment_schedules`;
- `mfa_codes`;
- `revoked_tokens`;
- `user_sessions`;
- `idempotency_keys`;
- `audit_logs`.

## Запуск

Проверьте зависимости:

```bash
go mod tidy
```

Запустите сервер:

```bash
go run ./cmd/server
```

Проверка health endpoint:

```http
GET http://localhost:8080/health
```

## Запуск через Docker Compose

Для серверного запуска используется `docker-compose.yml`.

```bash
docker compose up -d --build
```

В compose поднимаются:

```
bank-api — Go API;
bank-postgres — PostgreSQL;
bank-nginx — nginx reverse proxy, стартовая страница и React frontend.
```

Проверка:

```
docker compose ps
docker compose logs -f bank-api
docker compose logs -f bank-nginx
```

После запуска через nginx доступны адреса:

```text
/        стартовая страница API
/app/    React frontend
/api/... Go API
/docs    ссылка на README
```
Для `test.http` на сервере используйте:

```http
@baseUrl = http://18.185.7.63/api
```

### Проверка качества кода

Перед сдачей проект проверялся следующими командами:

```bash
gofmt -w ./cmd ./internal
go mod tidy
golangci-lint run ./...
govulncheck ./...
```

`golangci-lint` настроен в `.golangci.yml`.
Для проверки уязвимостей используется `govulncheck`.

## Frontend

Frontend находится в папке `web`.

Технологии:

- React;
- TypeScript;
- Vite;
- Tailwind CSS.

Frontend реализует ручную проверку основных сценариев:

- health check;
- register/login/logout;
- просмотр текущего пользователя и роли;
- admin actions;
- счета;
- карты;
- переводы;
- кредиты по выбранному счету;
- аналитика;
- ставки;
- SMTP test email.

Локальный запуск frontend:

```bash
cd web
npm install
npm run dev
```

Сборка frontend:

```bash
cd web
npm run build
```

При Docker Compose сборка frontend выполняется внутри nginx-образа.

Локально frontend доступен по адресу:

```
http://localhost:5173
```

На сервере frontend доступен через nginx:

```
http://18.185.7.63/app/
```

## Основные endpoints

### Public

```text
POST /register
POST /login
GET  /health
```

### Auth

```text
GET  /auth/check
POST /logout
POST /mfa/request
```

### Accounts

```text
POST /accounts
GET  /accounts
GET  /accounts/{accountId}
POST /accounts/{accountId}/deposit
POST /accounts/{accountId}/withdraw
POST /accounts/{accountId}/close
GET  /accounts/{accountId}/predict?days=N
```

### Transfers

```text
POST /transfer
```

### Cards

```text
POST /cards
GET  /cards
GET  /cards/{cardId}
POST /cards/{cardId}/pay
POST /cards/{cardId}/transfer
POST /cards/{cardId}/close
```

### Credits

```text
POST /credits/check
POST /credits
GET  /credits
GET  /credits/{creditId}
GET  /credits/{creditId}/schedule
```

### Analytics

```text
GET /analytics
GET /accounts/{accountId}/predict?days=N
```

### Rates

```text
GET /rates/key
```

### Notifications

```text
GET /notifications/test
```

### Admin

```text
GET  /admin/users
GET  /admin/logged-in-users
POST /admin/accounts/{accountId}/block
POST /admin/accounts/{accountId}/unblock
```

Первый зарегистрированный пользователь автоматически становится администратором.

## MFA

Для критических операций сначала запрашивается MFA-код:

```http
POST /mfa/request
Authorization: Bearer <token>
Content-Type: application/json

{
  "purpose": "transfer",
  "from_account_id": 1,
  "to_account_id": 2,
  "amount": "100.00"
}
```

Поддерживаемые purpose:

```text
withdraw
transfer
card_payment
card_transfer
credit_create
```

Код действует 5 минут и привязан к конкретной операции: типу операции, счетам/картам и сумме.

## Idempotency-Key

Для финансовых POST-запросов используется заголовок:

```http
Idempotency-Key: <uuid>
```

Применяется к:

```text
POST /accounts/{accountId}/deposit
POST /accounts/{accountId}/withdraw
POST /accounts/{accountId}/close
POST /transfer
POST /cards/{cardId}/pay
POST /cards/{cardId}/transfer
POST /cards/{cardId}/close
POST /credits
```

Один и тот же ключ с тем же body не выполнит операцию повторно. Один и тот же ключ с другим body вернет ошибку.

## Кредитная политика

Перед созданием кредита проверяется:

- нет ли overdue-кредита;
- не превышен ли лимит активных кредитов;
- не превышена ли сумма одного кредита;
- не превышена ли общая сумма активных кредитов;
- не превышена ли долговая нагрузка.

Политику можно отключить для отладки:

```env
CREDIT_POLICY_ENABLED=false
```

## Scheduler

При запуске приложения стартуют фоновые задачи:

- списание кредитных платежей каждые 12 часов;
- очистка expired MFA-кодов;
- очистка expired revoked tokens;
- очистка старых idempotency keys.

Для ручной проверки кредитных платежей можно в тестовой БД изменить `payment_schedules.payment_date` на `CURRENT_DATE` и перезапустить сервер.

## Тестирование

В проекте есть файл `test.http` для ручной проверки через REST Client в VS Code.

Рекомендуемый порядок:

1. `GET /health`
2. `POST /register`
3. `POST /login`
4. `GET /auth/check`
5. создать счета;
6. выполнить deposit;
7. запросить MFA для withdraw;
8. выполнить withdraw;
9. запросить MFA для transfer;
10. выполнить transfer;
11. выпустить карту;
12. выполнить card payment с MFA;
13. выполнить card transfer с MFA;
14. закрыть карту;
15. проверить кредит через `POST /credits/check`;
16. оформить кредит с MFA;
17. проверить график платежей;
18. проверить analytics;
19. проверить predict по конкретному счету и days;
20. проверить rates;
21. проверить notifications;
22. проверить admin routes;
23. проверить logout и revoked token;
24. проверить повторный login и инвалидирование старого токена.

## Проверка audit logs

```sql
SELECT action, status, user_id, resource_type, resource_id, details, created_at
FROM audit_logs
ORDER BY id DESC
LIMIT 30;
```

## Проверка idempotency

```sql
SELECT id, user_id, method, path, key, request_hash, created_at
FROM idempotency_keys
ORDER BY id DESC
LIMIT 20;
```

## Ограничения

- Поддерживается только валюта RUB.
- Максимальный период прогноза баланса — 365 дней.
- Закрытая карта не открывается повторно. Если нужна новая карта, пользователь выпускает новую.
- Закрытый счет не открывается повторно. Счет остается в истории, но финансовые операции по нему запрещены.
- Защита от volumetric DDoS должна выполняться на уровне reverse proxy или cloud provider.
- Тестовый сервер доступен по HTTP. Для production нужен домен и HTTPS.
