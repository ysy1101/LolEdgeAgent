import React, { useEffect, useState } from 'react';
import { api } from '../lib/api';
import type { Preferences } from '../types';
import { Card, Button } from '../components/ui';
import { Plus, X } from 'lucide-react';

export default function PreferencesPage() {
  const [pref, setPref] = useState<Preferences | null>(null);
  const [keywords, setKeywords] = useState<string[]>([]);
  const [excluded, setExcluded] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api.preferences.get().then((p) => {
      setPref(p);
      setKeywords(parseKeywords(p.keywords));
      setExcluded(parseKeywords(p.excluded_keywords));
    });
  }, []);

  const handleSave = async () => {
    if (!pref) return;
    setSaving(true);
    try {
      await api.preferences.update({
        ...pref,
        keywords: JSON.stringify(keywords),
        excluded_keywords: JSON.stringify(excluded),
      });
      alert('已保存');
    } catch (e: any) {
      alert('保存失败: ' + e.message);
    } finally {
      setSaving(false);
    }
  };

  if (!pref) return <p className="text-gray-500">加载中...</p>;

  return (
    <div className="max-w-2xl space-y-6">
      <h2 className="text-xl font-semibold text-gray-900">偏好设置</h2>

      {/* 关键词 */}
      <Card>
        <h3 className="mb-2 text-sm font-medium text-gray-700">关注关键词</h3>
        <p className="mb-2 text-xs text-gray-500">输入你感兴趣的技术领域或话题，用于文章排序</p>
        <TagInput tags={keywords} onChange={setKeywords} />
      </Card>

      <Card>
        <h3 className="mb-2 text-sm font-medium text-gray-700">排除关键词</h3>
        <p className="mb-2 text-xs text-gray-500">包含这些关键词的文章会被过滤掉</p>
        <TagInput tags={excluded} onChange={setExcluded} />
      </Card>

      {/* 数量限制 */}
      <Card>
        <h3 className="mb-3 text-sm font-medium text-gray-700">数量限制</h3>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="mb-1 block text-xs text-gray-500">每个源最大抓取</label>
            <input type="number" className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
              value={pref.max_articles_per_source}
              onChange={(e) => setPref({ ...pref, max_articles_per_source: +e.target.value })} />
          </div>
          <div>
            <label className="mb-1 block text-xs text-gray-500">简报包含最大文章数</label>
            <input type="number" className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
              value={pref.max_briefing_articles}
              onChange={(e) => setPref({ ...pref, max_briefing_articles: +e.target.value })} />
          </div>
        </div>
      </Card>

      {/* LLM 配置 */}
      <Card>
        <h3 className="mb-3 text-sm font-medium text-gray-700">LLM 配置</h3>
        <div className="space-y-3">
          <div>
            <label className="mb-1 block text-xs text-gray-500">API Key</label>
            <input type="password" className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
              placeholder="sk-..." value={pref.llm_api_key}
              onChange={(e) => setPref({ ...pref, llm_api_key: e.target.value })} />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="mb-1 block text-xs text-gray-500">模型</label>
              <input className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
                value={pref.llm_model}
                onChange={(e) => setPref({ ...pref, llm_model: e.target.value })} />
            </div>
            <div>
              <label className="mb-1 block text-xs text-gray-500">Base URL（可选）</label>
              <input className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
                placeholder="https://api.deepseek.com"
                value={pref.llm_base_url}
                onChange={(e) => setPref({ ...pref, llm_base_url: e.target.value })} />
            </div>
          </div>
        </div>
      </Card>

      {/* 定时调度 */}
      <Card>
        <h3 className="mb-2 text-sm font-medium text-gray-700">定时生成</h3>
        <p className="mb-2 text-xs text-gray-500">Cron 表达式，留空表示手动触发。如 "0 8 * * *" 表示每天早 8 点</p>
        <input className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm font-mono"
          placeholder="0 8 * * *" value={pref.briefing_schedule}
          onChange={(e) => setPref({ ...pref, briefing_schedule: e.target.value })} />
      </Card>

      <Button onClick={handleSave} disabled={saving}>
        {saving ? '保存中...' : '保存设置'}
      </Button>
    </div>
  );
}

/** 标签输入组件 */
function TagInput({ tags, onChange }: { tags: string[]; onChange: (t: string[]) => void }) {
  const [text, setText] = useState('');

  const add = () => {
    const t = text.trim();
    if (t && !tags.includes(t)) {
      onChange([...tags, t]);
      setText('');
    }
  };

  const remove = (idx: number) => {
    onChange(tags.filter((_, i) => i !== idx));
  };

  return (
    <div>
      <div className="flex flex-wrap gap-1.5 mb-2">
        {tags.map((tag, i) => (
          <span key={i} className="inline-flex items-center gap-1 rounded-full bg-blue-50 px-2.5 py-0.5 text-xs font-medium text-blue-700">
            {tag}
            <button onClick={() => remove(i)} className="text-blue-400 hover:text-blue-600">
              <X className="h-3 w-3" />
            </button>
          </span>
        ))}
      </div>
      <div className="flex gap-2">
        <input className="flex-1 rounded-lg border border-gray-300 px-3 py-1.5 text-sm"
          value={text} onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); add(); } }}
          placeholder="输入后回车添加" />
        <button onClick={add}
          className="rounded-lg border border-gray-300 px-2 py-1.5 text-gray-500 hover:bg-gray-100">
          <Plus className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}

function parseKeywords(s: string): string[] {
  try { return JSON.parse(s); } catch { return []; }
}
