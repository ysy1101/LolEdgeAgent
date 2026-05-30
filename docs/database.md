# 数据库设计

## 技术选型

- **数据库**：SQLite 3
- **ORM**：GORM（`gorm.io/gorm` + `gorm.io/driver/sqlite`）
- **文件位置**：`backend/data/loledgeagent.db`

---

## 表结构总览

| 表名 | 说明 | 关键索引 |
|------|------|---------|
| users | 用户账号 | username 唯一 |
| sources | 内容源配置 | - |
| articles | 抓取的文章 | source_id, dedup_hash, relevance_score |
| briefings | 生成的简报 | user_id, generated_at, status |
| briefing_articles | 简报-文章关联 | 联合主键 |
| bookmarks | 文章收藏 | user_id + article_id 唯一 |
| preferences | 用户偏好 | user_id 唯一 |
| fetch_logs | 抓取日志 | source_id, started_at |

---

## 详细表设计

### 1. users — 用户账号

```sql
CREATE TABLE users (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    username        TEXT    NOT NULL UNIQUE,              -- 用户名
    email           TEXT    NOT NULL UNIQUE,              -- 邮箱
    password_hash   TEXT    NOT NULL,                     -- bcrypt 哈希
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);
```

**说明：** MVP 阶段可以只有一个默认用户，后期扩展多用户时通过 `user_id` 关联各表数据隔离。

---

### 2. sources — 内容源配置

```sql
CREATE TABLE sources (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT    NOT NULL,                     -- 显示名称，如"Hacker News 首页"
    source_type     TEXT    NOT NULL,                     -- 源类型：rss / hackernews / github
    url             TEXT    NOT NULL,                     -- 源地址（RSS URL / API 地址等）
    enabled         INTEGER NOT NULL DEFAULT 1,           -- 是否启用 1=启用 0=禁用
    config_json     TEXT,                                 -- 源特定配置（JSON）
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT    NOT NULL DEFAULT (datetime('now'))
);
```

**config_json 按类型说明：**

| source_type | config_json 示例 | 字段说明 |
|-------------|-----------------|---------|
| rss | `{"max_items": 20}` | max_items: 最大抓取条数 |
| hackernews | `{"story_type": "top", "max_items": 30}` | story_type: top/new/best, max_items: 抓取条数 |
| github | `{"language": "", "since": "daily"}` | language: 语言过滤(空=全部), since: daily/weekly/monthly |

---

### 3. articles — 抓取的文章

```sql
CREATE TABLE articles (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id       INTEGER NOT NULL,                    -- 来源，关联 sources.id
    external_id     TEXT    NOT NULL,                    -- 源站唯一 ID（如 HN id、RSS guid）
    title           TEXT    NOT NULL,                    -- 文章标题
    url             TEXT    NOT NULL,                    -- 原文链接
    description     TEXT,                                -- 摘要/第一段
    content         TEXT,                                -- 正文（提取后的纯文本）
    author          TEXT,                                -- 作者
    published_at    TEXT,                                -- 发布时间
    fetched_at      TEXT    NOT NULL DEFAULT (datetime('now')),  -- 抓取时间
    dedup_hash      TEXT    NOT NULL,                    -- SHA256(title + url) 去重
    relevance_score REAL    NOT NULL DEFAULT 0.0,        -- LLM 相关性打分 0~1
    summary         TEXT,                                -- LLM 生成的 1~3 句摘要

    UNIQUE(source_id, external_id),
    FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE
);

CREATE INDEX idx_articles_source_id    ON articles(source_id);
CREATE INDEX idx_articles_dedup_hash   ON articles(dedup_hash);
CREATE INDEX idx_articles_fetched_at   ON articles(fetched_at);
CREATE INDEX idx_articles_relevance    ON articles(relevance_score DESC);
```

**字段计算逻辑：**

| 字段 | 计算方式 |
|------|---------|
| external_id | RSS: SHA256(link), HN: 数字 ID, GitHub: "github/{owner}/{repo}" |
| dedup_hash | SHA256(title + url) 忽略大小写 |
| relevance_score | 初始 0，管线排名阶段由 LLM 打分填充 |
| summary | 初始 NULL，摘要阶段由 LLM 生成填充 |

---

### 4. briefings — 简报

```sql
CREATE TABLE briefings (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    title           TEXT    NOT NULL,                    -- 如"每日简报 - 2026-05-30"
    content_markdown TEXT   NOT NULL,                    -- 完整 Markdown 内容
    article_count   INTEGER NOT NULL DEFAULT 0,           -- 包含的文章数
    generated_at    TEXT    NOT NULL DEFAULT (datetime('now')),
    status          TEXT    NOT NULL DEFAULT 'pending',   -- pending/generating/completed/failed
    error_message   TEXT                                  -- 失败时的错误信息
);

CREATE INDEX idx_briefings_generated_at ON briefings(generated_at DESC);
CREATE INDEX idx_briefings_status       ON briefings(status);
```

**状态流转：**
```
pending → generating → completed
                    ↘ failed → (手动重试) → pending
```

---

### 5. briefing_articles — 简报-文章关联

```sql
CREATE TABLE briefing_articles (
    briefing_id     INTEGER NOT NULL,                    -- 简报 ID
    article_id      INTEGER NOT NULL,                    -- 文章 ID
    rank_position   INTEGER NOT NULL,                    -- 1-based 排序位置

    PRIMARY KEY (briefing_id, article_id),
    FOREIGN KEY (briefing_id) REFERENCES briefings(id) ON DELETE CASCADE,
    FOREIGN KEY (article_id)  REFERENCES articles(id) ON DELETE CASCADE
);
```

**说明：** 一条简报包含多篇文章（按 relevance_score 排序 top N），rank_position 记录每篇文章在简报中的展示顺序。

---

### 6. bookmarks — 文章收藏

```sql
CREATE TABLE bookmarks (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    article_id  INTEGER NOT NULL,                        -- 收藏的文章
    note        TEXT    NOT NULL DEFAULT '',             -- 用户备注
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),

    UNIQUE(article_id),                                  -- 同一文章只能收藏一次
    FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE
);

CREATE INDEX idx_bookmarks_created_at ON bookmarks(created_at DESC);
```

**说明：** 收藏功能独立于简报，用户可以收藏任意已抓取的文章。删除文章时自动清除对应收藏。

**扩展查询 — 获取收藏列表含文章详情：**
```sql
SELECT a.*, b.note, b.created_at AS bookmarked_at
FROM bookmarks b
JOIN articles a ON b.article_id = a.id
ORDER BY b.created_at DESC;
```

---

### 7. preferences — 用户偏好

```sql
CREATE TABLE preferences (
    id                      INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id                 INTEGER NOT NULL UNIQUE,             -- 关联 users.id，每用户一配置
    keywords                TEXT    NOT NULL DEFAULT '[]',       -- 关注关键词 JSON 数组
    excluded_keywords       TEXT    NOT NULL DEFAULT '[]',       -- 排除关键词 JSON 数组
    max_articles_per_source INTEGER NOT NULL DEFAULT 20,         -- 每个源最大抓取数
    max_briefing_articles   INTEGER NOT NULL DEFAULT 10,         -- 简报包含最大文章数
    llm_provider            TEXT    NOT NULL DEFAULT 'openai',   -- LLM 服务商
    llm_model               TEXT    NOT NULL DEFAULT 'gpt-4.1-mini', -- 模型名
    llm_api_key             TEXT    NOT NULL DEFAULT '',         -- API Key
    llm_base_url            TEXT    NOT NULL DEFAULT '',         -- 自定义 API 地址
    briefing_schedule       TEXT    NOT NULL DEFAULT '',         -- Cron 表达式，空=手动
    updated_at              TEXT    NOT NULL DEFAULT (datetime('now')),

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

**keywords 示例：**
```json
["go", "rust", "llm", "分布式系统", "kubernetes"]
```

---

### 8. fetch_logs — 抓取日志

```sql
CREATE TABLE fetch_logs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id       INTEGER NOT NULL,                    -- 数据源
    status          TEXT    NOT NULL,                    -- 'success' / 'error'
    articles_fetched INTEGER NOT NULL DEFAULT 0,          -- 本次抓取条数
    error_message   TEXT,                                -- 错误信息
    started_at      TEXT    NOT NULL,
    completed_at    TEXT,

    FOREIGN KEY (source_id) REFERENCES sources(id) ON DELETE CASCADE
);

CREATE INDEX idx_fetch_logs_source ON fetch_logs(source_id, started_at DESC);
```

---

## ER 关系图

```
┌──────────┐       ┌──────────────┐       ┌──────────┐
│  users   │       │   articles   │       │ briefings│
│──────────│       │──────────────│       │──────────│
│ id       │──1:1──│ source_id   │       │ id       │
│ username │       │ external_id │       │ title    │
│ email   │       │ title       │──M:N──│ markdown │
│ password │       │ url         │   │   │ status   │
└──────────┘       │ dedup_hash  │   │   │ user_id ──│──┐
     │ 1:1          │ score       │   │   └──────────┘  │
     ├──────────────│ summary     │   │        │        │
     │ 1:N          └──────────────┘   │   briefing_    │
     │                   │ 1:1         │   articles      │
┌──────────┐       ┌──────────────┐   │   ───────────── │
│  sources │       │  bookmarks   │   │   briefing_id   │
│──────────│       │──────────────│   └───article_id    │
│ id       │       │ article_id  │       rank_position │
│ name     │       │ user_id ────│──┐                    │
│ type     │       │ note        │  │    ┌─────────────┐│
│ url      │       └──────────────┘  ├────│ preferences ││
│ enabled  │                          │    │─────────────││
│ config   │       ┌──────────────┐   │    │ user_id ───│┘
└──────────┘       │  fetch_logs  │   │    │ keywords    │
     │             │──────────────│   │    │ llm_*       │
     └──1:N────────│ source_id   │   │    │ schedule    │
                   │ status      │   │    └─────────────┘
                   │ count       │   │
                   │ error       │   │
                   └──────────────┘   │
                    users ────────────┘
                    (1:N briefings,
                     bookmarks,
                     preferences)
```

---

## 常用查询

### 检查文章是否已存在
```sql
SELECT id FROM articles WHERE dedup_hash = ? LIMIT 1;
```

### 获取某源最近抓取时间
```sql
SELECT MAX(completed_at) FROM fetch_logs
WHERE source_id = ? AND status = 'success';
```

### 获取简报详情（含文章列表）
```sql
SELECT b.*, a.*
FROM briefings b
JOIN briefing_articles ba ON b.id = ba.briefing_id
JOIN articles a ON ba.article_id = a.id
WHERE b.id = ?
ORDER BY ba.rank_position ASC;
```

### 获取未评分的新文章（等待 LLM 排名）
```sql
SELECT * FROM articles
WHERE relevance_score = 0.0
ORDER BY fetched_at DESC
LIMIT ?;
```
