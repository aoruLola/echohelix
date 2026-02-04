# GEMINI.md - EchoHelix Project Context

> **Identity**: This file is the Shared Brain for all Agents working on EchoHelix.
> **Policy**: **GREENFIELD DEVELOPMENT**. Do not copy legacy code.
> **Core Policy**: **VENDORING**. Clone official repositories. Modify source directly if wrapping is insufficient.
> **Conflict Priority**: **APP (Flexible) > BRIDGE > CORE (Stable)**.

## 1. Project Overview
EchoHelix is a mobile-first AI coding assistant that enables users to edit code on their server/PC from a mobile device using powerful local or remote LLMs.

## 2. Architecture: Dual-Core Bridge
The system uses a "Bridge" to expose two distinct AI kernels to the mobile app.

```mermaid
graph TD
    App[Mobile App (Flutter)] <-->|WS/JSON-RPC| Bridge[Go Bridge :8765]
    
    subgraph "The Cores"
        Bridge -->|Spawn/HTTP| KernelA[Gemini Core (Node.js)]
        Bridge -->|Spawn/HTTP| KernelB[Aider Core (Python)]
    end
    
    KernelA -.->|Fast/Chat| LLM1[LLM A]
    KernelB -.->|Complex/Edit| LLM2[LLM B]
```

### 2.1 The Bridge (Go)
*   **Role**: Gateway, Process Manager, Auth Provider.
*   **Tech**: Go 1.23+
*   **Key Resp**:
    *   Serve WebSocket at `/api/v2/chat/proxy`.
    *   Manage `node` and `python` subprocesses.
    *   Handle file system operations (LS, CAT, WRITE) directly for speed.

    > [!IMPORTANT]
    > **Bridge Philosophy**: The Bridge is a **Transparent Gateway** (Dumb Pipe). It must NOT attempt to "think", "summarize", or "modify" the prompt logic. Its only job is to reliably ship raw JSON between the App and the Core. The Intelligence lives 100% in the Core.

### 2.2 Core A: Gemini (Node.js)
*   **Role**: Conversationalist. Quick answers, explanations, single-file edits.
*   **Protocol**: Compatible with OpenAI Chat Completion API (simulated or proxied).
*   **Mod**: Must accept `COMPAT_BASE_URL` to work with any provider.

### 2.3 Core B: Aider (Python)
*   **Role**: Engineer. Multi-file refactoring, complex logic changes, git-aware.
*   **Native**: Wraps `aider` CLI.
*   **Interface**: Exposes a localized HTTP API (FastAPI) that the Bridge calls.

## 3. Communication Protocol (A2A v2)

### 3.1 App <-> Bridge
**WebSocket Format**:
```json
// Request
{
  "id": "uuid",
  "kernel": "gemini|aider",
  "method": "chat|edit|fs.ls",
  "params": { ... }
}

// Response
{
  "id": "uuid",
  "result": { ... }, // OR
  "error": { "code": 123, "message": "..." }
}
```

## 4. Agent Coordination Protocol (The "Sync Loop")

> [!IMPORTANT]
> **Workflow**: Execute -> Report -> Architect Design -> Next Instruction

### 4.1 Locations
*   **Sync Directory**: `services/echohelix/.agent_sync/`

### 4.2 The Loop
1.  **Agent Execution**: Agents work on their tasks.
2.  **Report**: At the end of **EVERY** run, each agent MUST write a report to:
    *   `report_app.md` (Agent 1)
    *   `report_bridge.md` (Agent 2)
    *   `report_core.md` (Agent 3)
    *   **Format**: Summary of done items, blocked items, and questions for Architect.
3.  **Architect Design**: The Architect (User/Root Agent) reads reports and writes specific next steps to:
    *   `instruction_app.md`
    *   `instruction_bridge.md`
    *   `instruction_core.md`
4.  **Next Step**: Agents read their respective `instruction_*.md` file as the Prompt for the next run.

### 4.3 Standards
*   **Language**: All Reports (`report_*.md`) and Instructions (`instruction_*.md`) MUST be written in **Chinese (中文)**.
*   **Terminal**: All terminal commands in plans/reports MUST use standard **PowerShell** syntax.

## 5. Workspaces & Agents
*   **Agent 1 (Frontend)**: `services/echohelix/elix-app`
*   **Agent 2 (Bridge)**: `services/echohelix/elix-bridge`
*   **Agent 3 (Core)**: `services/echohelix/elix-core`

## 5. Deployment
*   **User Journey**:
    1.  User downloads `EchoHelix Server` (Single Binary).
    2.  Runs `./echohelix server`.
    3.  Server checks env (Node? Python? Git?).
    4.  Asks user: "Select your primary AI Engine".
    5.  Starts Bridge + Selected Kernel.
    6.  Displays QR Code for App pairing.
