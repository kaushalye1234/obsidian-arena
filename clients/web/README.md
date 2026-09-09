# Obsidian Arena web client

The playable React client lives in the same repository as the Go backend.
Requires Node.js 22.13 or newer and Docker Desktop.

## Run locally

From the repository root, start the API and database:

```sh
docker compose up --build
```

In a second terminal, from the repository root:

```sh
cd clients/web
npm install
npm run dev
```

Open http://localhost:5173. Keep the Backend URL set to
http://localhost:8090. These ports match the repository's Docker Compose
configuration and allowed browser origins. Use localhost rather than 127.0.0.1.

## Play multiplayer

1. In one browser tab, enter a display name and create a room.
2. In another tab, enter a different name and join using the room code.
3. Mark both players ready, then start the match in either tab.
4. Move using WASD, arrows, or the direction buttons. Releasing a control stops movement.
5. At the end of the match, the server supplies final scores and the client stops input.

The server owns multiplayer positions, crystal collection and match results.
The existing completion handler records results in PostgreSQL. Backend levels
and a progress/leaderboard UI are still follow-up work.

## Offline demo

Play instant demo works without Docker. It has timed levels with cumulative
session scores. Demo scores and levels are local to the open page and are not
submitted to the multiplayer leaderboard or saved across reloads. Nyx is a
stationary demo marker, not a networked opponent.

## Checks

```sh
npm run build
```

This runs TypeScript validation and creates a production bundle in dist/.
The existing hosted preview is maintained separately; this folder is the
portable local client and does not need a Sites account.
