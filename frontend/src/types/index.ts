// 源配置
export interface Source {
  id: number;
  name: string;
  source_type: 'rss' | 'hackernews' | 'github' | string;
  url: string;
  enabled: boolean;
  config_json: string;
  created_at: string;
  updated_at: string;
}

// 文章
export interface Article {
  id: number;
  source_id: number;
  external_id: string;
  title: string;
  url: string;
  description: string;
  content: string;
  author: string;
  published_at: string;
  relevance_score: number;
  summary: string;
}

// 简报
export interface Briefing {
  id: number;
  title: string;
  content_markdown: string;
  article_count: number;
  generated_at: string;
  status: 'pending' | 'generating' | 'completed' | 'failed';
  error_message?: string;
  articles?: Article[];
}

// 用户偏好
export interface Preferences {
  keywords: string[];
  excluded_keywords: string[];
  max_articles_per_source: number;
  max_briefing_articles: number;
  llm_provider: string;
  llm_model: string;
  llm_api_key: string;
  llm_base_url: string;
  briefing_schedule: string;
}

// 统一响应
export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

// 分页
export interface Paginated<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}

// 抓取日志
export interface FetchLog {
  id: number;
  source_id: number;
  status: 'success' | 'error';
  articles_fetched: number;
  error_message: string;
  started_at: string;
  completed_at: string;
}

// 健康状态
export interface HealthStatus {
  status: string;
  db: string;
  llm: string;
}
