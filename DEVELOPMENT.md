# 开发实现文档

每个 Step 需要独立完成并通过验证后再进入下一步。

---

## Step 1 — 项目骨架搭建

**目标：** 搭建 Go 后端和 React 前端的完整目录结构，安装依赖，确保能启动。

### 1.1 Go 后端初始化

```
backend/
├── cmd/server/main.go        # 入口，一个简单可运行的 Gin server
├── internal/                  # 内部包
│   ├── config/config.go
│   ├── models/models.go
│   ├── repository/
│   ├── service/
│   ├── handler/
│   ├── pipeline/
│   ├── sources/
│   ├── llm/
│   ├── scheduler/
│   └── middleware/
├── api/v1/routes.go
├── pkg/response/response.go
├── go.mod
├── go.sum
└── data/                     # SQLite 数据库文件存放
```

**go.mod 依赖：**
```
module loledgeagent

go 1.22

require (
    github.com/gin-gonic/gin        # HTTP 框架
    github.com/cloudwego/eino        # AI Agent 编排
    gorm.io/gorm                    # ORM
    gorm.io/driver/sqlite           # SQLite 驱动
    github.com/mmcdole/gofeed/v2    # RSS 解析
    github.com/robfig/cron/v3       # 定时调度
    github.com/joho/godotenv        # 环境变量
)
```

**验证：** `go run cmd/server/main.go` 能启动并收到 `GET /health` 返回 200

### 1.2 React 前端初始化

用 Vite 创建 React + TypeScript 项目，安装 Tailwind CSS 3。

```
frontend/
├── src/
│   ├── main.tsx
│   ├── index.css
│   ├── layouts/
│   ├── pages/
│   ├── components/
│   ├── hooks/
│   ├── lib/
│   └── types/
├── index.html
├── package.json
├── tsconfig.json
├── vite.config.ts
└── tailwind.config.js
```

**npm 依赖：**
```
react, react-dom, react-router, react-markdown, react-hot-toast, lucide-react
```

**验证：** `npm run dev` 能启动 Vite dev server，页面显示 "LolEdgeAgent"

### 1.3 配置文件模板

```
configs/config.yaml         # 默认配置
backend/.env.example        # 环境变量模板
```

**验证：** 目录结构一致，两个 server 都能启动

### 1.4 Docker 部署

根目录下创建以下文件：

```
loledgeagent/
├── backend/Dockerfile          # Go 后端多阶段构建
├── frontend/Dockerfile         # React 前端构建 + nginx
├── docker-compose.yml          # 编排 backend + frontend
└── .dockerignore
```

**backend/Dockerfile** — 多阶段构建：
```dockerfile
# 阶段1：编译
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server cmd/server/main.go

# 阶段2：运行
FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/server .
COPY configs/ ./configs/
RUN mkdir -p data
EXPOSE 8080
CMD ["./server"]
```

**docker-compose.yml**：
```yaml
services:
  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    volumes:
      - ./backend/data:/app/data       # SQLite 数据持久化
      - ./configs:/app/configs         # 配置文件挂载
    environment:
      - LLM_API_KEY=${LLM_API_KEY}
    restart: unless-stopped

  
  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    ports:
      - "3000:80"
    depends_on:
      - backend
    restart: unless-stopped
```

**.dockerignore**：
```
**/node_modules
**/.git
**/data/*.db
**/.env
```

**验证：** `docker compose up` 后 `curl http://localhost:8080/api/v1/health` 返回 200，浏览器访问 `http://localhost:3000` 能看到前端页面

---

## Step 2 — 数据层 models + repository + SQLite

**目标：** 定义 GORM 数据模型，实现 repository CRUD，数据库表自动创建。

### 数据模型（`internal/models/models.go`）

6 张表对应的 struct：

| struct | 表名 | 关键字段 |
|--------|------|---------|
| Source | sources | name, source_type, url, enabled, config_json, created_at, updated_at |
| Article | articles | source_id, external_id, title, url, description, content, author, published_at, fetched_at, dedup_hash, relevance_score, summary |
| Briefing | briefings | title, content_markdown, article_count, generated_at, status, error_message |
| BriefingArticle | briefing_articles | briefing_id, article_id, rank_position |
| Preferences | preferences | id=1(单例), keywords(JSON), excluded_keywords(JSON), max_articles_per_source, max_briefing_articles, llm_provider, llm_model, llm_api_key, llm_base_url, briefing_schedule, updated_at |
| FetchLog | fetch_logs | source_id, status, articles_fetched, error_message, started_at, completed_at |

**验证：** GORM AutoMigrate 后 SQLite 生成正确的表结构

### Repository CRUD

每个 repository 文件实现对应表的 CRUD：

| 文件 | 关键方法 |
|------|---------|
| repository/sources.go | List, Get, Create, Update, Delete |
| repository/articles.go | Create, BatchCreate, FindByHash, List |
| repository/briefings.go | Create, List, GetByID（含联表 articles）, Delete, UpdateStatus |
| repository/preferences.go | Get, Update |
| repository/fetch_logs.go | Create, ListBySource |

**验证：** 写一个简单的 Go test 或通过 main 里临时代码验证 CRUD 正确

---

## Step 3 — 内容源插件系统

**目标：** 定义 Plugin 接口 + 注册表，实现 RSS 源，FetchService 驱动多源抓取。

### 3.1 Plugin 接口（`internal/sources/interface.go`）

```go
type Plugin interface {
    Name() string
    Fetch(ctx context.Context, source models.Source) ([]models.Article, error)
    Validate(source models.Source) error
}
```

全局 registry，`init()` 自注册模式。

### 3.2 RSS 源实现（`internal/sources/rss.go`）

- 用 `gofeed` 解析 RSS/Atom URL
- 将 feed items 映射为 `[]models.Article`
- `ExternalID` = SHA256(url)
- `DedupHash` = SHA256(title + url)
- `PublishedAt` 解析时间，失败则用当前时间

### 3.3 FetchService（`internal/service/fetch_service.go`）

- 遍历所有 enabled sources
- 根据 `source_type` 找到对应 Plugin
- 调用 `plugin.Fetch()` 获取文章
- 批量写入 articles 表（跳过重复的 dedup_hash）
- 记录 fetch_log
- 单个源失败不影响其他源

**验证：** 在 Source 表里插入一条 RSS 源记录（如 `https://www.ithome.com/rss/`），手动调 FetchService，确认 articles 表有新数据

---

## Step 4 — LLM 集成 Eino

**目标：** 创建 Eino ChatModel 工厂，实现 Ranking、Summary、Assembly 三条 Chain。

### 4.1 ChatModel 工厂（`internal/llm/provider.go`）

- 支持 OpenAI 兼容 API（通过 base_url 适配 DeepSeek、Qwen 等）
- 从 Preferences 读取 provider、model、api_key、base_url
- 返回 `model.ChatModel` 实例

### 4.2 Ranking Chain（`internal/llm/chains.go`）

```
输入：user_keywords + article_list(JSON)
输出：JSON 数组 [{id, title, score, rationale}]
```

系统 prompt 说明打分规则（关键词匹配、专业性、新鲜度），只返回 JSON。

### 4.3 Summary Chain（`internal/llm/chains.go`）

```
输入：title + content
输出：1-3 句中/英文摘要
```

### 4.4 Assembly Chain（`internal/llm/chains.go`）

```
输入：ranked_articles + user_interests
输出：完整 Markdown 简报（标题、分类、原文链接）
```

### 4.5 Callbacks（`internal/llm/callback.go`）

- OnStart: 记录开始时间
- OnEnd: 记录耗时和 token
- OnError: 记录错误

**验证：** 写一个简单测试，传几篇假文章，验证 Chain 能正确调用 LLM 并返回预期格式的结果

---

## Step 5 — 管线引擎

**目标：** 实现 PipelineEngine，编排完整流程。

### 5.1 Dedup 引擎（`internal/pipeline/dedup.go`）

- 按 `dedup_hash` 去重（同一批内）
- 与数据库已存在的 dedup_hash 去重（历史去重）

### 5.2 PipelineEngine（`internal/pipeline/engine.go`）

主流程：
```
1. FetchAll()         → rawArticles
2. Deduplicate()      → unique
3. UpsertBatch()      → 入库
4. Load preferences   → keywords, limits
5. RankChain.Invoke() → ranked (带 relevance_score)
6. 取 top N           → 对每篇调 SummaryChain.Invoke()（可并发）
7. AssemblyChain.Invoke() → markdown
8. Save briefing      → {title, content_markdown, status: completed}
```

容错：
- 某源 fetch 失败 → 记录日志继续
- Ranking 失败 → fallback 关键词匹配打分
- Summary 失败 → 用 description 作为摘要
- Assembly 失败 → 简单拼接模板

### 5.3 PipelineService（`internal/service/pipeline_service.go`）

- 暴露 `Run(ctx) (*models.Briefing, error)`
- 管理 briefing status（pending → generating → completed/failed）
- 异步执行（goroutine），返回 `briefing_id`

**验证：** 配置好 RSS 源和 LLM，手动调用 PipelineService.Run()，确认 SQLite 中生成一条 briefing 且 content_markdown 不为空

---

## Step 6 — HTTP API（Gin handlers）

**目标：** 实现全部 REST 端点。

### 6.1 统一响应格式（`pkg/response/response.go`）

```json
{"code": 0, "message": "success", "data": {...}}
```

### 6.2 中间件（`internal/middleware/middleware.go`）

- CORS：允许前端 Vite 开发域
- Logger：请求日志
- Recovery：panic 恢复

### 6.3 Handler 列表

| Handler | 端点 |
|---------|------|
| health_handler.go | GET /api/v1/health |
| source_handler.go | GET/POST /api/v1/sources, GET/PUT/DELETE /api/v1/sources/:id, POST /api/v1/sources/:id/fetch |
| briefing_handler.go | GET /api/v1/briefings, GET/DELETE /api/v1/briefings/:id, POST /api/v1/briefings/generate |
| article_handler.go | GET /api/v1/articles, POST /api/v1/articles/fetch |
| preference_handler.go | GET/PUT /api/v1/preferences |

### 6.4 路由注册（`api/v1/routes.go`）

所有路由挂载在 `/api/v1` 组下。

**验证：** 用 curl 或 Postman 逐个测试所有端点，确认 CRUD 和 generate 流程完整可用

---

## Step 7 — 前端 React 页面

**目标：** 实现完整的前端仪表盘。

### 7.1 布局与路由

- `DashboardLayout.tsx`：侧边栏 + 顶栏 + `<Outlet />`
- `Sidebar.tsx`：导航链接 + 生成按钮
- `Header.tsx`：页面标题 + 上次生成时间

### 7.2 页面

| 页面 | 路由 | 功能 |
|------|------|------|
| BriefingList | /briefings | 简报卡片列表 + 空状态 + 生成按钮 |
| BriefingDetail | /briefings/:id | Markdown 渲染 + 来源文章列表 |
| SourceList | /sources | 源表格 + 新增/编辑弹窗 + 启用切换 + 删除确认 |
| Preferences | /preferences | 关键词标签 + LLM 配置 + 定时 cron |

### 7.3 核心组件

| 组件 | 说明 |
|------|------|
| MarkdownViewer | react-markdown + remark-gfm 渲染 |
| GenerateButton | 调用 POST /briefings/generate，显示 loading，完成后跳转 |
| SourceForm | Dialog 弹窗，动态字段（类型不同时显示不同配置项） |
| KeywordTag | 标签输入框，支持添加/删除 |

### 7.4 API 封装（`lib/api.ts`）

对所有端点做类型安全的 fetch 封装，自动处理 JSON 解析和错误。

### 7.5 状态管理

用自定义 hooks（`useBriefings`, `useSources`, `usePreferences`）管理 API 调用和本地状态，不引入状态库。

**验证：** 浏览器端完整走通：添加源 → 抓取 → 生成简报 → 查看 Markdown → 修改偏好

---

## Step 8 — 调度与更多源

**目标：** Cron 定时自动生成，扩展 HackerNews + GitHub 源。

### 8.1 Cron 调度器（`internal/scheduler/scheduler.go`）

- 使用 robfig/cron，启动时读取 Preferences 的 `briefing_schedule`
- 到时间自动调用 `PipelineService.Run()`
- Preferences 更新时可重新加载 cron

### 8.2 HackerNews 源（`internal/sources/hackernews.go`）

- 调用 `https://hacker-news.firebaseio.com/v0/topstories.json` 获取 ID 列表
- 取前 N 个 ID，并发获取 `/v0/item/{id}.json`
- 映射为 Article

### 8.3 GitHub Trending 源（`internal/sources/github.go`）

- 用 goquery 解析 `https://github.com/trending`
- 提取 repo 名称、描述、语言、今日 star 数
- 映射为 Article

**验证：** 添加 HN / GitHub 源 → 抓取 → 生成简报，内容正确；设置 cron 每分钟执行一次 → 观察是否自动生成

---

## Step 9 — 高级功能

**目标：** Docker 化、RAG 问答、后续扩展。

### 9.1 Docker 部署

- `Dockerfile`：多阶段构建（Go build + 运行）
- `docker-compose.yml`：backend + frontend（或前端单独 nginx）

### 9.2 RAG 问答

- 文章嵌入：用 Eino Retriever 对 articles 建索引
- `/api/v1/qa` 端点：用户提问 → 检索相关文章 → LLM 结合上下文回答
- 前端：在 BriefingDetail 页面加一个聊天输入框

### 9.3 推送渠道（可选）

- 飞书 Webhook
- Slack Webhook
- 邮件发送

---

## 进度记录

| Step | 状态 | 开始时间 | 完成时间 |
|------|------|---------|---------|
| Step 0 — README 文档 | ✅ 已完成 | | 2026-05-29 |
| Step 1 — 项目骨架 | ✅ 已完成 | 2026-05-30 | 2026-05-30 |
| Step 2 — 数据层 | ⏳ 待开始 | | |
| Step 3 — 插件系统 | ⏳ 待开始 | | |
| Step 4 — LLM 集成 | ⏳ 待开始 | | |
| Step 5 — 管线引擎 | ⏳ 待开始 | | |
| Step 6 — HTTP API | ⏳ 待开始 | | |
| Step 7 — 前端页面 | ⏳ 待开始 | | |
| Step 8 — 调度 + 源 | ⏳ 待开始 | | |
| Step 9 — 高级功能 | ⏳ 待开始 | | |
