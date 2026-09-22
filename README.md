# High-Performance URL Shortener in Go

API built with Go, designed for low-latency redirects, caching, rate limiting, authentication, and analytics.

## Features

- **URL Management** — Create, read, update, delete, and list short URLs with Base62-generated codes or custom aliases.
- **Expiration** — Configurable link expiry with automatic 410 Gone responses.
- **Authentication** — User registration/login with bcrypt password hashing and JWT-based authentication.
- **Authorization** — Users can only modify or delete their own URLs.
- **Guest Support** — Anonymous users can create short URLs with stricter rate limits.
- **Redis Caching** — Cache-aside URL lookups with automatic cache invalidation and TTL management.
- **Rate Limiting** — Redis-based sliding-window rate limiting using Lua scripts.
- **Click Analytics** — Asynchronous click tracking with referrer, user-agent, IP, timestamps, and aggregated analytics.

 ## Tech Stack

 - Go 1.27+
- Chi v5 — HTTP router
- PostgreSQL 18 — Primary database
- pgx/v5 — PostgreSQL driver
- Redis 8 — Caching & rate limiting
- bcrypt — Password hashing
- JWT v5 — Authentication

 ## Quick Start

 Start PostgreSQL and Redis:

```
docker compose up -d
```

 Configure environment variables:

```
cp .env.example .env
```

 Run the server:

```
go run cmd/server/main.go
```

 The API will be available at http://localhost:8080.

 ## API Overview

 ### Authentication

```
POST /api/v1/auth/register
POST /api/v1/auth/login
GET  /api/v1/auth/me
```

 ### URLs

```
POST   /api/v1/urls
GET    /api/v1/urls/{code}
GET    /api/v1/urls
PATCH  /api/v1/urls/{code}
DELETE /api/v1/urls/{code}
GET    /api/v1/urls/{code}/analytics
```

 ### Redirect

```
GET /{code}
```

 Returns a 302 Found redirect to the original URL.

_Built as a practice project to explore Go backend development using chi and net/http packages, Redis caching, PostgreSQL, JWT authentication_
