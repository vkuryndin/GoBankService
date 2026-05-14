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
- React 19
- TypeScript
- Vite
- Tailwind CSS
- ESLint
- @tanstack/react-query
- react-router-dom
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
- Защита CBR-интеграции через cache fallback и circuit breaker.
- SMTP-уведомления.
- Admin API для просмотра пользователей, активных сессий и блокировки счетов.
- Backend hardening: timeouts, graceful shutdown, structured logging, CORS и gzip через nginx.
- React frontend с TypeScript и Vite для ручной проверки всех основных сценариев.
- Кросс-вкладочная синхронизация сессий в frontend.
- Обработка ошибок через ErrorBoundary и ToastProvider.
- Разделение публичного и приватного layout в frontend.
- Защищённые маршруты с проверкой ролей.
- Кеширование API-запросов через React Query.
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
- cache revoked/inactive tokens для снижения нагрузки на БД;
- проверка владельца счетов, карт и кредитов;
- MFA для критических операций: withdraw, transfer, card payment, card transfer, credit create;
- MFA-код атомарно помечается использованным, чтобы исключить повторное применение одного кода при параллельных запросах;
- ограничение попыток MFA и CVV;
- cooldown/rate limit на запрос MFA-кода;
- Idempotency-Key для финансовых POST-операций;
- request_hash для защиты от переиспользования Idempotency-Key с другим body;
- rate limiting;
- ограничение размера request body;
- HTTP server timeouts;
- request context timeout для операций backend → DB;
- security headers;
- request id;
- strict JSON decoding;
- дополнительная backend-валидация входных данных;
- strict parsing конфигурации: ошибки в bool/int env-переменных не подменяются молча default-значениями;
- audit logs;
- DB CHECK constraints;
- graceful shutdown;
- корректное завершение scheduler-ов при shutdown;
- JSON structured logging;
- configurable CORS;
- gzip compression через nginx;
- CBR circuit breaker для защиты от зависаний внешнего сервиса;
- явные проверки active/status для финансовых операций;
- кросс-вкладочная синхронизация сессий в frontend через localStorage events;
- обработка ошибок через ErrorBoundary;
- toast notifications для пользовательских сообщений;
- axios interceptors для автоматической подстановки токенов и обработки 401;
- кеширование ключевой ставки с TTL;
- очистка таймеров в frontend-компонентах при unmount.

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
nginx                   nginx reverse proxy, gzip compression и стартовая страница
web                     React frontend с TypeScript, Vite, Tailwind CSS
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
│   ├── Dockerfile
│   ├── eslint.config.js
│   ├── index.html
│   ├── package.json
│   ├── package-lock.json
│   ├── public
│   ├── src
│   │   ├── api
│   │   ├── App.css
│   │   ├── App.tsx
│   │   ├── assets
│   │   ├── components
│   │   ├── config
│   │   ├── contexts
│   │   ├── features
│   │   ├── hooks
│   │   ├── index.css
│   │   ├── layout
│   │   ├── main.tsx
│   │   ├── pages
│   │   ├── routes
│   │   ├── styles
│   │   ├── types
│   │   └── utils
│   ├── tsconfig.app.json
│   ├── tsconfig.json
│   ├── tsconfig.node.json
│   └── vite.config.ts
├── README.md
└── test.http
```

## Переменные окружения

Создайте `.env` по примеру `.env.example`.

```env
SERVER_PORT=8080
LOG_FORMAT=json

NGINX_HTTP_PORT=80
POSTGRES_PASSWORD=change_me

DATABASE_URL=postgres://bank_user:change_me@localhost:5432/bank_service?sslmode=disable

JWT_SECRET=change_me_long_random_secret

CARD_PGP_KEY=change_me_card_pgp_key
CARD_HMAC_SECRET=change_me_card_hmac_secret

CBR_URL=https://www.cbr.ru/DailyInfoWebServ/DailyInfo.asmx
CBR_CACHE_TTL_SECONDS=3600
CBR_BREAKER_FAILURE_LIMIT=3
CBR_BREAKER_RESET_TIMEOUT_SECONDS=60

SMTP_ENABLED=false
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=noreply@example.com
SMTP_PASSWORD=change_me
SMTP_FROM=noreply@example.com

SERVER_REQUEST_TIMEOUT_SECONDS=20
SERVER_READ_HEADER_TIMEOUT_SECONDS=5
SERVER_READ_TIMEOUT_SECONDS=15
SERVER_WRITE_TIMEOUT_SECONDS=30
SERVER_IDLE_TIMEOUT_SECONDS=60
SERVER_MAX_HEADER_BYTES=1048576

SECURITY_MAX_REQUEST_BODY_BYTES=1048576
TOKEN_REVOCATION_CACHE_TTL_SECONDS=5

CORS_ENABLED=false
CORS_ALLOWED_ORIGINS=http://localhost:5173
CORS_ALLOWED_METHODS=GET,POST,OPTIONS
CORS_ALLOWED_HEADERS=Authorization,Content-Type,Idempotency-Key,X-Request-ID
CORS_ALLOW_CREDENTIALS=false
CORS_MAX_AGE_SECONDS=600

RATE_LIMIT_ENABLED=true
RATE_LIMIT_CLEANUP_INTERVAL_SECONDS=300
RATE_LIMIT_GLOBAL_REQUESTS=1000
RATE_LIMIT_GLOBAL_WINDOW_SECONDS=60
RATE_LIMIT_LOGIN_REQUESTS=30
RATE_LIMIT_LOGIN_WINDOW_SECONDS=60
RATE_LIMIT_REGISTER_REQUESTS=30
RATE_LIMIT_REGISTER_WINDOW_SECONDS=60
RATE_LIMIT_MFA_REQUESTS=30
RATE_LIMIT_MFA_WINDOW_SECONDS=300
RATE_LIMIT_FINANCIAL_REQUESTS=120
RATE_LIMIT_FINANCIAL_WINDOW_SECONDS=60
RATE_LIMIT_ADMIN_REQUESTS=120
RATE_LIMIT_ADMIN_WINDOW_SECONDS=60
RATE_LIMIT_RATE_REQUESTS=120
RATE_LIMIT_RATE_WINDOW_SECONDS=60

MFA_MAX_FAILED_ATTEMPTS=5
MFA_LOCKOUT_SECONDS=600
MFA_REQUEST_COOLDOWN_SECONDS=60

CVV_MAX_FAILED_ATTEMPTS=5
CVV_LOCKOUT_SECONDS=600

IDEMPOTENCY_ENABLED=true
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
bank-nginx — nginx reverse proxy, gzip compression, стартовая страница и React frontend.
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

### Локальный запуск frontend

Для разработки frontend локально:

```bash
cd web
npm install
npm run dev
```

Frontend будет доступен по адресу:

```
http://localhost:5173
```

Dev server проксирует API-запросы на `/api` к backend по адресу `http://localhost:8080`.

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

- React 19;
- TypeScript;
- Vite;
- Tailwind CSS;
- ESLint;
- @tanstack/react-query;
- react-router-dom.

Frontend реализует ручную проверку основных сценариев API через React-приложение с:

- Аутентификацией и авторизацией (JWT, MFA);
- Управлением состоянием через React Context (AuthContext, SharedAccountContext);
- Защищёнными маршрутами (ProtectedRoute, AdminRoute);
- Разделением layout (PublicLayout для входа, AppLayout для приложения);
- Обработкой ошибок (ErrorBoundary, ToastProvider);
- Кросс-вкладочной синхронизацией сессий;
- Кешированием данных через React Query;
- Многоуровневой архитектурой компонентов (pages, features, components, ui).

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

## Backend hardening

В backend добавлены дополнительные меры устойчивости:

- финансовые операции выполняются с усиленными транзакционными и status-проверками;
- revoked/inactive tokens кешируются, чтобы снизить количество запросов к БД на защищенных маршрутах;
- DB-запросы ограничены request context timeout;
- scheduler-ы используют application context и завершаются при graceful shutdown;
- CORS настраивается через переменные окружения;
- nginx включает gzip compression;
- CBR-интеграция защищена circuit breaker-ом и cache fallback;
- MFA request имеет cooldown/rate limit;
- логи могут выводиться в JSON-формате;
- входные данные дополнительно валидируются на backend-е.

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

Scheduler-ы запускаются с application context и корректно завершаются при graceful shutdown.

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

## Backup и rollback

Перед рискованным deploy рекомендуется сделать backup БД:

```bash
docker exec bank-postgres pg_dump -U bank_user bank_service > backup.sql
```

### Восстановление из backup

```bash
cat backup.sql | docker exec -i bank-postgres psql -U bank_user -d bank_service
```

Без необходимости не выполнять команды, которые удаляют Docker volumes:

```bash
docker compose down -v
docker volume rm <volume_name>
docker system prune --volumes
```

### Rollback приложения

```
git log --oneline
git checkout <previous_commit>
docker compose build --no-cache --progress=plain bank-api bank-nginx
docker compose up -d --remove-orphans
```

Если миграции БД не менялись, rollback проще: достаточно откатить код и пересобрать контейнеры.


## Ограничения

- Поддерживается только валюта RUB.
- Максимальный период прогноза баланса — 365 дней.
- Закрытая карта не открывается повторно. Если нужна новая карта, пользователь выпускает новую.
- Закрытый счет не открывается повторно. Счет остается в истории, но финансовые операции по нему запрещены.
- Защита от volumetric DDoS должна выполняться на уровне reverse proxy или cloud provider.
- Тестовый сервер доступен по HTTP. Для production нужен домен и HTTPS.
- In-memory rate limit и token cache рассчитаны на один экземпляр backend. Для нескольких экземпляров нужен Redis или другой общий storage.
