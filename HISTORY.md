# EchoHelix Version History 📜

## v3.0: The Rebirth (Current)
- **Status**: Development
- **Goal**: Standard A2A Protocol, Clean Architecture.
- **Key Changes**:
  - Reverting to standard `gemini-cli` Core.
  - "Transparent Pipe" Bridge architecture.
  - Mobile-first "Vibe Coding" UI.

---

## v2.0: The Foundation (Legacy)
> *Archived in `archive_v2`*

- **Core Concept**: Workspace + Session Management.
- **Key Features**:
  - **Backend**: Implemented complete CRUD for Workspaces/Sessions.
  - **Frontend**: Riverpod-based Architecture, Workspace Picker UI, Session List.
  - **Fixes**: Resolved Gemini port binding (`0.0.0.0`) and path issues.
- **Lessons Learned**:
  - Attempting to translate Chat to Command (`/executeCommand`) was a mistake.
  - Protocol mismatch (HTTP vs JSON-RPC) blocked progress.
  - **Pivot**: Decided to strictly follow the official A2A protocol in v3.

## v1.0: The Prototype (Legacy)
> *Archived in `archive_v1`*

- **Core Concept**: Initial "App - Bridge - DualCore" Architecture.
- **Key Features**:
  - **Bridge**: Basic process spawning (Gemini/Aider).
  - **App**: Prototype UI.
  - **Integration**: Initial Python wrapper for Aider (`aider_chk.py`).
- **Issues**:
  - Process management was unstable (zombie processes).
  - "Connection Refused" issues due to localhost binding.
