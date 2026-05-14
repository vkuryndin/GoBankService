# Bank Service Web

Frontend for the Go bank-service REST API.

## Stack

- React
- TypeScript
- Vite
- TanStack React Query
- Axios-compatible API wrapper through `fetch`
- React Router
- Plain CSS files by page/feature

## Local start

Install dependencies:

```bash
npm ci
```

Start the frontend dev server:

```bash
npm run dev
```

By default Vite starts on:

```text
http://localhost:5173
```

## Backend API contract

The frontend sends requests to API paths with the `/api` prefix:

```text
/api/health
/api/login
/api/accounts
/api/cards
/api/credits
```

In the deployed stand, nginx maps `/api/...` to the Go backend root routes.

Recommended local options:

1. Run the full docker/nginx stack and open `/app/`.
2. Or configure a Vite proxy for `/api` if you run the Go backend directly on `localhost:8080`.

## Implemented pages

- Health
- Register
- Login/logout
- Accounts
- Cards
- Transfers
- Credits
- Analytics
- Rates
- Notifications
- Admin

## Cards security flow

A normal card details request does not expose the full card number.

Safe card details:

```http
GET /api/cards/{cardId}
```

Full card number reveal requires MFA:

```http
POST /api/mfa/request
```

with body:

```json
{
  "purpose": "card_reveal",
  "card_id": 1
}
```

Then reveal:

```http
POST /api/cards/{cardId}/reveal
```

with body:

```json
{
  "mfa_code": "123456"
}
```

## Build

```bash
npm run build
```

The production build is written to:

```text
dist/
```

## Important repository hygiene

Do not commit or archive:

```text
node_modules/
dist/
```

Use `npm ci` to restore dependencies from `package-lock.json`.
