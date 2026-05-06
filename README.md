# Bank Service

REST API банковского сервиса на Go.

## Что реализовано

- регистрация и вход пользователей;
- JWT-аутентификация;
- создание счетов;
- пополнение, списание и переводы;
- выпуск и просмотр виртуальных карт;
- оплата картой;
- оформление кредита;
- график платежей;
- автоматическая обработка кредитных платежей scheduler-ом;
- штраф за просрочку платежа;
- аналитика доходов, расходов и кредитной нагрузки;
- прогноз баланса;
- интеграция с ЦБ РФ для получения ключевой ставки;
- SMTP-уведомления;
- MFA-коды для критических операций;
- admin API для просмотра пользователей, активных сессий и блокировки счетов.

## Стек

- Go 1.23+;
- PostgreSQL;
- gorilla/mux;
- lib/pq;
- golang-jwt/jwt/v5;
- logrus;
- bcrypt;
- pgcrypto;
- gomail;
- beevik/etree.

## Переменные окружения

Создать файл `.env` по примеру `.env.example`.

Основные переменные:

```env
SERVER_PORT=8080
DATABASE_URL=postgres://bank_user:password@localhost:5432/bank_service?sslmode=disable
JWT_SECRET=change_me
CARD_PGP_KEY=change_me
CARD_HMAC_SECRET=change_me
CBR_URL=https://www.cbr.ru/DailyInfoWebServ/DailyInfo.asmx
SMTP_ENABLED=true
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USER=user@example.com
SMTP_PASSWORD=change_me
SMTP_FROM=user@example.com
```

Реальные значения `.env` нельзя коммитить в Git.

## База данных

Создать базу PostgreSQL и применить миграцию:

```bash
psql -d bank_service -f migrations/001_init.sql
```

Миграция создает расширение `pgcrypto` и основные таблицы:

- `users`;
- `accounts`;
- `cards`;
- `transactions`;
- `credits`;
- `payment_schedules`;
- `revoked_tokens`;
- `mfa_codes`;
- `user_sessions`.

## Запуск

```bash
go mod download
go run ./cmd/server
```

Проверка:

```http
GET http://localhost:8080/health
```

## Основные endpoint-ы

Публичные:

- `POST /register`;
- `POST /login`;
- `GET /health`.

Защищенные:

- `GET /auth/check`;
- `POST /logout`;
- `POST /mfa/request`;
- `POST /accounts`;
- `GET /accounts`;
- `GET /accounts/{accountId}`;
- `POST /accounts/{accountId}/deposit`;
- `POST /accounts/{accountId}/withdraw`;
- `POST /transfer`;
- `POST /cards`;
- `GET /cards`;
- `GET /cards/{cardId}`;
- `POST /cards/{cardId}/pay`;
- `GET /rates/key`;
- `POST /credits`;
- `GET /credits`;
- `GET /credits/{creditId}`;
- `GET /credits/{creditId}/schedule`;
- `GET /analytics`;
- `GET /accounts/{accountId}/predict`;
- `GET /notifications/test`.

Admin endpoint-ы:

- `GET /admin/users`;
- `GET /admin/logged-in-users`;
- `POST /admin/accounts/{accountId}/block`;
- `POST /admin/accounts/{accountId}/unblock`.

Первый зарегистрированный пользователь становится администратором.

## Проверка

Для ручной проверки можно использовать `test.http` в VS Code REST Client.

Перед сдачей проверить сценарии:

- регистрация;
- вход;
- проверка JWT;
- logout и revoked token;
- создание счета;
- пополнение;
- списание;
- перевод;
- выпуск карты;
- оплата картой с CVV и MFA;
- создание кредита с MFA;
- получение графика платежей;
- обработка просроченного платежа scheduler-ом;
- аналитика;
- прогноз баланса;
- получение ставки ЦБ РФ;
- тестовое SMTP-письмо;
- admin block/unblock.
