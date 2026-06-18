# LolEdgeAgent — 内容聚合简报 Agent

基于 **Go + Gin + React** 的智能内容聚合与简报生成 Agent。支持多源内容采集、LLM 管线简报、RAG 知识问答、Agent 对话交互。

---

## 功能概览

| 模块 | 功能 |
|------|------|
| 用户系统 | JWT 注册/登录、偏好配置 |
| 内容源 | RSS、HackerNews、GitHub Trending 三源插件 |
| 简报管线 | 采集 → LLM 排名 → 摘要 → 组装 Markdown |
| Agent 对话 | LLM 自主选工具、多轮对话、上下文截断保护 |
| RAG 问答 | 文章向量化 + 语义搜索 + LLM 回答 |
| 对话持久化 | 历史对话列表、消息存储、切换/删除 |
| 定时调度 | Cron 自动生成简报 |

---

## 架构

```
 ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
 │ 采集层    │→│ 清洗层    │→│ 筛选层    │→│ 摘要层    │→│ 输出层    │
 │ Sources  │ │ Dedup    │ │ Rank     │ │ Summarize│ │ Deliver  │
 │──────────│ │──────────│ │──────────│ │──────────│ │──────────│
 │ plain Go │ │ SHA256   │ │ LLM Chat │ │ LLM Chat │ │ Markdown │
 └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘

 ┌──────────────────────────────────────────────────────┐
 │                Agent 对话                            │
 │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │
 │  │ System Prompt│  │ 工具系统     │  │ 记忆系统    │  │
 │  │ (角色+工具) │  │ (5 Tools)  │  │ (会话持久) │  │
 │  └─────────────┘  └─────────────┘  └─────────────┘  │
 └──────────────────────────────────────────────────────┘
```

---

## 技术栈

| 层 | 技术 |
|----|------|
| HTTP 框架 | Gin |
| ORM | GORM + SQLite（纯 Go 驱动） |
| LLM 调用 | OpenAI 兼容 HTTP Client（DeepSeek/Qwen 等） |
| 认证 | JWT + bcrypt |
| 内容源 | gofeed (RSS) / Firebase API (HN) / goquery (GitHub) |
| 向量检索 | Embedding API + 余弦相似度 |
| 定时调度 | robfig/cron |
| 前端 | React 19 + TypeScript + Tailwind CSS 4 |
| 路由 | react-router v7 |
| 图标 | lucide-react |

---

## 快速开始

```bash
# 后端
cd backend
set LLM_API_KEY=sk-your-key       # DeepSeek API Key
set LLM_BASE_URL=https://api.deepseek.com
go run cmd/server/main.go          # → :8080

# 前端
cd frontend
npm install
npm run dev                         # → :5173
```

Docker：
```bash
docker compose up --build           # 后端 :8080 + 前端 :3000
```

---

## API 设计

Base: `/api/v1`

### 认证
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/auth/register` | 注册 |
| POST | `/auth/login` | 登录 |
| GET | `/auth/verify` | 验证 token |

### 内容源
| 方法 | 路径 | 说明 |
|------|------|------|
| GET/POST | `/sources` | 列表 / 新增 |
| GET/PUT/DELETE | `/sources/:id` | 详情 / 更新 / 删除 |
| POST | `/sources/:id/fetch` | 手动抓取 |

### 简报
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/briefings` | 简报列表 |
| GET | `/briefings/:id` | 简报详情 |
| POST | `/briefings/generate` | 异步生成 |
| DELETE | `/briefings/:id` | 删除 |

### Agent 对话
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/agent/chat` | Agent 对话（多轮工具调用） |
| POST | `/conversations` | 新建对话 |
| GET | `/conversations` | 对话列表 |
| GET | `/conversations/:id/messages` | 历史消息 |
| DELETE | `/conversations/:id` | 删除对话 |

### 其他
| 方法 | 路径 | 说明 |
|------|------|------|
| GET/PUT | `/preferences` | 偏好设置 |
| GET | `/articles` | 已采集文章 |
| POST | `/search` | RAG 搜索 |
| POST | `/ask` | RAG 问答 |

统一响应：
```json
{"code":0, "message":"success", "data":{...}}
```

---

## 数据库

| 表 | 说明 |
|----|------|
| users | 用户账号 |
| preferences | 用户偏好（LLM 配置、关键词、定时） |
| sources | 内容源配置 |
| articles | 抓取的文章 |
| briefings | 简报 |
| briefing_articles | 简报-文章关联 |
| bookmarks | 收藏 |
| fetch_logs | 抓取日志 |
| conversations | Agent 对话 |
| messages | 对话消息 |
| article_embeddings | 文章向量（RAG） |

---

## LLM 配置

支持的模型：DeepSeek（deepseek-chat / deepseek-v4-pro）、Qwen、moonshot 等所有 OpenAI 兼容 API。

配置方式（优先级从高到低）：
1. 环境变量 `LLM_API_KEY` / `LLM_BASE_URL` / `LLM_MODEL`
2. 前端偏好设置页（存入 DB，重启生效）

未配置 LLM 时管线自动降级为模板输出。

---

## 源插件系统

```go
type Plugin interface {
    Name() string
    Fetch(ctx context.Context, source models.Source) ([]models.Article, error)
    Validate(source models.Source) error
}
```

已实现：`rss` / `hackernews` / `github`，各插件通过 `init()` 自注册。

---

## 项目结构

```
loledgeagent/
├── backend/
│   ├── cmd/server/main.go
│   ├── internal/
│   │   ├── agent/        # Agent 引擎 + 工具系统
│   │   ├── config/       # 配置加载
│   │   ├── handler/      # HTTP handlers
│   │   ├── llm/          # LLM Client
│   │   ├── middleware/    # CORS、JWT Auth
│   │   ├── models/       # GORM 模型
│   │   ├── pipeline/     # 简报管线引擎
│   │   ├── repository/   # 数据访问层
│   │   ├── scheduler/    # Cron 调度
│   │   ├── service/      # 业务逻辑
│   │   └── sources/      # 内容源插件
│   └── api/v1/routes.go
├── frontend/
│   └── src/
│       ├── layouts/       # 布局组件
│       ├── pages/         # Chat / Briefings / Sources / Preferences / Login
│       ├── components/    # UI 基础组件
│       ├── lib/api.ts     # API 封装
│       ├── routes/        # 路由定义
│       └── types/         # TypeScript 类型
├── configs/config.yaml
├── docker-compose.yml
└── docs/                  # 设计文档
```

---

## 更新日志

- [x] 项目骨架 + Docker
- [x] 数据层 models + repository
- [x] 源插件系统（RSS + HN + GitHub）
- [x] LLM Client（OpenAI 兼容）
- [x] 管线引擎（采集→排名→摘要→组装）
- [x] HTTP API 全量端点
- [x] 前端页面（对话/简报/源/偏好/登录）
- [x] JWT 用户认证
- [x] 对话持久化（历史列表 + 消息存储）
- [x] RAG 知识库（向量检索 + 问答）
- [x] Agent 引擎（工具系统 + 多轮循环）
- [x] 定时调度器
- [x] Docker 部署

---

## 许可

MIT
