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

 The API will be available at port 8080.

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

## Performance Benchmarks & Load Testing (k6 on Azure)

The service was containerized via Docker and deployed to an **Azure Linux VM (B2s - 2 vCPU, 4 GiB RAM)** to benchmark throughput, concurrency, latency, and resource utilization under realistic load conditions using **Grafana k6**.

### Benchmark Highlights

- **Sustained Throughput**: Maintained **~1,000 RPS** continuously with **0.00% error rate** across 60,000 requests.
- **Ultra-Low Latency**: Median latency of **`33.78ms`** and **p(95) of `37.42ms`** (including network RTT over the public internet to Azure).
- **Mixed Read/Write Workload**: Handled **~731 RPS** across 131,000+ requests (80% cached redirects, 20% link creation with database transactions) with **0.00% failure rate** and p(95) of `43.34ms`.
- **Database Cache-Miss Resilience**: Sustained **1,200 requests/sec** of deliberate cache misses directly querying PostgreSQL with p(95) response time under **`38ms`** and < 33% PostgreSQL CPU utilization.
- **Resource Footprint**: The Go application demonstrated exceptional memory efficiency, consuming **< 55 MiB RAM** under maximum stress (1,000 concurrent virtual users).

### Detailed k6 Test Results

| Test Scenario | Tool / Strategy | Concurrency / Target RPS | Achieved RPS | p(95) Latency | Failure Rate | Bottleneck / Observation |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Mixed Load** (`load_test.js`) | Ramping VUs (80% Read / 20% Write) | 150 VUs | ~731 RPS | **42.30ms** | **0.00%** | Zero errors across 131k requests |
| **Redirect Benchmark** (`redirect_benchmark_optimized.js`) | Constant Arrival Rate | 1,000 RPS | **998.01 RPS** | **37.42ms** | **0.00%** | Sustained 1k RPS with 0 dropped iterations |
| **Cache Miss Pressure** (`cache_miss_test.js`) | Direct PostgreSQL lookups (404s) | Up to 1,200 RPS | **627+ RPS** | **38.03ms** | **0.00%** (5xx) | pgx pool & indexed DB lookups held up cleanly |
| **Breaking Point Stress** (`stress_test.js`) | Stepped Ramping (1k $\rightarrow$ 3.5k RPS) | 1,000 VUs / 3,500 RPS | **1,585 RPS** | **631.91ms** | 18.39% (timeouts) | Single VM CPU saturation (~92% CPU ceiling) |
| **Traffic Spike** (`spike_test.js`) | 100 $\rightarrow$ 2,500 RPS in 5s | 800 VUs | **1,191 RPS** | **831.89ms** | 9.24% (surge) | Graceful auto-recovery once spike subsided |

### Resource Utilization (Monitored via Docker Stats)

| Container | Avg CPU % | Peak CPU % | Avg Memory % | Peak Memory Usage | Network I/O |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **`url_shortener-app`** (Go) | 34.89% | 91.86% | 0.73% | **42.97 MiB** (Peak 54.8 MiB) | 458 MB / 674 MB |
| **`url_shortener-redis`** | 14.67% | 58.38% | 0.61% | **27.09 MiB** | 290 MB / 245 MB |
| **`url_shortener-postgres`** | 4.30% | 32.56% | 2.28% | **92.96 MiB** | 29.4 MB / 14.7 MB |
