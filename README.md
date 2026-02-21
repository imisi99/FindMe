# FindMe — Backend Service

> A platform for developers to discover collaborators and contribute to projects, powered by AI-driven recommendations and vector embeddings.

---

## Overview

[FindMe](https://findmeapi.duckdns.org/swagger/index.html) is a microservices-based application that connects developers with projects and other developers based on their skills, interests, and bios. This repository contains the **core backend service** — a REST API built in Go that handles user management, project listings, messaging, subscriptions, and coordinates with two supporting services via gRPC:

- **[Embedding Service](https://github.com/imisi99/FindMeML)** (`emb`) — generates and maintains vector embeddings for users and projects
- **[Recommendation Service](https://github.com/imisi99/FindMeMLR)** (`rec`) — returns ranked recommendations of users/projects using vector similarity

---

## Architecture

```
┌─────────────────────────────────┐
│        FindMe Backend (Go)      │
└────┬────────────────────────────┘
     │ gRPC                  │ gRPC
┌────▼──────┐         ┌──────▼───────┐
│ Embedding │         │Recommendation│
│  Service  │         │   Service    │
│ (python)  │         │   (python)   │
└───────────┘         └──────────────┘
```

Communication with both gRPC services is handled asynchronously via in-process job queues (worker pool pattern), with automatic retry logic on failures.

---

## Features

- **Authentication** — JWT-based auth with email/password and GitHub OAuth
- **User Profiles** — skills, interests, bio, and availability management
- **Projects** — create, edit, search, bookmark, and apply to collaborative projects
- **GitHub integration** — links a user github repository for richer recommendations
- **AI Recommendations** — user and project recommendations via the recommendation service
- **Vector Embeddings** — user and project embeddings are automatically kept in sync via the embedding service whenever profile data changes
- **Real-time Messaging** — WebSocket-powered chat with support for direct messages with friends and projects group chats
- **Subscriptions & Payments** — Paystack integration for subscription plans, webhooks, card management, and retry logic
- **Email Notifications** — async email queue for friend requests, project applications, subscription events, and free trial reminders
- **Cron Jobs** — daily trial-ending reminder emails
- **Swagger UI** — auto-generated API documentation available at `/swagger/index.html`
- **Health Checks** — detailed health endpoint covering database and Redis connectivity

---

## Tech Stack

| Concern | Technology |
|---|---|
| Language | Go 1.24 |
| HTTP Framework | Gin |
| ORM | GORM (PostgreSQL driver) |
| Cache | Redis (`go-redis`) |
| Real-time | WebSockets (`gorilla/websocket`) |
| Inter-service | gRPC + Protocol Buffers |
| Auth | JWT (`golang-jwt/jwt`) |
| Payments | Paystack |
| Email | SMTP via `go-mail` |
| Docs | Swaggo (Swagger) |
| Containerization | Docker + Docker Compose |
| Reverse Proxy | Nginx + Let's Encrypt (Certbot) |
| Testing | `testify` |

---

## Project Structure

```
.
├── core/           # Logic interfaces and implementations
│   ├── cache.go    # Redis cache layer
│   ├── conf.go     # Utilities (hashing, OTP, username generation)
│   ├── cron.go     # Scheduled jobs (trial ending reminders)
│   ├── database.go # DB interface + GORM implementation
│   ├── email.go    # Async email worker and queue
│   ├── emb.go      # Embedding service gRPC worker pool
│   ├── msg.go      # Chat hub (WebSocket connection manager)
│   └── rec.go      # Recommendation service gRPC worker pool
├── database/       # DB and Redis connection setup
├── docs/           # Auto-generated Swagger docs
├── emb/            # Generated gRPC code for embedding service
├── handlers/       # HTTP handler implementations
│   ├── conf.go     # Handler wiring and shared utilities
│   ├── github.go   # GitHub OAuth handler
│   ├── handler.go  # Route registration
│   ├── health.go   # Health check endpoint
│   ├── msg.go      # Chat and messaging handlers
│   ├── post.go     # Project handlers
│   ├── realtime.go # WebSocket upgrade handler
│   ├── transc.go   # Payment and subscription handlers
│   └── user.go     # User profile handlers
├── model/          # GORM data models
├── nginx/          # Nginx config (HTTP + HTTPS + rate limiting)
├── proto/          # Protobuf definitions for emb and rec services
├── rec/            # Generated gRPC code for recommendation service
├── schema/         # Request/response schemas (Gin binding + Swagger)
├── test/           # Unit tests with mocks
├── docker-compose.yml
├── Dockerfile
├── generate.sh     # Proto code generation script
└── main.go
```

---

## Getting Started

### Prerequisites

- Go 1.24+
- Docker & Docker Compose
- `protoc` and `protoc-gen-go` / `protoc-gen-go-grpc` (for proto regeneration only)
- The embedding and recommendation services running and accessible

### Environment Variables

Create a `.env` file in the project root:

```env
# Database
POSTGRES_USER=your_db_user
POSTGRES_PASSWORD=your_db_password
POSTGRES_DB=findme
POSTGRES_HOST=db

# Redis
REDIS_PASS=your_redis_password
REDIS_ADDR=redis:6379

# Auth
JWTSECRET=your_jwt_secret

# GitHub OAuth
GIT_CLIENT_ID=your_github_client_id
GIT_CLIENT_SECRET=your_github_client_secret
GIT_CALLBACK_URL=https://yourdomain.com/api/user/github/callback

# Email (Gmail SMTP)
EMAIL=your_email@gmail.com
EMAIL_APP_PASSWORD=your_gmail_app_password

# Payments
PAYSTACK_API_KEY=your_paystack_secret_key
```

### Running with Docker Compose

```bash
# Create the shared network (only needed once)
docker network create findme-shared-network

# Start all services
docker compose up -d --build
```

This will start PostgreSQL, Redis, the backend app, Nginx, and Certbot. The embedding and recommendation services are expected to be reachable at `emb:8000` and `rec:8050` on the same Docker network.

### Running Locally

```bash
go mod download
go run .
```

The API will be available at `http://localhost:8080`.

### Regenerating Proto Files

If you modify the `.proto` definitions, regenerate the Go code with:

```bash
chmod +x generate.sh
./generate.sh
```

---

## API Documentation

Swagger UI is available at:

```
http://localhost:8080/swagger/index.html
```

### API Groups

| Tag | Description |
|---|---|
| `User` | Registration, login, profile management, skills, friends, GitHub OAuth |
| `Post` | Create, edit, search, bookmark, and apply to projects |
| `Msg` | Chat creation, messaging, group chat management |
| `Transaction` | Paystack payment init, subscription management, webhooks |
| `Health` | Service health check |

All authenticated endpoints require a `Bearer <token>` header.

---

## Testing

```bash
go test ./test/unit/...
```

The test suite uses mocks for the DB, cache, email, embedding, and recommendation interfaces to allow unit testing without external dependencies.

---

## Microservices

This service communicates with two companion services over gRPC. Both services should be deployed on the same Docker network.

### [Embedding Service](https://github.com/imisi99/FindMeML) (`emb:8000`)

Maintains vector representations of users and projects. Called automatically when:

- A user registers, updates their bio/skills/interests, changes availability, or is deleted
- A project is created, updated, changes availability, or is deleted

### [Recommendation Service](https://github.com/imisi99/FindMeMLR) (`rec:8050`)

Returns ranked similarity scores for user and project recommendations. Called when:

- A user requests recommended projects (matched by their profile vector)
- A user requests recommended collaborators (matched by a project's vector)

---

## License

See [LICENSE](./LICENSE).
