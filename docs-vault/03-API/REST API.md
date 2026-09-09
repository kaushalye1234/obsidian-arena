# REST API

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/health` | Service health |
| POST | `/api/v1/players` | Create/retrieve player |
| POST | `/api/v1/rooms` | Create room |
| GET | `/api/v1/leaderboard?limit=20` | Top players |
| GET | `/api/v1/players/{id}/progress` | Load progress |
| PUT | `/api/v1/players/{id}/progress` | Save progress JSON |

Create a player with `{"displayName":"Chamindu"}`. Save progress with `{"progress":{"level":2,"unlockedSkins":["default"]}}`.
