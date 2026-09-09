# WebSocket Protocol

Connect with:

```text
GET /api/v1/rooms/{code}/connect?playerId={uuid}&name={displayName}
```

## Client messages

```json
{"type":"ready","ready":true}
```

```json
{"type":"start"}
```

```json
{"type":"input","dx":0.7,"dy":-0.7,"sequence":42}
```

`sequence` must increase so delayed inputs can be rejected. Server events are `connected`, `player_joined`, `player_left`, `snapshot`, `match_finished`, and `error`.

## Snapshot

```json
{
  "type":"snapshot",
  "data":{
    "roomCode":"A7CD2K",
    "status":"playing",
    "players":{"player-uuid":{"id":"player-uuid","name":"Chamindu","x":42.2,"y":18.4,"score":83,"ready":true}},
    "remainingMs":45120,
    "tick":298
  }
}
```
