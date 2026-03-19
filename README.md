# Event Log Service - Take-Home Challenge

## Overview

Build a simple event logging service in Go. The service exposes an HTTP API for submitting events, querying them with filters and pagination, and retrieving basic statistics.

**Time expectation:** 2-3 hours maximum. We value working software over polish, it's fine to cut corners, just note what you'd improve.

## Requirements

### API Endpoints

- `POST /events` — create an event with `type` (string), `payload` (JSON object), and `timestamp` (auto-generated)
- `GET /events` — list events, with optional filtering by `type` and date range, and **pagination** via `limit`/`offset` query params (default limit: 50). Response must include `total` count or a `has_more` flag
- `GET /events/stats` — return event counts grouped by `type`. Supports optional `start`/`end` query params to filter the time range. Example response: `[{"type": "user.login", "count": 42}]`

### Additional Requirements

- **Deduplication** — if an event with the same `type` and `payload` was already created within the last 5 minutes, reject it with `409 Conflict` instead of creating a duplicate
- Persist events to SQLite or another database of your choice. Data should survive server restarts.
- Include tests covering **at least two distinct behaviors**

## Project Structure

This repo contains a bare `main.go` to save you some setup. Feel free to restructure, add dependencies, or start from scratch – we're interested in your solution, not this starting point.

## Getting Started

```bash
go run .
# Server runs on http://localhost:8080
```

## What We're Looking For

- **Working code** that we can run locally
- **Reasonable structure** — navigable, not necessarily perfect
- **Your judgment** — make assumptions where needed, just document them

## Deliverables

1. This repo with your implementation
2. Your answers to the questions below
3. Be prepared for a 15-30-minute walkthrough where we'll discuss your decisions and possibly make a small modification
   together

---

## Your Answers

Please fill these out before submitting.

### 1. What trade-offs did you make and why?

_Your answer here_

### 2. If this needed to handle 10,000 events per second, what would you change?

_Your answer here_

### 3. What's one thing you'd add or refactor given another few hours?

_Your answer here_