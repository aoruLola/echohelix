# EchoHelix 战略路线图 (ROADMAP)

> **核心愿景**: 打造移动端原生 (Mobile-Native) 的全能 AI 编程工作台。
> **层级定位**: 宏观战略 (Vision) -> 阶段目标 (Milestone)。

## 📅 时间轴与阶段 (Timeline)

| 阶段 (Phase) | 核心目标 (Goal) | 状态 (Status) | 复杂度 | 交付物 (Deliverables) |
| :--- | :--- | :--- | :--- | :--- |
| **Phase 1** | **原型验证 (Prototype)**<br>验证 Go Bridge + Node.js 基础通信 | ✅ 完成 | ⭐ | Basic RPC, Process Spawning |
| **Phase 2** | **架构确立 (Architecture)**<br>双核架构 (Gemini/Aider) + WebSocket 路由 | ✅ 完成 | ⭐⭐ | Bridge v2, Router Logic |
| **Phase 3** | **内核生产化 (Production Core)**<br>A2A 协议适配, Aider 流式输出 | 🔄 进行中 | ⭐⭐⭐ | Robust Process Manager, Streaming API |
| **Phase 4** | **移动端基建 (Mobile Foundation)**<br>Flutter UI, 基础聊天与文件树 | 🔄 进行中 | ⭐⭐ | App v1 (APK), Chat UI |
| **Phase 5** | **全链路验证 (E2E Verification)**<br>App -> Bridge -> Core -> Code Edit 闭环 | 🚧 阻塞中 | ⭐⭐⭐⭐ | Verified E2E Flow |
| **Phase 6** | **体验打磨 (Experience Polish)**<br>符号化输入 (@, /), 紧急停止 (🛑) | 📅 计划中 | ⭐⭐ | Smart Input, Safety Controls |
| **Phase 7** | **未来交互 (Future Interaction)**<br>语音编程 (Voice), 多会话并行 | 🔮 规划中 | ⭐⭐⭐⭐⭐ | Voice Engine, Multi-Session Manager |

## 🚩 关键里程碑 (Key Milestones)

- [x] **M1: Hello World**: 跑通第一个 "User -> Bridge -> LLM" 回合。
- [ ] **M2: Stable Edit**: 实现第一次 "手机端发起 -> 后端自动修改文件 -> 手机端确认" 的稳定闭环 (Phase 5)。
- [ ] **M3: Vibe Coding**: 引入 `@` 和 `/` 指令，实现流畅的手机端操作流 (Phase 6)。
- [ ] **M4: Hands-free**: 实现语音控制，脱离键盘 (Phase 7)。
