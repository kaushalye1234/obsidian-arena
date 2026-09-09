# Test Strategy

Unit tests cover readiness and arena boundaries. Add integration tests for migrations, match persistence, leaderboard order, and progress round-trips. Add WebSocket tests for the four-player limit, snapshots, disconnects, and delayed packets.

Before public deployment, simulate 100 four-player rooms and measure latency, CPU, memory, dropped snapshots, and database writes.
