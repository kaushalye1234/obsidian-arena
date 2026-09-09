# Game Design

Two to four players enter a top-down obsidian arena for a 60-second survival match. The server controls official positions, scores, time, and results.

## MVP loop

1. Create or join a room.
2. Mark all players ready.
3. Start and move around the arena.
4. Gain placeholder movement points.
5. Finish after 60 seconds and persist results.

Movement scoring makes the networking slice playable. Milestone two replaces it with server-spawned crystals, collision detection, health, and expanding lava hazards.

## Constraints

- 2-4 players per room
- 100 x 100 server-coordinate arena
- 20 authoritative ticks per second
- Maximum normalized speed: 18 units per second
- Clients send intention, never trusted position or score
