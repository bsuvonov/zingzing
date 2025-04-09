<div align="center">
  <img src="./assets/logo.png" alt="ZingZing-logo" width="400" />
</div>
<h1 align="center">ZingZing</h1>


<p align="center">ZingZing is a lightweight social media API web server built with Go and PostgreSQL, inspired by Twitter.</p>

## 🌟 Features

- **User Authentication:** Secure signup, login, and JWT token management.
- **Secure Endpoints:** Protect endpoints with JWT authentication and refresh tokens.
- **Zinger Management:** Post, retrieve, filter, sort, and delete zingers (tweets).
- **Webhook Integration:** Upgrade user statuses securely via webhook events from third-party services.
- **Refresh Tokens:** Issue and revoke refresh tokens securely stored in PostgreSQL.
- **Advanced Queries:** Sort and filter zingers by creation date and author ID.
- **Admin Metrics:** Monitor and reset application metrics (development only).

## 🛠 Tech Stack

- **Go (1.24+)**
- **PostgreSQL**
- **JWT (golang-jwt)**
- **bcrypt** for password hashing
- **Goose** for database migrations
- **SQLC** for type-safe queries

## 📚 Setup & Installation

### Step 1: Clone the repository

```bash
git clone https://github.com/bsuvonov/zingzing.git
cd zingzing
```

### Step 2: Set up environment variables

Generate a strong secret for JWT:

```bash
openssl rand -base64 64
```

Create a `.env` file in your project's root, and put the relevant credentials to their respective places:

```env
DB_URL=your_postgres_db_url
JWT_SECRET=your_jwt_secret
ZINGPAY_KEY=your_zingpay_api_key
```

Zingpay is used to demonstrate webhooks and isn't a real provider so use any generated API key in the env)

### Step 3: Run database migrations

```bash
goose postgres "$DB_URL" --dir sql/schema up
```

### Step 4: Install dependencies and run

```bash
go mod tidy
go build -o out && ./out
```

Server runs on `http://localhost:8080`

## 🔑 API Endpoints

### Users

- `POST /api/users` - Create user
- `PUT /api/users` - Update user's email/password
- `POST /api/login` - User login, returns JWT & refresh token
- `POST /api/refresh` - Refresh JWT using refresh token
- `POST /api/revoke` - Revoke refresh token

### Zingers

- `POST /api/zingers` - Post zinger
- `GET /api/zingers` - Retrieve all zingers (supports filtering and sorting)
- `GET /api/zingers/{zingerID}` - Retrieve zinger by ID
- `DELETE /api/zingers/{zingerID}` - Delete zinger by ID (authenticated)

### Webhooks

- `POST /api/polka/webhooks` - Upgrade user to premium (requires API key)

### Admin

- `GET /admin/metrics` - Display metrics
- `POST /admin/reset` - Reset metrics and delete all users
