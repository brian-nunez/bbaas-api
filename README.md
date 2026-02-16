
# Browser API Layer (BBAAS)

This includes a browser-management API layer that sits in front of your existing CDP manager service.

- User registration/login with secure session cookies
- Dashboard for application registration and API key management
- Scoped API keys (`READ`, `WRITE`, `DELETE`) for browser lifecycle APIs
- Running/completed browser session tracking per user/application
- Go SDK wrapper (`/sdk/go/bbaas`) for API consumption from Go projects
- Service boundaries/interfaces to support future extension (dashboards, admin tooling, live session views)

## Environment Variables

- `PORT` (default `8080`)
- `CDP_MANAGER_BASE_URL` (default `http://127.0.0.1:8081`)
- `DB_DRIVER` (default `sqlite`, supported: `sqlite`, `postgres`)
- `DB_DSN` (default for sqlite: `file:bbaas.db?_pragma=foreign_keys(1)`)

Note: the `postgres` adapter is wired in the app layer; to run with Postgres, link a Postgres SQL driver in your binary (kept out of the default to minimize dependencies).

## API Endpoints

Base path: `/api/v1`

- `GET /health` (public): health check
- `POST /browsers` (auth): spawn browser
- `GET /browsers` (auth): list browsers for API key's application
- `GET /browsers/:id` (auth): fetch browser details
- `POST /browsers/:id/keepalive` (auth): extend idle timeout
- `DELETE /browsers/:id` (auth): close browser

Authentication:
- `Authorization: Bearer <api_token>` or `X-API-Key: <api_token>`

Web UI flows:
- `GET /register`, `POST /register`
- `GET /login`, `POST /login`, `POST /logout`
- `GET /dashboard`
- `POST /dashboard/applications`
- `POST /dashboard/applications/:applicationId/api-keys`
- `POST /dashboard/applications/:applicationId/api-keys/:keyId/revoke`

## Go SDK Quickstart

Import path:

```go
import "github.com/brian-nunez/bbaas-api/sdk/go/bbaas"
```

Example:

```go
ctx := context.Background()

client, _ := bbaas.NewClient("http://localhost:8080", bbaas.WithAPIToken("bka_..."))
spawned, _ := client.SpawnBrowser(ctx, bbaas.SpawnBrowserRequest{})
fmt.Println(spawned.Browser.CDPURL)
```

---

## Features

- Fast startup, zero config needed
- Opinionated folder structure with `/cmd` and `/internal`
- Templ components using [`templ`](https://templ.guide)
- Utility-first styling with TailwindCSS
- Templ UI support preconfigured (All components are installed)
- Clean routing with Echo (My favorite Go web framework)
- Centralized error handling
- Graceful shutdown
- Makefile-driven dev workflow

---

## Tool Links

- **[Echo](https://echo.labstack.com/)**: A high-performance, minimalist web framework for Go.
- **[Templ](https://templ.guide/)**: A Go HTML templating engine that allows you to build reusable components.
- **[TailwindCSS](https://tailwindcss.com/)**: A utility-first CSS framework for rapid UI development.
- **[Templ UI](https://templui.io/)**: A collection of prebuilt components for Templ, making it easy to build beautiful UIs.
- **[Air](https://github.com/air-verse/air)**: A live reloading tool for Go applications, making development faster and smoother.


## Project Structure

``` If you're actually looking at this, message me if you do anything cool with this template! xD
├── cmd/                # Entrypoint (main.go)
├── internal/           # Application logic
│   ├── handlers/       # HTTP handlers
│   │   ├── errors/     # Centralized error response logic
│   │   └── v1/         # Versioned routing
│   └── httpserver/     # Server wiring & middleware
├── .templui.json       # Templ UI config
├── Makefile            # Dev commands
```

---

## Get Started

### 1. Clone the repo

```bash
git clone https://github.com/brian-nunez/bbaas-api.git
cd bbaas-api
```

### 2. Install dependencies

* Go 1.22+
* templ
* tailwindcss
* air (for live reloading)

### 3. Run in dev mode

```bash
make dev
```

## Reach out if you have questions or just want to chat!

- [GitHub](https://www.github.com/brian-nunez)
- [LinkedIn](https://www.linkedin.com/in/brianjnunez)
