# LolEdgeAgent — 内容聚合简报 Agent

基于 **Go + Gin + Eino + React** 的智能内容聚合与简报生成 Agent。自动从多源采集内容，通过 LLM 筛选、摘要、排版，最终输出可读性高的 Markdown 简报。

---

## 核心流程

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  采集层   │ →  │  清洗层   │ →  │  筛选层   │ →  │  摘要层   │ →  │  排版层   │ →  │  输出层   │
│ Sources  │    │ Dedup    │    │ Rank     │    │ Summarize│    │ Assemble │    │ Deliver  │
│──────────│    │──────────│    │──────────│    │──────────│    │──────────│    │──────────│
│ plain Go │    │ hashing  │    │ Eino LLM │    │ Eino LLM │    │ Eino LLM │    │ Markdown │
└──────────┘    └──────────┘    └──────────┘    └──────────┘    └──────────┘    └──────────┘
     │               │               │               │               │               │
 RSS/HN/GitHub   SHA256去重    关键词+LLM打分   批量 LLM 摘要   组装 Markdown   保存+推送+Web
```

### 各层说明

| 阶段 | 技术手段 | 说明 |
|------|---------|------|
| **采集** | 可插拔 Source Plugin | 统一接口，支持 RSS、HackerNews、GitHub Trending 等，易扩展 |
| **清洗** | 正文提取 + 去重 | `gofeed` 解析结构化字段，`SHA256(title+url)` 哈希去重 |
| **筛选** | 关键词匹配 + LLM 打分 | 先用规则粗筛（关键词+热度+新鲜度），再 Eino Chain 精筛 |
| **摘要** | Eino Chain 批量 LLM | 每篇文章生成 1-3 句摘要，可并行 |
| **排版** | Eino Chain 组装 | LLM 根据 ranked articles + 用户偏好组装 Markdown 简报 |
| **输出** | 保存 + Web 展示 | 存入 SQLite，通过 React 面板查看历史简报 |

---

## 技术栈

| 层 | 技术 | 说明 |
|----|------|------|
| **HTTP 框架** | [Gin](https://github.com/gin-gonic/gin) | REST API 路由、中间件 |
| **AI Agent** | [Eino](https://github.com/cloudwego/eino) | LLM 调用编排（Chain）、可观测（Callbacks） |
| **ORM** | [GORM](https://gorm.io/) + SQLite | 数据持久化，零配置 |
| **RSS 解析** | [gofeed](https://github.com/mmcdole/gofeed) | RSS/Atom 解析 |
| **网页抓取** | [goquery](https://github.com/PuerkitoBio/goquery) | GitHub Trending HTML 解析 |
| **定时调度** | [cron](https://github.com/robfig/cron) | 定时自动生成简报 |
| **前端** | React 19 + TypeScript + Tailwind CSS 3 | 仪表盘 |
| **Markdown 渲染** | react-markdown + remark-gfm | 简报内容展示 |

### 为什么 Eino 只用于 LLM 环节？

- Eino 的 `Chain` 适合**线性 LLM 调用链路**（模板 → 模型 → 解析），我们的 ranking、summary、assembly 正是这个场景
- Eino 的 `Tool` / `Graph` 服务于 **LLM 自主决策调用工具**的场景，内容采集是确定性流程，不需要 LLM 决策
- **原则：LLM 只用在刀刃上** — 采集、清洗、去重用纯 Go 实现，只在排序和摘要环节调用 LLM

---

## 目录结构

```
loledgeagent/
├── backend/
│   ├── cmd/server/main.go              # 入口：依赖注入、启动 Gin
│   ├── internal/
│   │   ├── config/config.go            # Viper 配置加载
│   │   ├── models/models.go            # GORM 数据模型
│   │   ├── repository/
│   │   │   ├── sources.go              # 源 CRUD
│   │   │   ├── articles.go             # 文章 CRUD + 去重查询
│   │   │   ├── briefings.go            # 简报 CRUD + 联表
│   │   │   ├── preferences.go          # 偏好设置（单例）
│   │   │   └── fetch_logs.go           # 抓取日志
│   │   ├── service/
│   │   │   ├── source_service.go       # 源验证 + CRUD 逻辑
│   │   │   ├── fetch_service.go        # 多源抓取编排
│   │   │   ├── pipeline_service.go     # 管线主流程调用
│   │   │   ├── briefing_service.go     # 简报 CRUD 逻辑
│   │   │   └── preference_service.go   # 偏好管理
│   │   ├── handler/
│   │   │   ├── source_handler.go       # /api/v1/sources 相关
│   │   │   ├── briefing_handler.go     # /api/v1/briefings 相关
│   │   │   ├── article_handler.go      # /api/v1/articles 相关
│   │   │   ├── preference_handler.go   # /api/v1/preferences 相关
│   │   │   └── health_handler.go       # /api/v1/health
│   │   ├── pipeline/
│   │   │   ├── engine.go               # 管线编排器（核心）
│   │   │   ├── dedup.go                # 标题+URL 哈希去重
│   │   │   ├── ranking.go              # LLM 相关性打分
│   │   │   └── summarizer.go           # LLM 摘要生成
│   │   ├── sources/
│   │   │   ├── interface.go            # Plugin 接口 + 全局注册表
│   │   │   ├── rss.go                  # RSS 源（gofeed）
│   │   │   ├── hackernews.go           # HN 源（Firebase API）
│   │   │   └── github.go              # GitHub Trending（goquery）
│   │   ├── llm/
│   │   │   ├── provider.go             # Eino ChatModel 工厂
│   │   │   ├── chains.go               # Ranking、Summary、Assembly 三条 Chain
│   │   │   └── callback.go             # 日志/追踪回调
│   │   ├── scheduler/
│   │   │   └── scheduler.go            # Cron 定时任务
│   │   └── middleware/
│   │       └── middleware.go           # CORS、日志、恢复
│   ├── api/v1/routes.go                # 路由注册
│   └── pkg/response/response.go        # 统一响应格式
├── frontend/
│   ├── src/
│   │   ├── layouts/DashboardLayout.tsx
│   │   ├── pages/
│   │   │   ├── Briefings/BriefingList.tsx & BriefingDetail.tsx
│   │   │   ├── Sources/SourceList.tsx & SourceForm.tsx
│   │   │   └── Preferences.tsx
│   │   ├── components/
│   │   │   ├── ui/                     # 基础组件（Button, Card, Dialog...）
│   │   │   ├── layout/                 # Sidebar, Header
│   │   │   └── features/               # 业务组件（BriefingCard, MarkdownViewer...）
│   │   ├── hooks/                      # 自定义 hooks
│   │   ├── lib/api.ts                  # API 调用封装
│   │   └── types/index.ts             # TypeScript 类型定义
│   └── (vite、tailwind 配置)
├── configs/config.yaml                 # 默认配置
├── README.md
└── .gitignore
```

---

## API 设计

Base URL: `http://localhost:8080/api/v1`

统一响应格式：
```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

### 源管理 `/sources`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/sources` | 列表（可选 `?enabled=true`） |
| POST | `/sources` | 新增源 |
| GET | `/sources/:id` | 详情 |
| PUT | `/sources/:id` | 更新 |
| DELETE | `/sources/:id` | 删除 |
| POST | `/sources/:id/fetch` | 手动触发单个源抓取 |

### 简报 `/briefings`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/briefings?page=1&limit=20` | 简报列表 |
| GET | `/briefings/:id` | 简报详情（含文章列表） |
| POST | `/briefings/generate` | 触发生成 → 返回 `202` |
| DELETE | `/briefings/:id` | 删除简报 |

### 文章 `/articles`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/articles?source_id=&page=&limit=` | 文章列表 |
| POST | `/articles/fetch` | 从所有启用的源抓取 |

### 偏好设置 `/preferences`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/preferences` | 获取偏好（单例） |
| PUT | `/preferences` | 全量更新偏好 |

### 健康检查 `/health`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | DB + LLM 连通性检查 |

---

## 数据库设计

### SQLite 表结构（6 张表）

```sql
-- 内容源配置
sources (
    id, name, source_type, url, enabled, config_json,
    created_at, updated_at
)

-- 抓取的文章
articles (
    id, source_id(FK), external_id, title, url, description,
    content, author, published_at, fetched_at,
    dedup_hash UNIQUE, relevance_score, summary
)

-- 生成的简报
briefings (
    id, title, content_markdown, article_count,
    generated_at, status, error_message
)

-- 简报-文章关联（含排序位置）
briefing_articles (
    briefing_id(FK), article_id(FK), rank_position,
    PRIMARY KEY (briefing_id, article_id)
)

-- 用户偏好（单例，id=1）
preferences (
    id CHECK(id=1), keywords(JSON), excluded_keywords(JSON),
    max_articles_per_source, max_briefing_articles,
    llm_provider, llm_model, llm_api_key, llm_base_url,
    briefing_schedule(cron), updated_at
)

-- 抓取日志
fetch_logs (
    id, source_id(FK), status, articles_fetched,
    error_message, started_at, completed_at
)
```

---

## 内容源插件系统

```go
// internal/sources/interface.go

type Plugin interface {
    Name() string
    Fetch(ctx context.Context, source models.Source) ([]models.Article, error)
    Validate(source models.Source) error
}
```

每个源类型实现 `Plugin` 接口并通过 `init()` 自注册到全局 registry。新增源只需实现接口即可插拔。

**MVP 源类型：**
- `rss` — 通用 RSS/Atom 订阅
- `hackernews` — HN 热门/最新/最高分
- `github` — GitHub Trending（按语言/时间范围）

---

## Eino LLM 集成

三条 Chain，均使用 `compose.NewChain` 构建：

| Chain | 输入 | 输出 |
|-------|------|------|
| **RankingChain** | user_keywords + article_list(JSON) | `[{id, score, rationale}]` 打分排序 |
| **SummaryChain** | title + content | 1-3 句中/英文摘要 |
| **AssemblyChain** | ranked_articles + user_interests | 完整 Markdown 简报 |

通过 `callbacks.HandlerBuilder` 记录每次 LLM 调用的延迟、token、状态。

ChatModel 工厂支持所有 OpenAI 兼容 API（DeepSeek、Qwen、moonshot 等），只需配置 `base_url` 和 `api_key`。

---

## React 前端

### 路由结构

```
/  → 重定向到 /briefings
├── /briefings          → BriefingList      （简报列表 + 生成按钮）
├── /briefings/:id      → BriefingDetail    （Markdown 渲染简报详情）
├── /sources            → SourceList         （源列表 + 新增/编辑弹窗）
├── /preferences        → Preferences        （关键词、LLM、定时配置）
```

### 核心交互

- **生成简报** — 点击"生成简报"按钮 → POST `/briefings/generate` → 202 → 轮询状态 → 完成后跳转详情
- **管理源** — 表格展示所有源，弹窗表单新增/编辑，支持启用/禁用切换和手动抓取
- **偏好设置** — 关键词标签输入、LLM provider/model/api_key 配置、定时 cron 表达式

### 技术选型

- **UI 组件** — Tailwind CSS + 手写基础组件（Button、Card、Dialog 等），不引入重型组件库
- **图标** — lucide-react
- **提示** — react-hot-toast
- **路由** — react-router v7

---

## 开发路线图

### Phase 1 — MVP（核心跑通）
- [ ] 项目骨架（Go module + React + Tailwind + 目录结构）
- [ ] 数据层（GORM models + SQLite + repository CRUD）
- [ ] 源插件系统（Plugin 接口 + RSS 源）
- [ ] Eino LLM 集成（ChatModel + 三条 Chain）
- [ ] 管线引擎（fetch → dedup → rank → summarize → assemble）
- [ ] HTTP API（Gin 全量端点）
- [ ] 前端页面（BriefingList/Detail、SourceList/Form、Preferences）

### Phase 2 — 完善
- [ ] HN + GitHub 源
- [ ] Cron 定时调度
- [ ] 抓取日志与监控

### Phase 3 — 进阶
- [ ] RAG 问答（基于历史简报语义搜索）
- [ ] Docker 部署
- [ ] 更多推送渠道（飞书/Slack 通知）

---

## 快速开始（开发中）

```bash
# 后端
cd backend
go run cmd/server/main.go

# 前端
cd frontend
npm install
npm run dev
```

---

## 许可

MIT
