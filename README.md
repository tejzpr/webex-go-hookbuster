# Webex Go Hookbuster

[![CI](https://github.com/tejzpr/webex-go-hookbuster/actions/workflows/ci.yml/badge.svg)](https://github.com/tejzpr/webex-go-hookbuster/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/tejzpr/webex-go-hookbuster)](https://goreportcard.com/report/github.com/tejzpr/webex-go-hookbuster)
[![Go Reference](https://pkg.go.dev/badge/github.com/tejzpr/webex-go-hookbuster.svg)](https://pkg.go.dev/github.com/tejzpr/webex-go-hookbuster)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL%202.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)
[![Docker](https://img.shields.io/docker/v/tejzpr/webex-go-hookbuster?label=Docker&sort=semver)](https://hub.docker.com/r/tejzpr/webex-go-hookbuster)

A WebSocket-to-HTTP event bridge for Webex, written in Go using the [webex-go-sdk](https://github.com/tejzpr/webex-go-sdk).

Hookbuster connects to the Webex Mercury WebSocket service and forwards real-time events as HTTP POST requests to a local target application — eliminating the need for public webhook URLs during development.

## Features

- **Real-time event forwarding** via Webex Mercury WebSocket
- **Supported resources**: rooms, messages, memberships, attachmentActions
- **Interactive CLI** for guided setup
- **Environment variable mode** for automated / container deployments
- **Firehose mode** subscribes to all resources and all events
- **Graceful shutdown** on SIGINT / SIGTERM
- **End-to-end decryption** of message content via the SDK's KMS integration

## Supported Resources & Events

| Resource          | Events                        |
| ----------------- | ----------------------------- |
| rooms             | created, updated              |
| messages          | created, deleted              |
| memberships       | created, updated, deleted     |
| attachmentActions | created                       |

## Quick Start

### Prerequisites

- Go 1.24+
- A Webex access token (get one at https://developer.webex.com)

### Build

```bash
go build -o hookbuster .
```

### Run — Interactive Mode

```bash
./hookbuster
```

You will be prompted for:
1. Webex access token
2. Forwarding target (e.g. `localhost`)
3. Forwarding port (e.g. `8080`)
4. Resource selection
5. Event selection

### Run — Environment Variable Mode

```bash
TOKEN=<your-webex-token> PORT=8080 TARGET=localhost ./hookbuster
```

When `TOKEN` and `PORT` are set, hookbuster automatically subscribes to **all** resources with **all** events (firehose mode).

| Variable | Required | Default     | Description                    |
| -------- | -------- | ----------- | ------------------------------ |
| `TOKEN`  | Yes      | —           | Webex access token             |
| `PORT`   | Yes      | —           | Target forwarding port         |
| `TARGET` | No       | `localhost` | Target hostname or IP address  |

### Docker

Build and run from the parent directory (which contains both `webex-go-sdk/` and `webex-go-hookbuster/`):

```bash
docker build -f webex-go-hookbuster/Dockerfile -t hookbuster .
docker run -e TOKEN=<your-token> -e PORT=8080 -e TARGET=host.docker.internal hookbuster
```

## How It Works

```
Webex Cloud (Mercury WebSocket)
        │
        ▼
   ┌──────────┐
   │ Mercury  │  WebSocket connection with auto-reconnect
   │ Client   │  and ping/pong keepalive
   └────┬─────┘
        │
        ▼
   ┌──────────────┐
   │ Conversation │  Parses activities, decrypts messages,
   │ Client       │  dispatches by verb (post, share, add...)
   └────┬─────────┘
        │
        ▼
   ┌──────────┐
   │ Listener │  Maps verbs to resource/event pairs,
   │          │  filters by user's subscription
   └────┬─────┘
        │
        ▼
   ┌───────────┐
   │ Forwarder │  HTTP POST to target:port
   └───────────┘
        │
        ▼
   Your Application (localhost:port)
```

## License

MPL-2.0
