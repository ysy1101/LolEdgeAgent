# LLM Agent 推理系统

## 定位

LLM 是系统的"大脑"——理解用户意图、选择工具、组织回答。现有的采集管线、简报生成等退居幕后，成为 Agent 可调用的工具。

---

## Agent 决策循环（ReAct 模式）

```
用户输入
    │
    ▼
┌─────────────────────────────────────────┐
│  1. 组装消息                              │
│  ├─ System Prompt（角色 + 工具列表 + 规则）  │
│  ├─ 对话历史（最近 N 轮）                   │
│  └─ 用户当前消息                            │
│                                          │
│  2. 调用 LLM（30s 超时）                   │
│     → 返回文本                             │
│                                          │
│  3. 解析响应                               │
│     ├─ JSON {"tool":"...","args":"..."}    │
│     │   → 执行工具 → 结果拼回消息 → 回到 2   │
│     └─ 普通文本                             │
│         → 最终回答，结束                     │
│                                          │
│  最多 8 轮，超过则返回超时提示               │
└─────────────────────────────────────────┘
```

---

## System Prompt 设计

```
你是 LolEdgeAgent，一个内容聚合和知识助手。

## 可用工具
- list_briefings: 查看最近生成的简报列表
- get_briefing: 查看某份简报的详细内容
- search_articles: 根据关键词搜索已采集的文章库
- generate_briefing: 生成一份新的内容简报
- get_preferences: 查看当前用户的偏好设置

## 回复格式
需要调用工具时：
{"tool":"<工具名>","args":"<JSON 参数>"}

直接回答时，正常用 Markdown 回复。

## 规则
1. 用户说"查看简报""最近的简报"→ 调用 list_briefings
2. 查看具体简报 → 调用 get_briefing
3. 搜索文章 → 调用 search_articles
4. 生成新简报 → 调用 generate_briefing
5. 闲聊问候不需要工具，直接回答
6. 用中文回复，引用文章时带标题和链接
```

---

## JSON 输出约束

模型返回两种格式之一：

**工具调用：**
```json
{"tool":"list_briefings","args":"{}"}
```

**直接回答：** 正常 Markdown 文本（不做 JSON 包裹）

解析时取第一个 `{` 到最后一个 `}`，尝试反序列化为 `{tool, args}`。解析失败则视为最终回答。

这种 JSON-in-text 方式兼容所有 OpenAI 兼容 API（DeepSeek、Qwen 等），不依赖原生 Function Calling。

---

## Tool 定义（5 个）

用 Eino 的 `InferTool` 自动推断 JSON Schema，类型安全：

```go
tool, _ := toolutils.InferTool[Input, Output](name, desc, handler)
```

| 工具 | 触发场景 | 对应后端 |
|------|---------|---------|
| `search_articles` | "找关于 Go 的文章" | RAGService.Search() |
| `list_briefings` | "最近有哪些简报" | BriefingRepo.List() |
| `get_briefing` | "第二份简报说了什么" | BriefingRepo.GetByID() |
| `generate_briefing` | "帮我生成今天的简报" | BriefingService.GenerateAsync() |
| `get_preferences` | "我的偏好是什么" | PreferenceRepo.Get() |

每个 Tool 通过闭包捕获依赖，`RegisterAllTools()` 注入 repos/services。

---

## 超时与上下文控制

- **单次 LLM 调用超时**：30 秒（`context.WithTimeout`）
- **最大循环轮数**：8 轮
- **工具结果截断**：超过 300 字符截断（避免 token 爆炸）
- **LLM 未配置降级**：返回友好的中文提示

---

## 与现有管线的关系

```
Agent（决策层）
  │
  ├── generate_briefing Tool
  │     └── Pipeline Engine（采集→排序→摘要→组装→保存）
  │
  ├── search_articles Tool
  │     └── RAGService（查询向量化→余弦检索→返回文章）
  │
  └── list_briefings / get_briefing Tool
        └── BriefingRepo（直接查 DB）
```

Agent 不替代管线，而是作为入口层，按需调度现有业务模块。

---

## 前端可视化

Agent 返回 `steps` 字段，前端可折叠显示每一步：

- 🟡 `tool_call` — 模型决定调用什么工具
- 🟢 `tool` — 工具执行结果
- 🔵 `model` — 最终回答

便于调试和面试时演示推理过程。
