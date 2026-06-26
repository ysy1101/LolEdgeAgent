# Agent 工作流程

## 架构：Agent 驱动 vs 管线驱动

```
管线模式（一键生成）                 Agent 模式（自由对话）
──────────────────                 ──────────────────
用户点击"生成简报"                   用户自由提问
  → 固定流程：采集→排序→摘要→组装       → LLM 自主决定调用哪个工具
  → 无论用户想不想，都走全流程           → 可以只看不生成、可以搜索、可以查偏好
```

两种模式共存：管线模式走 BriefingService，Agent 模式走 Agent Chat。

---

## Agent 决策循环

```
用户: "最近有什么简报？"
    │
    ▼
┌──────────────────────────────────────────┐
│  第 1 轮                                   │
│  ├─ 发送消息 [system + history + user]     │
│  ├─ LLM 返回:                              │
│  │   {"tool":"list_briefings","args":"{}"} │
│  ├─ 执行 list_briefings → 获得简报列表     │
│  └─ 结果拼回消息，继续下一轮               │
│                                           │
│  第 2 轮                                   │
│  ├─ 发送消息 [system + ... + tool_result]  │
│  ├─ LLM 看到工具结果，分析数据              │
│  └─ 返回 Markdown 文本（无 JSON tool）     │
│     → 视为最终回答，展示给用户              │
└──────────────────────────────────────────┘
```

**循环规则：**
- 每轮 LLM 返回 `{"tool":"..."}` → 执行工具 → 继续循环
- LLM 返回普通文本（无 tool JSON） → 最终回答
- 最大 8 轮，防死循环
- 单次调用 30s 超时

---

## 工具调用协议

**请求（System Prompt 中定义）：**
```
需要调用工具时：
{"tool":"<工具名>","args":"<JSON 参数>"}
```

**实际 LLM 输出示例：**
```json
{"tool":"search_articles","args":"{\"query\":\"Rust\",\"limit\":10}"}
```

**工具结果（拼入下轮消息）：**
```
工具 search_articles 执行结果:
[{"id":1,"title":"Rust 2026 Roadmap","desc":"...","url":"..."},...]
```

**兼容性：** 这是纯文本协议，不依赖 OpenAI 的 `tools` 参数，所有 LLM 都支持（DeepSeek、Qwen、Moonshot 等）。

---

## 当前实现的 5 个工具

| 工具 | 输入 | 输出 | 后端实现 |
|------|------|------|---------|
| search_articles | `{query, limit}` | 文章列表（id, title, desc, url） | RAGService.Search() |
| list_briefings | `{limit}` | 简报列表（id, title, count, status） | BriefingRepo.List() |
| get_briefing | `{id}` | 简报详细内容（markdown） | BriefingRepo.GetByID() |
| generate_briefing | `{}` | `{briefing_id, status}` | BriefingService.GenerateAsync() |
| get_preferences | `{}` | `{keywords, max_briefing_articles}` | PreferenceRepo.Get() |

---

## 对话管理

```
conversations                     messages
───────────                       ────────
id                                id
user_id                           conversation_id
title                             role (user / assistant)
created_at                        content
updated_at                        created_at
```

- **对话列表**：支持新建/切换/删除
- **消息持久化**：每次对话的 user/assistant 消息存入 DB
- **历史召回**：切换对话时加载全部消息
- **上下文传递**：Agent 调用时携带最近 20 条历史消息

---

## 前端交互

对话页面（Chat.tsx）：

```
┌──────────────────────────────────────────┐
│  [新对话]                                 │
│  □ 历史对话1     ← 可切换                  │
│  □ 历史对话2     ← 可删除                  │
│                                           │
│  ┌──────────────────────────────────┐    │
│  │ 用户: 最近有什么简报？              │    │
│  └──────────────────────────────────┘    │
│  ┌──────────────────────────────────┐    │
│  │ 助手: 最近生成了 3 份简报...        │    │
│  │                                   │    │
│  │ ▸ Agent 思考过程 (3 步)            │    │
│  │   model  我来帮你查看简报          │    │
│  │   tool_call  list_briefings({})   │    │
│  │   model  找到3份简报：1.xxx 2.xxx  │    │
│  └──────────────────────────────────┘    │
│                                           │
│  [输入问题...]              [发送]         │
└──────────────────────────────────────────┘
```

- 助手回复用 ReactMarkdown 渲染（支持标题、链接、代码等）
- 工具调用过程可折叠查看（用于调试和面试演示）
- 左侧对话列表支持切换历史对话

---

## 启动方式

```bash
# 后端
cd backend
set LLM_API_KEY=sk-your-key
go run cmd/server/main.go

# 前端
cd frontend
npm run dev
```

进入 `/chat` 页面即可开始对话。
