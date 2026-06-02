import type { Source, Briefing, Article, Preferences, Paginated } from '../types';

const BASE_URL = '/api/v1';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    headers: { 'Content-Type': 'application/json', ...options?.headers },
    ...options,
  });
  const json: ApiResponse<T> = await res.json();
  if (!res.ok || json.code !== 0) {
    throw new Error(json.message || `HTTP ${res.status}`);
  }
  return json.data;
}

export const api = {
  // 健康检查
  health: () => request<{ status: string; db: string; llm: string }>('/health'),

  // 源管理
  sources: {
    list: (enabled?: boolean) =>
      request<Source[]>(`/sources${enabled !== undefined ? `?enabled=${enabled}` : ''}`),
    get: (id: number) => request<Source>(`/sources/${id}`),
    create: (data: Partial<Source>) =>
      request<Source>('/sources', { method: 'POST', body: JSON.stringify(data) }),
    update: (id: number, data: Partial<Source>) =>
      request<Source>(`/sources/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
    delete: (id: number) => request<void>(`/sources/${id}`, { method: 'DELETE' }),
    fetchNow: (id: number) =>
      request<{ articles_fetched: number }>(`/sources/${id}/fetch`, { method: 'POST' }),
  },

  // 简报
  briefings: {
    list: (page = 1, limit = 20) =>
      request<Paginated<Briefing>>(`/briefings?page=${page}&limit=${limit}`),
    get: (id: number) => request<Briefing>(`/briefings/${id}`),
    generate: () => request<{ briefing_id: number }>('/briefings/generate', { method: 'POST' }),
    delete: (id: number) => request<void>(`/briefings/${id}`, { method: 'DELETE' }),
  },

  // 文章
  articles: {
    list: (sourceId?: number, page = 1, limit = 20) =>
      request<Paginated<Article>>(
        `/articles?source_id=${sourceId || ''}&page=${page}&limit=${limit}`,
      ),
    fetch: () => request<{ articles_fetched: number }>('/articles/fetch', { method: 'POST' }),
  },

  // 偏好设置
  preferences: {
    get: () => request<Preferences>('/preferences'),
    update: (data: Partial<Preferences>) =>
      request<Preferences>('/preferences', { method: 'PUT', body: JSON.stringify(data) }),
  },
};
