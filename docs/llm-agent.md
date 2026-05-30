# LLM Agent 架构设计

## 定位

LLM 不是流水线上的一个节点，而是整个系统的"大脑"——做决策、选工具、组织回答。现有的采集管线、摘要生成等退居幕后，成为 Agent 可调用的工具。

---

## 整体架构

```
用户输入
    │
    ▼
┌─────────────────────────────────────────────────────┐
│                    Agent 核心                         │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐       │
│  │  系统提示   │  │  记忆召回  │  │  用户画像  │       │
│  │ (System)  │  │ (Memory)  │  │ (Profile) │       │
│  └───────────┘  └───────────┘  └───────────┘       │
│         │            │              │               │
│         └────────────┼──────────────┘               │
│                      ▼                              │
│            ┌─────────────────┐                      │
│            │   LLM + Tools   │                      │
│            │  (Eino Graph)   │                      │
│            └────────┬────────┘                      │
│                     │                               │
│     ┌──────┬────────┼────────┬──────┐              │
│     ▼      ▼        ▼        ▼      ▼              │
│  搜索文章  生成简报  收藏管理  知识问答  查看源       │
│  (Tool)  (Tool)   (Tool)   (Tool)  (Tool)          │
└─────────────────────────────────────────────────────┘
     │              │              │
     ▼              ▼              ▼
┌──────────┐  ┌──────────┐  ┌──────────┐
│ 现有管线  │  │ 向量知识库 │  │ 用户数据  │
│ 采集/摘要 │  │ (RAG)    │  │ 偏好/历史 │
└──────────┘  └──────────┘  └──────────┘
```

---

## 记忆系统（三层）

### 第一层：短期记忆

**当前对话上下文中的 messages**，直接拼入 prompt，受 model context window 限制。

- 最近 N 轮对话完整保留（N 根据模型 token 上限动态调整）
- 超出窗口的历史消息由 LLM 生成压缩摘要，存入长期记忆

### 第二层：长期记忆

**跨会话的记忆存档**，每轮对话结束后 LLM 生成压缩摘要存入 DB。下次对话时按关键词检索相关记忆，拼入 System Prompt。

```sql
CREATE TABLE conversations (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL,
    title       TEXT NOT NULL DEFAULT '新对话',
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE messages (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id INTEGER NOT NULL,
    user_id         INTEGER NOT NULL,
    role            TEXT NOT NULL,     -- user / assistant / system / tool
    content         TEXT NOT NULL,
    tool_calls      TEXT,              -- JSON, LLM 工具调用记录
    token_count     INTEGER DEFAULT 0,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
CREATE INDEX idx_messages_conversation ON messages(conversation_id, created_at);

CREATE TABLE memories (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL,
    content     TEXT NOT NULL,                     -- 压缩后的记忆内容
    keywords    TEXT NOT NULL DEFAULT '[]',         -- 关键词 JSON，用于检索
    source_type TEXT NOT NULL DEFAULT 'conversation',
    importance  REAL NOT NULL DEFAULT 0.5,          -- 重要性 0~1
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id)
);
CREATE INDEX idx_memories_user ON memories(user_id, created_at DESC);
```

**记忆召回时机：**
- 每轮对话开始，从 memories 中按关键词检索 top N 条最相关记忆
- 拼入 System Prompt 的 `{memory_context}` 占位符
- 对话结束时，LLM 生成本轮压缩摘要 → 写回 memories

### 第三层：知识记忆

**所有采集文章做 embedding 存入向量库**，支持 RAG 问答。

```sql
CREATE TABLE article_embeddings (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    article_id  INTEGER NOT NULL UNIQUE,
    embedding   TEXT NOT NULL,                      -- JSON 格式存储的向量
    FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE
);
```

**使用场景：** 用户问"之前那篇关于 Go 泛型的文章说了什么"→ 语义检索 → LLM 结合上下文回答。

---

## Agent 工具定义

Eino Graph 模式——LLM 根据用户意图自主选择工具：

| 工具名 | 功能 | 触发场景 |
|--------|------|---------|
| `search_articles` | 关键词搜索文章库 | "找关于 Go 语言的文章" |
| `get_article_detail` | 获取文章全文 | "这篇文章详细说了什么" |
| `generate_briefing` | 触发管线生成简报 | "帮我生成今天的简报" |
| `get_latest_briefing` | 获取最近一期简报 | "今天的简报有什么" |
| `add_bookmark` | 收藏文章 | "收藏这篇" |
| `remove_bookmark` | 取消收藏 | "取消收藏第一篇" |
| `list_bookmarks` | 列出用户收藏 | "我收藏了哪些" |
| `update_preferences` | 更新用户偏好关键词 | "我想关注 Kubernetes" |
| `get_preferences` | 查看当前偏好 | "我的偏好是什么" |
| `analyze_topic` | 分析话题趋势 | "AI 领域最近在讨论什么" |
| `chat_with_docs` | RAG 问答 | "之前有一篇说过 Go 泛型..." |
| `manage_sources` | 管理内容源 | "帮我加一个 RSS 源" |

**Tool 实现原则：** 每个 Tool 复用现有 Service（FetchService、PipelineService 等），不重复写逻辑。

---

## 对话流程示例

```
用户: "最近 AI 领域有什么值得关注的？"
    │
    ▼
[组装 System Prompt]
├── role: 你是内容聚合 Agent
├── profile: {用户偏好关键词: AI, LLM, 大模型...}
├── memory:  {上次用户对 xx 文章感兴趣，偏好深度技术分析...}
└── 当前时间: 2026-05-30

[LLM 决策]
├── 调用 search_articles("AI", limit=20)
├── 分析结果，选出 5 篇最重要的
├── 调用 get_article_detail(id=42) 深入了解关键文章
├── 发现用户的偏好分析缺少 RAG 实践内容
└── 组织回答：推荐列表 + 每篇一句话点评

用户: "第二篇帮我收藏一下"
[LLM 决策]
├── 上下文中有上一轮的搜索结果
├── 调用 add_bookmark(id=第二篇的article_id)
└── 回复"已收藏"

[对话结束后]
├── LLM 生成本轮对话摘要
├── 存入 memories 表（含用户兴趣发现）
└── 提取新偏好 → update_preferences
```

---

## 与现有管线的关系

```
之前的设计                         现在的定位
────────────────────              ────────────────────
Pipeline Engine                   Agent 的一个 Tool
  采集→去重→排序→摘要→组装            由 generate_briefing Tool 触发

┌─────────────────────────────────────┐
│              Agent                   │
│                                      │
│  generate_briefing Tool              │
│        │                             │
│        ▼                             │
│  Pipeline Engine（复用现有代码）       │
│  采集→去重→排序→摘要→组装              │
└─────────────────────────────────────┘
```

---

## 实现路线

### Phase A — LLM 基础（当前 Step 4）

- Eino ChatModel 工厂（OpenAI 兼容 API）
- Ranking Chain + Summary Chain（简洁版，为管线服务）
- Callbacks 日志追踪

### Phase B — 管线跑通（Step 5-6）

- Pipeline Engine
- HTTP API Handler
- 前端面板

### Phase C — Agent 升级（之后）

- conversations / messages / memories 表
- Eino Graph 搭建 Agent
- 全部 Tools 实现
- RAG 知识库（article_embeddings 表 + 向量检索）
- 前端对话界面

---

## 数据库总览（含 Agent 表）

| 表名 | 说明 | 所属阶段 |
|------|------|---------|
| users | 用户账号 | Step 2 |
| preferences | 用户偏好 | Step 2 |
| sources | 内容源配置 | Step 2 |
| articles | 抓取文章 | Step 2 |
| briefings | 简报 | Step 2 |
| briefing_articles | 简报-文章关联 | Step 2 |
| bookmarks | 收藏 | Step 2 |
| fetch_logs | 抓取日志 | Step 2 |
| conversations | 会话 | Phase C |
| messages | 消息 | Phase C |
| memories | 长期记忆 | Phase C |
| article_embeddings | 文章向量 | Phase C |
