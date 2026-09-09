# Obsidian Arena

Obsidian Arena is a server-authoritative real-time multiplayer mini-game backend written in Go. One API supports React web and React Native mobile clients.

## MVP capabilities

- 2-4 player rooms with shareable room codes
- WebSocket input and 20 Hz authoritative state snapshots
- Ready/start flow and 60-second matches
- PostgreSQL player progress, scores, wins, and leaderboard
- Input sequencing, arena boundary validation, origin allow-listing, and payload limits
- Docker Compose development environment
- Obsidian-ready project vault with architecture, protocol, tasks, and templates

## Run

Requirements: Docker Desktop or Docker Engine with Compose.

```bash
cp .env.example .env
docker compose up --build
```

The API starts at `http://localhost:8080`. Check it with `curl http://localhost:8080/health`.

## Basic client flow

1. `POST /api/v1/players` with `{"displayName":"Chamindu"}`.
2. `POST /api/v1/rooms` and copy the room code.
3. Connect to `ws://localhost:8080/api/v1/rooms/{code}/connect?playerId={id}&name={name}`.
4. Send `{"type":"ready","ready":true}` from at least two players.
5. Send `{"type":"start"}` once.
6. Send `{"type":"input","dx":1,"dy":0,"sequence":1}`.

Open `docs-vault` as an Obsidian vault for the complete design and protocol.

## Development

```bash
make test
make fmt
make up
make down
```

Next: replace placeholder movement scoring with collectibles and hazards, add authentication and reconnect tokens, then build the React and React Native clients.
