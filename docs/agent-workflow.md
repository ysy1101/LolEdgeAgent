# Agent 工作流程设计

## 当前模式 vs Agent 模式

```
当前（管线模式）                      目标（Agent 模式）
──────────────────                   ──────────────────
用户点击"生成简报"                    用户自由对话
  → 固定流程：采集→排序→摘要→组装         → LLM 自主决定做什么
                                          → 可以只查不生成、可以提问、可以改偏好

用户搜索文章                          同一入口
  → 固定调用 embedding 搜索              → 搜索、生成、问答、管理 统一走 Agent
```

---

## Agent 决策循环

```
用户: "最近 Rust 有什么新进展？帮我整理一个周报"
    │
    ▼
┌──────────────────────────────────────────┐
│  ① 组装 System Prompt                     │
│  ├─ role: 你是内容聚合 Agent               │
│  ├─ user_profile: {关键词偏好}             │
│  ├─ memories: {...近期记忆...}             │
│  └─ tools: [search_articles, generate_briefing, ...]
│                                           │
│  ② LLM 决策（第 1 轮）                     │
│  → 调用 search_articles("Rust", limit=20)  │
│                                           │
│  ③ 获取 Tool 结果                          │
│  ← [{id:1, title:"Rust 2026 roadmap"},...] │
│                                           │
│  ④ LLM 决策（第 2 轮）                     │
│  → 分析结果："找到 20 篇，筛选出 5 篇重点"   │
│  → 调用 analyze_topic(筛选结果)             │
│                                           │
│  ⑤ 获取分析结果                             │
│  ← "社区焦点：异步运行时、嵌入式..."         │
│                                           │
│  ⑥ LLM 决策（第 3 轮）                     │
│  → 调用 generate_briefing(筛选文章, 周报)   │
│                                           │
│  ⑦ 获取简报结果                             │
│  ← Markdown 周报                           │
│                                           │
│  ⑧ LLM 最终回复                            │
│  → 总结 + 展示简报 + 推荐重点                │
└──────────────────────────────────────────┘
    │
    ▼
用户看到结果（含简报 Markdown + 分析）
```

**循环规则：**
- 每轮 LLM 可以返回 `tool_call` 或 `final_answer`
- `tool_call` → 执行 → 结果拼回 messages → 继续循环
- `final_answer` → 展示给用户 → 结束
- 有最大轮数限制（比如 10 轮），防止死循环

---

## System Prompt 结构

每次对话开始时组装：

```
你是 LolEdgeAgent，一个内容聚合和知识助手。

## 你的能力
- 搜索已采集的文章库（semantic search）
- 生成内容简报（Markdown 格式）
- 分析话题趋势
- 管理文章收藏
- 管理内容源
- 根据用户反馈调整偏好

## 你拥有的数据
- 从 RSS/HN/GitHub 采集的大量技术文章
- 每篇文章有标题、描述、正文、来源、发布时间
- 支持向量语义搜索

## 用户画像
{从 preferences 加载的 keywords, excluded_keywords, 偏好语言等}

## 相关记忆
{从 memories 检索的近期对话摘要，最多 5 条}

## 当前时间
{now}

## 回复风格
- 中文回复
- 简洁，不说废话
- 给出事实和来源
```

---

## Tool 调用格式

```
Role: assistant
Content: null
ToolCalls: [
  {
    "id": "call_1",
    "function": {
      "name": "search_articles",
      "arguments": "{\"query\": \"Rust async\", \"limit\": 10}"
    }
  }
]
```

```
Role: tool
Content: [{"id": 1, "title": "...", "url": "...", "summary": "..."}, ...]
ToolCallID: "call_1"
```

---

## Tool 列表与实现映射

| Tool | 后端实现 | 输入 | 输出 |
|------|---------|------|------|
| search_articles | RAGService.Search() | query, limit | []Article |
| get_article | ArticleRepo.Get() | article_id | Article |
| generate_briefing | BriefingService.GenerateAsync() | - | briefing_id |
| list_briefings | BriefingService.List() | page, limit | []Briefing |
| get_latest_briefing | BriefingService.List(page=1) | - | Briefing |
| add_bookmark | BookmarkRepo.Create() | article_id | - |
| remove_bookmark | BookmarkRepo.Delete() | article_id | - |
| list_bookmarks | BookmarkRepo.List() | - | []Article |
| get_preferences | PreferenceRepo.Get() | - | Preference |
| update_preferences | PreferenceRepo.Update() | Preference | - |
| analyze_topic | LLM.Chat() + RAG | topic | 分析文本 |
| list_sources | SourceRepo.List() | - | []Source |

---

## 记忆系统工作流

```
对话开始
    │
    ├─ 检索 memories（按 keywords 匹配 top 5）
    ├─ 拼入 {memory_context}
    ├─ 加载最近 N 轮 messages（在 context window 内）
    │
    ▼
【Agent 对话循环 ...】
    │
    ▼
对话结束
    │
    ├─ 提取本轮对话的全部 messages
    ├─ 调 LLM 生成压缩摘要：
    │    "用户关注 Rust 异步运行时和嵌入式发展，
    │     偏好技术分析类深度文章，对新闻速报兴趣较低"
    ├─ 存入 memories 表 (user_id, content, keywords, importance)
    │
    ▼
下一轮对话时检索到这条记忆
```

**压缩 Prompt：**
```
根据以下对话记录，生成一段简洁的摘要，包含：
1. 用户的核心意图是什么
2. 用户是否表露了新的偏好或兴趣
3. 用户对哪些文章/话题表现出特别关注

只返回摘要，不超过 150 字。
```

---

## 对话管理（数据库）

```
conversations                    messages
───────────                      ────────
id                              id
user_id                         conversation_id
title ("Rust 周报讨论")          role (system/user/assistant/tool)
created_at                      content
updated_at                      tool_calls (JSON)
                                created_at

memories
────────
id
user_id
content ("用户关注 Rust 异步...")
keywords (["rust","async","embedded"])
importance (0.0~1.0)
created_at
```

---

## 前端升级

当前 Chat.tsx 只支持一问一答。Agent 模式前端需要：

| 功能 | 说明 |
|------|------|
| 多轮对话 | 保留全部 messages 上下文，后轮带着前轮结果 |
| 工具调用展示 | 当 LLM 调用工具时，显示 "🔧 正在搜索文章..."
| 流式输出 | SSE 推送 LLM 生成的 token，逐字显示 |
| 简报卡片内嵌 | 当 LLM 返回简报时，不显示 Markdown 文本，而是渲染一个"查看简报"卡片 |
| 对话列表 | 左侧显示历史对话列表，可切换/新建 |
| 记忆提示 | 顶部显示"基于你的偏好和之前对 Rust 的关注，为你推荐..." |

---

## 实现路线（再次细化）

| 阶段 | 内容 | 工作量 |
|------|------|--------|
| **A** | conversations + messages 表 + repository | 小 |
| **B** | Tool 定义 + 注册表（12 个 Tool） | 中 |
| **C** | Agent 循环引擎（替换管线） | 中 |
| **D** | memories 表 + 压缩摘要 | 小 |
| **E** | 前端多轮对话 + 工具调用展示 + 对话列表 | 中 |
| **F** | 流式输出（SSE） + 简报卡片内嵌 | 中 |

---

## 启动方式

前端从 /chat 进入，后端需要新增：

```
POST /api/v1/agent/chat          ← 对话入口（替代 /ask，支持 tool calling）
GET  /api/v1/conversations       ← 历史对话列表
GET  /api/v1/conversations/:id   ← 某对话详情含 messages
POST /api/v1/conversations       ← 新建对话
DELETE /api/v1/conversations/:id ← 删除对话
```
