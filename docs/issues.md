# Issues & Future Ideas

Running list of known issues and feature ideas. Not prioritized or scheduled — just a backlog to draw from. Item numbers are stable identifiers (referenced elsewhere), so they are preserved even after an item moves to **Done**.

## Open


7. **K8s manifest generation via Compose Bridge.** Add support for using the Docker Compose Bridge to generate Kubernetes manifests from the compose file, so the app can be deployed to a k8s cluster. Add scripts to easily deploy the generated manifests to a local k8s cluster running on Docker desktop (Kind cluer). Use the industry standard tools to do this - no surprises, it should work as expected.

8. **Improve documentation and user notes**. Review the code and configuration and add explanatory notes throughout to make this an instructive project for people looking to learn about docker technologies, and the Go language.

9. **Create a learning outcomes focused approach to the README.** The current Readme explains how to start the app, run it, develop it, which we still need. But it would be much better if the README was structured around learning outcomes. What will the user learn from looking at this repo. Review the repo, identify the learning outcomes, document them in a "What you will learn" style section. Structure the README as a tutorial that progresses through the learning outcomes in a structured, logical manner. Add a summary section at the end that explains what the user has learnt by looking through the tutorial. We will not ask the user to write any code, only to run command to make things happen and to look at code. The focus should be on the docker technolgies in particular. I would leave development using SBX out of the tutorial. That can be addressed later.

10. **Add support for using Testcontainers cloud.** Test container runs can be shipped to testcontainers cloud. Research how this would work, in particular from within a dev container. Using this research add support, enabled through some configuration, to allow for users to puhs test container runs to the cloud. Ensure that the SBX startup scripts and direnv config are updated to support this.

11. **Remove Refences to GDS.** This is no longer used and references to it dhoudl be removed from the project. Instead we use speckit and links to that should be added.

13. **Harden the security around Reddis in the App.** Make Reddis only accessible from the go app, with no publicly exposed endpoints.

## Done

1. ✅ **Mobile support.** Make the game work better on phones: tap-to-move controls for the whale, and full-screen landscape play (prompt the user to rotate their phone when in portrait mode). Also, update the home page so that the QR code is displayed on the right hand side (2-col layout) and ensure that it refreshes identically to how it does on the QR code page.

2. ✅ **Dev container support.** Add support for developing, testing, and running the whole app inside a dev container, with matching VS Code config.

3. ✅ **Leaderboard storage + score submission flow.** Store player name and score in a leaderboard backed by Redis. _(Implemented — see [`specs/003-leaderboard-score-submission`](specs/003-leaderboard-score-submission).)_
   - Before the game starts, ask the player to enter their name.
   - On game over (death), show "Game Over" with their score, and write the name + score to the leaderboard store.
   - Show a "Replay" button below the game-over screen that restarts the game.
   - The leaderboard is its own Go-based API, defined with OpenAPI.
   - Security: only the game client should be able to write to the API — use a secret or similar mechanism so arbitrary clients can't submit scores.

4. ✅ **Docker Hardened Images (DHI).** Migrate all container images used by the app to DHI. _(Implemented — see [`specs/005-dhi-image-migration`](specs/005-dhi-image-migration). golang/alpine/redis migrated to DHI; ngrok exempt (no DHI equivalent). Note the DHI Redis `protected-mode` workaround documented in that feature's image inventory.)_

5. ✅ **Leaderboard page.** A standalone Go-based app/page that dynamically refreshes by polling/calling the leaderboard API. _(Implemented — see [`specs/004-leaderboard-page`](specs/004-leaderboard-page).)_

12. ✅ **Add CC attributions to the game.** They should be there, but they are not. If allowed, add the CC attributions on the start view of the app, where the user enters their name. Also add a link to the game to get to the leader-board.

6. ✅ **Container/bug-themed obstacles.** Update the game so some obstacles are container/bug themed. _(Implemented — see [`specs/010-container-obstacles`](specs/010-container-obstacles). Truck-lane obstacles replaced with the "Container" 3D model (CC BY 4.0, Willy Decarpentrie) loaded via GLTFLoader, with voxel fallback.)_
