package models

import "time"

// User 用户账号
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"not null;uniqueIndex" json:"username"`
	Email        string    `gorm:"not null;uniqueIndex" json:"email"`
	PasswordHash string    `gorm:"not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (User) TableName() string { return "users" }

// Source 内容源配置
type Source struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Name       string    `gorm:"not null" json:"name"`
	SourceType string    `gorm:"not null" json:"source_type"` // rss, hackernews, github
	URL        string    `gorm:"not null" json:"url"`
	Enabled    bool      `gorm:"not null;default:true" json:"enabled"`
	ConfigJSON string    `json:"config_json"` // 源特定 JSON 配置
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (Source) TableName() string { return "sources" }

// Article 抓取的文章
type Article struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	SourceID       uint      `gorm:"not null;uniqueIndex:idx_source_external" json:"source_id"`
	ExternalID     string    `gorm:"not null;uniqueIndex:idx_source_external" json:"external_id"`
	Title          string    `gorm:"not null" json:"title"`
	URL            string    `gorm:"not null" json:"url"`
	Description    string    `json:"description"`
	Content        string    `json:"content"`
	Author         string    `json:"author"`
	PublishedAt    *time.Time `json:"published_at"`
	FetchedAt      time.Time `gorm:"not null" json:"fetched_at"`
	DedupHash      string    `gorm:"not null;index" json:"dedup_hash"`
	RelevanceScore float64   `gorm:"not null;default:0;index" json:"relevance_score"`
	Summary        string    `json:"summary"`
}

func (Article) TableName() string { return "articles" }

// Briefing 生成的简报
type Briefing struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserID          uint      `gorm:"not null;index" json:"user_id"`
	Title           string    `gorm:"not null" json:"title"`
	ContentMarkdown string    `gorm:"not null" json:"content_markdown"`
	ArticleCount    int       `gorm:"not null;default:0" json:"article_count"`
	GeneratedAt     time.Time `gorm:"not null;index" json:"generated_at"`
	Status          string    `gorm:"not null;default:pending;index" json:"status"`
	ErrorMessage    string    `json:"error_message"`
}

func (Briefing) TableName() string { return "briefings" }

// BriefingArticle 简报-文章关联
type BriefingArticle struct {
	BriefingID   uint `gorm:"primaryKey" json:"briefing_id"`
	ArticleID    uint `gorm:"primaryKey" json:"article_id"`
	RankPosition int  `gorm:"not null" json:"rank_position"`
}

func (BriefingArticle) TableName() string { return "briefing_articles" }

// Bookmark 文章收藏
type Bookmark struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_user_article" json:"user_id"`
	ArticleID uint      `gorm:"not null;uniqueIndex:idx_user_article" json:"article_id"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

func (Bookmark) TableName() string { return "bookmarks" }

// Preference 用户偏好
type Preference struct {
	ID                     uint      `gorm:"primaryKey" json:"id"`
	UserID                 uint      `gorm:"not null;uniqueIndex" json:"user_id"`
	Keywords               string    `gorm:"not null;default:'[]'" json:"keywords"` // JSON 数组
	ExcludedKeywords       string    `gorm:"not null;default:'[]'" json:"excluded_keywords"`        // JSON 数组
	MaxArticlesPerSource   int       `gorm:"not null;default:20" json:"max_articles_per_source"`
	MaxBriefingArticles    int       `gorm:"not null;default:10" json:"max_briefing_articles"`
	LLMProvider            string    `gorm:"not null;default:deepseek" json:"llm_provider"`
	LLMModel               string    `gorm:"not null;default:deepseek-chat" json:"llm_model"`
	LLMAPIKey              string    `json:"llm_api_key"`
	LLMBaseURL             string    `json:"llm_base_url"`
	BriefingSchedule       string    `json:"briefing_schedule"` // cron 表达式，空=手动
	UpdatedAt              time.Time `json:"updated_at"`
}

func (Preference) TableName() string { return "preferences" }

// FetchLog 抓取日志
type FetchLog struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	SourceID        uint       `gorm:"not null;index:idx_fl_source_date" json:"source_id"`
	Status          string     `gorm:"not null" json:"status"` // success / error
	ArticlesFetched int        `gorm:"not null;default:0" json:"articles_fetched"`
	ErrorMessage    string     `json:"error_message"`
	StartedAt       time.Time  `gorm:"not null;index:idx_fl_source_date" json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at"`
}

// Conversation 对话会话
type Conversation struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	Title     string    `gorm:"not null;default:'新对话'" json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Conversation) TableName() string { return "conversations" }

// ChatMessage 对话消息
type ChatMessage struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ConversationID uint      `gorm:"not null;index" json:"conversation_id"`
	Role           string    `gorm:"not null" json:"role"` // user / assistant
	Content        string    `gorm:"not null" json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

func (ChatMessage) TableName() string { return "messages" }

// ArticleEmbedding 文章向量
type ArticleEmbedding struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ArticleID uint   `gorm:"not null;uniqueIndex" json:"article_id"`
	Embedding string `gorm:"not null" json:"embedding"` // JSON 向量数组
}

func (ArticleEmbedding) TableName() string { return "article_embeddings" }

// Memory 长期记忆（LLM 压缩摘要）
type Memory struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"not null;index" json:"user_id"`
	Content     string    `gorm:"not null" json:"content"`      // LLM 压缩后的摘要
	Keywords    string    `gorm:"not null;default:'[]'" json:"keywords"` // JSON 关键词数组
	Importance  float64   `gorm:"not null;default:0.5" json:"importance"` // 重要性 0~1
	AccessCount int       `gorm:"not null;default:0" json:"access_count"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Memory) TableName() string { return "memories" }

// MemoryEmbedding 记忆向量
type MemoryEmbedding struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	MemoryID  uint   `gorm:"not null;uniqueIndex" json:"memory_id"`
	Embedding string `gorm:"not null" json:"embedding"` // JSON 向量数组
}

func (MemoryEmbedding) TableName() string { return "memory_embeddings" }

func (FetchLog) TableName() string { return "fetch_logs" }
