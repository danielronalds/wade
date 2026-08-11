# AGENTS.md

## Start here

Read [CONTRIBUTING.md](CONTRIBUTING.md) completely before changing the project.
It contains the development workflow, links to the architecture documentation,
frontend ownership rules and validation commands that apply to all changes.

Before changing the backend, also read
[`docs/backend-architecture.md`](docs/backend-architecture.md) completely and
follow every linked document relevant to the task.

## Project

WADE is a local-first, Web-based Agentic Development Environment. The Go backend
serves a Vue 3 and TypeScript frontend and connects browser terminals to real
project shells through PTYs and WebSockets.

## Product direction

- Keep primary workflows keyboard driven and provide predictable focus
  behaviour.
- Keep agent, miscellaneous, server and scratchpad shells close to the work.
- Preserve terminals for the lifetime of the server.
- Surface useful repository context without interrupting the coding workflow.
- Keep shells local and protect them from other origins.

## Implementation reminders

- Use xterm.js for terminals and preserve the existing Escape and WebSocket
  control-message behaviour.
- Avoid CDN JavaScript for the local application.
- Follow the backend dependency direction and frontend ownership rules in
  [CONTRIBUTING.md](CONTRIBUTING.md).
- Prefer a keyboard path when adding user interface workflows.
- Run `mise run test` before considering a change complete.
