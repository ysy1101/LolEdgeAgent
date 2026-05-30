# 用户认证设计

## 当前状态（MVP）

- 启动时自动创建默认 `admin` 用户
- 无鉴权，所有 API 直通
- 数据层已按 `user_id` 隔离（briefings、bookmarks、preferences 等 repository 方法均已接受 `userID` 参数）

## 后续实现路径

### 1. 注册

```
POST /api/v1/auth/register
```

**请求：**
```json
{
  "username": "zhangsan",
  "email": "zhangsan@example.com",
  "password": "123456"
}
```

**后端逻辑：**
1. 校验 username/email 唯一性
2. `bcrypt.GenerateFromPassword` 哈希密码
3. 写入 users 表
4. 自动创建 preferences 记录

---

### 2. 登录

```
POST /api/v1/auth/login
```

**请求：**
```json
{
  "username": "zhangsan",
  "password": "123456"
}
```

**后端逻辑：**
1. 查 users 表
2. `bcrypt.CompareHashAndPassword` 验证
3. 生成 JWT，包含 `{ user_id, username, exp }`
4. 返回 token

**响应：**
```json
{
  "code": 0,
  "data": {
    "token": "eyJhbGciOi...",
    "user": { "id": 1, "username": "zhangsan" }
  }
}
```

---

### 3. 鉴权中间件

`internal/middleware/auth.go` — Gin middleware：

```go
func AuthRequired(jwtSecret string) gin.HandlerFunc {
    return func(c *gin.Context) {
        auth := c.GetHeader("Authorization")
        tokenStr := strings.TrimPrefix(auth, "Bearer ")

        token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
            return []byte(jwtSecret), nil
        })
        if err != nil || !token.Valid {
            c.AbortWithStatusJSON(401, gin.H{"code": 401, "message": "未登录"})
            return
        }

        claims := token.Claims.(jwt.MapClaims)
        c.Set("user_id", uint(claims["user_id"].(float64)))
        c.Next()
    }
}
```

路由分组：
```go
auth := api.Group("")
auth.POST("/auth/register", handler.Register)
auth.POST("/auth/login", handler.Login)

// 以下需要登录
protected := api.Group("", middleware.AuthRequired(secret))
protected.GET("/briefings", ...)
protected.POST("/briefings/generate", ...)
// ...
```

---

### 4. Handler 层获取用户

```go
func (h *BriefingHandler) Generate(c *gin.Context) {
    userID := c.GetUint("user_id")
    briefing, err := h.service.Run(c.Request.Context(), userID)
    // ...
}
```

所有现有 handler 只需加一行 `userID := c.GetUint("user_id")`，传递给已经接受 `userID` 参数的 repository 方法。

---

### 5. 前端

| 页面 | 路由 | 说明 |
|------|------|------|
| Login | /login | 登录表单 |
| Register | /register | 注册表单 |

- token 存 `localStorage`
- axios/fetch 拦截器自动注入 `Authorization: Bearer <token>`
- Router 鉴权：未登录重定向到 `/login`

---

## 依赖

```go
// go.mod 新增
require (
    github.com/golang-jwt/jwt/v5   // JWT 签发与验证
    golang.org/x/crypto             // bcrypt
)
```

---

## 安全要点

- JWT secret 走环境变量 `JWT_SECRET`，不硬编码
- JWT 有效期建议 7 天，可刷新
- 密码最小长度 6 位
- bcrypt cost 用默认值 10
