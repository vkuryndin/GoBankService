# Bank Service REST API

REST API банковского сервиса на Go. Проект реализует регистрацию пользователей, JWT-аутентификацию, счета, карты, переводы, кредиты, аналитику, интеграцию с ЦБ РФ и SMTP-уведомления.

## Стек

- Go 1.23+
- gorilla/mux
- PostgreSQL + lib/pq
- pgcrypto
- JWT
- bcrypt
- HMAC-SHA256
- logrus
- gomail
- beevik/etree

## Основные возможности

- Регистрация и вход пользователя.
- JWT с временем жизни 24 часа.
- Повторный login инвалидирует старые активные токены пользователя.
- Logout и revoked tokens.
- Создание и просмотр счетов.
- Пополнение счета.
- Списание со счета с MFA.
- Переводы между счетами с MFA.
- Выпуск виртуальных карт.
- Просмотр своих карт.
- Оплата картой с CVV и MFA.
- Закрытие карты через soft close.
- Оформление кредита с MFA.
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

## Безопасность

В проекте реализованы:

- bcrypt-хеширование паролей;
- bcrypt-хеширование CVV;
- PGP-шифрование номера и срока карты через pgcrypto;
- HMAC-SHA256 для проверки целостности данных карты;
- JWT-аутентификация;
- JWT `jti`;
- single-session login;
- revoked tokens;
- проверка владельца счетов, карт и кредитов;
- MFA для критических операций: withdraw, transfer, card payment, credit create;
- ограничение попыток MFA и CVV;
- Idempotency-Key для финансовых POST-операций;
- request_hash для защиты от переиспользования Idempotency-Key с другим body;
- rate limiting;
- ограничение размера request body;
- HTTP server timeouts;
- security headers;
- request id;
- strict JSON decoding;
- audit logs;
- DB CHECK constraints;
- graceful shutdown.

## Структура проекта

```text
cmd/server              запуск приложения
internal/config         конфигурация
internal/db             подключение к PostgreSQL
internal/models         модели БД
internal/dto            DTO запросов и ответов
internal/repositories   SQL-запросы и транзакции
internal/services       бизнес-логика
internal/handlers       HTTP handlers
internal/middleware     middleware
internal/router         маршрутизация
internal/security       JWT, bcrypt, карты, HMAC
internal/integrations   внешние интеграции
internal/scheduler      фоновые задачи
internal/audit          audit events
migrations              SQL-миграции
```

## Переменные окружения

Создайте `.env` по примеру `.env.example`.

```env
SERVER_PORT=8080

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

Установите зависимости:

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
GET  /accounts/{accountId}/predict
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
POST /cards/{cardId}/close
```

### Credits

```text
POST /credits
GET  /credits
GET  /credits/{creditId}
GET  /credits/{creditId}/schedule
```

### Analytics and rates

```text
GET /analytics
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
credit_create
```

Код действует 5 минут и привязан к конкретной операции.

## Idempotency-Key

Для финансовых POST-запросов используется заголовок:

```http
Idempotency-Key: <uuid>
```

Применяется к:

```text
POST /accounts/{accountId}/deposit
POST /accounts/{accountId}/withdraw
POST /transfer
POST /cards/{cardId}/pay
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
4. создать счета;
5. выполнить deposit;
6. запросить MFA для withdraw;
7. выполнить withdraw;
8. запросить MFA для transfer;
9. выполнить transfer;
10. выпустить карту;
11. выполнить card payment с MFA;
12. закрыть карту;
13. оформить кредит с MFA;
14. проверить график платежей;
15. проверить analytics и predict;
16. проверить admin routes;
17. проверить logout и revoked token.

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
- Защита от volumetric DDoS должна выполняться на уровне reverse proxy или cloud provider.
