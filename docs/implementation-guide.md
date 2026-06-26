# LolEdgeAgent 实现详解（面试用）

---

## 1. Agent 推理系统

### 在哪

`backend/internal/agent/agent.go` — 全部 289 行

### 怎么实现

**Agent 结构体**（第 46-50 行）：
```go
type Agent struct {
    defaultCfg llm.Config              // 环境变量默认配置
    prefRepo   *repository.PreferenceRepo // DB 用户配置（热加载）
    logger     *slog.Logger
}
```

**ReAct 循环**（第 84-162 行）：
```
for round := 1; round <= maxRounds(8); round++ {
    1. trimContext(msgs)                        // 上下文截断
    2. chatModel.Generate(ctx, msgs, tools)     // 原生 Function Calling
    3. if resp.ToolCalls > 0:
         msgs += assistant(含ToolCalls)         // 记录助手调用
         msgs += tool结果(每对tool_call_id)     // 记录工具结果
         continue                               // 下一轮
    4. else:
         return resp.Content                    // 最终回答
}
```

**三个关键设计**：

a) **原生 Function Calling**（第 90 行）：
```go
resp, err := chatModel.Generate(llmCtx, msgs, model.WithTools(toolInfos))
```
工具 schema 通过 `tool.InvokableTool.Info()` 获得，`model.WithTools()` 传给模型。模型返回 `resp.ToolCalls` 结构化数组（含 id、name、arguments），比 JSON 解析更稳定。

b) **tool_calls 和 tool 消息必须配对**（第 114-143 行）：
```go
// assistant 消息带 ToolCalls
assistantMsg := &schema.Message{Role: assistant, ToolCalls: resp.ToolCalls}
msgs = append(msgs, assistantMsg)

// 每个 tool_call 对应一条 tool 消息
for _, tc := range resp.ToolCalls {
    result := executeToolCall(tc.Name, tc.Arguments, ...)
    msgs = append(msgs, schema.ToolMessage(result, tc.ID)) // content=toolCallID 错了会报400
}
```
DeepSeek API 严格校验：assistant 的 `tool_calls` 必须被相同数量的 `tool` 消息响应，且 `tool_call_id` 匹配。**参数顺序写反会把结果 JSON 当成 tool_call_id，API 直接 400。**

c) **上下文截断不能拆散配对**（第 244-282 行）：
```go
func trimContext(msgs []*schema.Message) []*schema.Message {
    // 从后往前统计，遇到 tool 消息增加 pendingTools 计数
    // 遇到含 ToolCalls 的 assistant 消息减少计数
    // 截断时保证配对完整：不能只保留 assistant(tool_calls) 而扔掉对应 tool 响应
}
```
如果简单按 token 数截断，可能把 assistant 的 tool_calls 和后面的 tool 响应拆到两批，API 校验失败。

d) **超时 + 轮次上限**：
- 单次 Generate 调用 `context.WithTimeout(ctx, 30s)`（第 89 行）
- 循环最多 8 轮（第 84 行 `maxRounds = 8`）
- 防止模型反复调工具死循环

**工具定义**（`backend/internal/agent/tools.go`）：
用 Eino 的 `InferTool` 自动推断 JSON Schema：
```go
type SearchArticlesInput struct {
    Query string `json:"query" jsonschema:"required" jsonschema_description:"搜索关键词"`
    Limit int    `json:"limit" jsonschema_description:"返回数量，默认5"`
}
t, _ := toolutils.InferTool("search_articles", "搜索文章", func(ctx, input) { ... })
```
Go 泛型 + struct tag 自动生成 JSON Schema，编译期类型安全，不用手写 schema 字符串。

**热加载 LLM 配置**（第 184-206 行）：
每次 `Run()` 从 DB 读最新偏好，覆盖环境变量默认值，用户在前端改配置后下次对话立即生效，无需重启。

---

## 2. RAG 语义检索系统

### 在哪

- `backend/internal/service/rag_service.go` — 核心逻辑
- `backend/internal/repository/embeddings.go` — 向量 CRUD
- `backend/internal/llm/provider.go` — embedding 调用（第 128-131 行 Embeddings 方法）
- `backend/internal/llm/callback.go` — Eino callback 监控

### 怎么实现

**索引流程**（`rag_service.go` 第 34-61 行）：
```go
func (s *RAGService) IndexArticle(ctx, article) {
    text := article.Title + " " + article.Description    // 拼标题+描述
    vecs := llmClient.Embeddings(ctx, []string{text})     // 调 embedding API
    vecJSON := json.Marshal(vecs[0])                       // []float64 → JSON string
    embRepo.Upsert(ArticleID, vecJSON)                     // 存 SQLite
}
```

**检索流程**（第 64-113 行）：
```go
func (s *RAGService) Search(ctx, query, topK) {
    queryVec := llmClient.Embeddings(ctx, []string{query})  // 问题向量化
    all := embRepo.GetAll()                                   // 全表加载到内存
    for each (articleID, dbVec) {
        sim := cosineSimilarity(queryVec, dbVec)               // Go 手算余弦
    }
    sort(by sim desc) → topK                                   // 排序取 TopK
    articleRepo.Get(articleIDs)                                // 联表取文章
}
```

**问答链路**（第 116-133 行）：
```go
func (s *RAGService) Ask(ctx, question, topK) {
    articles := Search(question, topK)                    // 检索
    prompt := "文章：{articles}\n问题：{question}"        // 拼 prompt
    answer := llmClient.Chat(ctx, system, prompt)          // LLM 生成
    return answer, articles
}
```

**为什么不做向量数据库**：
- 当前文章量百/千级别，SQLite 全表扫描 + Go 内存余弦计算毫秒级
- Vector DB（Milvus/Pinecone）引入额外基础设施成本，过早优化

**Eino Embedding 组件**（`provider.go` 第 63-68 行）：
```go
emb, err := openaiembed.NewEmbedder(ctx, &openaiembed.EmbeddingConfig{
    APIKey:  cfg.EmbeddingAPIKey,     // 独立 Key（不配则复用 LLM_API_KEY）
    Model:   cfg.EmbeddingModel,      // 独立 Model（默认 text-embedding-3-small）
    BaseURL: cfg.EmbeddingBaseURL,    // 独立 BaseURL（默认 api.openai.com）
})
```
Chat 和 Embedding 的 Key/Model/BaseURL **三者全部独立**，因为 DeepSeek 提供不了 embedding。

**Eino Callback 监控**（`callback.go`）：
```go
func SetupGlobalCallbacks(logger) {
    handler := NewHandlerHelper().
        ChatModel(&ModelCallbackHandler{OnStart/OnEnd/OnError}).
        Embedding(&EmbeddingCallbackHandler{OnStart/OnEnd/OnError}).
        Handler()
    callbacks.AppendGlobalHandlers(handler)  // 全局注册，所有调用自动触发
}
```
每次 Chat/Embedding 调用自动输出耗时、token 用量、输入输出大小，不侵入业务代码。

---

## 3. 技术简报生成流程

### 在哪

- `backend/internal/pipeline/engine.go` — 管线编排（223 行）
- `backend/internal/llm/ranker.go` — LLM 排序
- `backend/internal/llm/summarizer.go` — LLM 摘要 + 组装
- `backend/internal/service/briefing_service.go` — 业务层

### 怎么实现

**主流程**（`engine.go` 第 42-123 行）：

```
1. 加载用户偏好（keywords, max_articles...）
2. LLM 排序：RankArticles(ctx, articles, interests)
   → fallback: defaultRank(均分 0.5)
3. 取 TopN
4. LLM 逐篇摘要：SummarizeBatch(ctx, articles)
   → fallback: 用 article.Description
5. LLM 组装简报：AssembleBriefing(ctx, articles, interests)
   → fallback: templateBriefing(Go 模板拼凑)
6. 存 DB（briefings + briefing_articles 关联表）
```

**每步降级**：
- 排序失败 → 所有文章给 0.5 分，不中断管线
- 摘要失败 → 用原文 description，不卡住
- 组装失败 → Go 模板生成简易 Markdown

**设计原则：管线永不硬失败，始终产出可用的降级结果。**

---

## 4. 插件化数据源架构

### 在哪

- `backend/internal/sources/interface.go` — Plugin 接口定义
- `backend/internal/sources/rss.go` — RSS 实现
- `backend/internal/sources/hackernews.go` — HackerNews 实现
- `backend/internal/sources/github.go` — GitHub Trending 实现
- `backend/internal/service/fetch_service.go` — 并发抓取编排

### 怎么实现

**Plugin 接口**（`interface.go`）：
```go
type Plugin interface {
    Name() string
    Fetch(ctx, source models.Source) ([]models.Article, error)
    Validate(source models.Source) error
}

// 全局注册表
var registry = make(map[string]Plugin)

func Register(p Plugin) {        // 注册
    registry[p.Name()] = p
}
```

**自注册模式**（每个 `xxx.go` 文件）：
```go
func init() {
    sources.Register(&RSSPlugin{})
}
```

**并发抓取 + 错误隔离**（`fetch_service.go`）：
```go
for _, source := range enabledSources {
    go func(src) {
        plugin := registry[src.SourceType]
        articles, err := plugin.Fetch(ctx, src)   // 单个源抓取
        if err != nil {
            log.Warn("fetch failed", "source", src.Name, "error", err)
            return                                  // 不阻断其他源
        }
        repo.BatchCreate(articles)                  // 入库（去重哈希）
    }(source)
}
```
每个源独立 goroutine，一个挂了不影响其他。去重用 `dedup_hash = SHA256(title + url)`。

---

## 5. 全栈实现与部署

### 在哪

- `backend/cmd/server/main.go` — Go 入口
- `backend/api/v1/routes.go` — 路由注册 + 依赖注入
- `frontend/src/pages/Chat.tsx` — 前端对话页
- `docker-compose.yml` — 编排
- `backend/Dockerfile`、`frontend/Dockerfile` — 镜像

### 怎么实现

**后端**：
- Gin 路由挂 `/api/v1` 组，认证用 JWT + bcrypt
- GORM + SQLite（纯 Go 驱动，零依赖）
- `RegisterRoutes(gin, gorm.DB, *slog.Logger)` — 手动依赖注入

**前端消息发送流程**（`Chat.tsx`）：
```
1. DB.Append(user)          → 先持久化用户消息
2. DB.GetMessages()         → 从 DB 取真实历史（不用 React state 快照）
3. agent.Run(q, history)    → 完整历史交给 Agent
4. DB.Append(assistant)     → 助手回复落 DB
5. UI.Append(assistant)     → 更新界面（含 steps 可视化）
```

**Docker**：
- 多阶段构建（Go 编译+运行分离、前端 build+nginx）
- nginx `proxy_pass http://backend:8080/api/` — 反向代理消除 CORS
- `depends_on: condition: service_healthy` — 等后端 healthcheck 过了再启前端
- 本地 `./backend/data` 目录直接挂载，共享 SQLite 文件
