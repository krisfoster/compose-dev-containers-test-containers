# Issues & Future Ideas

Running list of known issues and feature ideas. Not prioritized or scheduled — just a backlog to draw from.

## Open

1. **Mobile support.** Make the game work better on phones: tap-to-move controls for the whale, and full-screen landscape play (prompt the user to rotate their phone when in portrait mode).

2. **Dev container support.** Add support for developing, testing, and running the whole app inside a dev container, with matching VS Code config.

3. **Leaderboard storage + score submission flow.** Store player name and score in a leaderboard backed by Redis.
   - Before the game starts, ask the player to enter their name.
   - On game over (death), show "Game Over" with their score, and write the name + score to the leaderboard store.
   - Show a "Replay" button below the game-over screen that restarts the game.
   - The leaderboard is its own Go-based API, defined with OpenAPI.
   - Security: only the game client should be able to write to the API — use a secret or similar mechanism so arbitrary clients can't submit scores.

4. **Docker Hardened Images (DHI).** Migrate all container images used by the app to DHI.

5. **Leaderboard page.** A standalone Go-based app/page that dynamically refreshes by polling/calling the leaderboard API.

6. **Container/bug-themed obstacles.** Update the game so some obstacles are container/bug themed (e.g. large bugs, tumbleweed, etc.) — options to be discussed at implementation time.

7. **K8s manifest generation via Compose Bridge.** Add support for using the Docker Compose Bridge to generate Kubernetes manifests from the compose file, so the app can be deployed to a k8s cluster.
