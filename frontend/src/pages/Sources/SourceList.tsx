import React, { useEffect, useState } from 'react';
import { api } from '../../lib/api';
import type { Source } from '../../types';
import { Card } from '../../components/ui';
import { Plus, RefreshCw, Trash2, Edit3, Power } from 'lucide-react';

export default function SourceList() {
  const [sources, setSources] = useState<Source[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [edit, setEdit] = useState<Source | null>(null);

  const load = async () => {
    try { setSources(await api.sources.list()); } catch { /* */ }
  };
  useEffect(() => { load(); }, []);

  const handleDelete = async (id: number) => {
    if (!confirm('确定删除此源？')) return;
    await api.sources.delete(id);
    load();
  };

  const handleFetch = async (id: number) => {
    try {
      const r = await api.sources.fetchNow(id);
      alert(`抓取完成，新文章 ${r.articles_fetched} 篇`);
      load();
    } catch (e: any) {
      alert('抓取失败: ' + e.message);
    }
  };

  const handleToggle = async (s: Source) => {
    await api.sources.update(s.id, { enabled: !s.enabled });
    load();
  };

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-xl font-semibold text-gray-900">内容源管理</h2>
        <button
          onClick={() => { setEdit(null); setShowForm(true); }}
          className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700"
        >
          <Plus className="h-4 w-4" /> 添加源
        </button>
      </div>

      {sources.length === 0 ? (
        <Card>
          <p className="text-center text-gray-500 py-8">暂无配置的内容源，点击"添加源"开始。</p>
        </Card>
      ) : (
        <div className="space-y-2">
          {sources.map((s) => (
            <Card key={s.id} className="flex items-center justify-between">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="font-medium text-gray-900 truncate">{s.name}</span>
                  <SourceBadge type={s.source_type} />
                  {s.enabled ? (
                    <span className="inline-block h-1.5 w-1.5 rounded-full bg-green-500" />
                  ) : (
                    <span className="inline-block h-1.5 w-1.5 rounded-full bg-gray-300" />
                  )}
                </div>
                <p className="text-xs text-gray-500 truncate mt-0.5">{s.url}</p>
              </div>
              <div className="flex items-center gap-1 ml-4">
                <button onClick={() => handleFetch(s.id)}
                  className="rounded p-1.5 text-gray-400 hover:bg-gray-100 hover:text-blue-600"
                  title="立即抓取">
                  <RefreshCw className="h-4 w-4" />
                </button>
                <button onClick={() => handleToggle(s)}
                  className="rounded p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
                  title={s.enabled ? '禁用' : '启用'}>
                  <Power className="h-4 w-4" />
                </button>
                <button onClick={() => { setEdit(s); setShowForm(true); }}
                  className="rounded p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
                  title="编辑">
                  <Edit3 className="h-4 w-4" />
                </button>
                <button onClick={() => handleDelete(s.id)}
                  className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600"
                  title="删除">
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            </Card>
          ))}
        </div>
      )}

      {showForm && (
        <SourceFormPanel
          edit={edit}
          onClose={() => setShowForm(false)}
          onSaved={() => { setShowForm(false); load(); }}
        />
      )}
    </div>
  );
}

function SourceBadge({ type }: { type: string }) {
  const color = type === 'rss' ? 'bg-orange-100 text-orange-700'
    : type === 'hackernews' ? 'bg-amber-100 text-amber-700'
    : type === 'github' ? 'bg-gray-100 text-gray-700'
    : 'bg-blue-100 text-blue-700';
  return <span className={`inline-flex rounded px-1.5 py-0.5 text-xs font-medium ${color}`}>{type}</span>;
}

function parseConfig(cfg: string): Record<string, unknown> {
  try { return JSON.parse(cfg) as Record<string, unknown>; } catch { return {}; }
}

function SourceFormPanel({ edit, onClose, onSaved }: {
  edit: Source | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const cfg = parseConfig(edit?.config_json || '');
  const [name, setName] = useState(edit?.name || '');
  const [stype, setStype] = useState(edit?.source_type || 'rss');
  const [url, setUrl] = useState(edit?.url || '');
  const [enabled, setEnabled] = useState(edit?.enabled ?? true);
  const [saving, setSaving] = useState(false);

  // HN fields
  const [hnType, setHnType] = useState((cfg.story_type as string) || 'top');
  const [hnMax, setHnMax] = useState((cfg.max_items as number) || 30);
  // GitHub fields
  const [ghLang, setGhLang] = useState((cfg.language as string) || '');
  const [ghSince, setGhSince] = useState((cfg.since as string) || 'daily');

  const buildConfig = (): string => {
    if (stype === 'hackernews') return JSON.stringify({ story_type: hnType, max_items: hnMax });
    if (stype === 'github') return JSON.stringify({ language: ghLang, since: ghSince });
    return '';
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      const payload = { name, source_type: stype, url, enabled, config_json: buildConfig() };
      if (edit) {
        await api.sources.update(edit.id, payload);
      } else {
        await api.sources.create(payload);
      }
      onSaved();
    } catch (err: any) {
      alert('保存失败: ' + err.message);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={onClose}>
      <form onSubmit={handleSubmit} onClick={(e) => e.stopPropagation()}
        className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl">
        <h3 className="mb-4 text-lg font-semibold text-gray-900">
          {edit ? '编辑源' : '添加内容源'}
        </h3>

        <label className="mb-1 block text-sm font-medium text-gray-700">名称</label>
        <input className="mb-3 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
          value={name} onChange={(e) => setName(e.target.value)} required placeholder="如 IT之家 RSS" />

        <label className="mb-1 block text-sm font-medium text-gray-700">类型</label>
        <select className="mb-3 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
          value={stype} onChange={(e) => setStype(e.target.value)}>
          <option value="rss">RSS</option>
          <option value="hackernews">HackerNews</option>
          <option value="github">GitHub Trending</option>
        </select>

        <label className="mb-1 block text-sm font-medium text-gray-700">URL</label>
        <input className="mb-3 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
          value={url} onChange={(e) => setUrl(e.target.value)} required
          placeholder={stype === 'hackernews' ? 'https://hacker-news.firebaseio.com' : stype === 'github' ? 'https://github.com' : 'https://...'} />

        {/* HN 额外配置 */}
        {stype === 'hackernews' && (
          <div className="mb-3 rounded-lg bg-amber-50 p-3">
            <p className="mb-2 text-xs font-medium text-amber-800">HackerNews 配置</p>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="mb-1 block text-xs text-amber-700">榜单类型</label>
                <select className="w-full rounded border border-amber-300 bg-white px-2 py-1.5 text-sm"
                  value={hnType} onChange={(e) => setHnType(e.target.value)}>
                  <option value="top">Top（热门）</option>
                  <option value="new">New（最新）</option>
                  <option value="best">Best（最佳）</option>
                </select>
              </div>
              <div>
                <label className="mb-1 block text-xs text-amber-700">抓取条数</label>
                <input type="number" className="w-full rounded border border-amber-300 bg-white px-2 py-1.5 text-sm"
                  value={hnMax} onChange={(e) => setHnMax(+e.target.value)} min={1} max={100} />
              </div>
            </div>
          </div>
        )}

        {/* GitHub 额外配置 */}
        {stype === 'github' && (
          <div className="mb-3 rounded-lg bg-gray-100 p-3">
            <p className="mb-2 text-xs font-medium text-gray-700">GitHub Trending 配置</p>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="mb-1 block text-xs text-gray-600">时间范围</label>
                <select className="w-full rounded border border-gray-300 bg-white px-2 py-1.5 text-sm"
                  value={ghSince} onChange={(e) => setGhSince(e.target.value)}>
                  <option value="daily">今日</option>
                  <option value="weekly">本周</option>
                  <option value="monthly">本月</option>
                </select>
              </div>
              <div>
                <label className="mb-1 block text-xs text-gray-600">语言</label>
                <input className="w-full rounded border border-gray-300 bg-white px-2 py-1.5 text-sm"
                  value={ghLang} onChange={(e) => setGhLang(e.target.value)}
                  placeholder="留空=全部 如 go" />
              </div>
            </div>
          </div>
        )}

        <label className="mb-4 flex items-center gap-2 text-sm text-gray-700">
          <input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
          启用
        </label>

        <div className="flex justify-end gap-2">
          <button type="button" onClick={onClose}
            className="rounded-lg px-4 py-2 text-sm text-gray-600 hover:bg-gray-100">取消</button>
          <button type="submit" disabled={saving}
            className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">
            {saving ? '保存中...' : '保存'}
          </button>
        </div>
      </form>
    </div>
  );
}
