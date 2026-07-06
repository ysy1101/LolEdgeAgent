import React, { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router';
import { api } from '../../lib/api';
import type { Briefing } from '../../types';
import { Card, Badge, Spinner } from '../../components/ui';
import { Plus, Trash2, RefreshCw } from 'lucide-react';

export default function BriefingList() {
  const [briefings, setBriefings] = useState<Briefing[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const pageSize = 5;
  const [generating, setGenerating] = useState(false);
  const navigate = useNavigate();

  const load = useCallback(async () => {
    try {
      const r = await api.briefings.list(page, pageSize);
      setBriefings(r.items);
      setTotal(r.total);
    } catch { /* */ }
  }, [page]);
  useEffect(() => { load(); }, [load]);

  // 如果存在正在生成的简报，每 5 秒轮询
  const hasGenerating = briefings.some((b) => b.status === 'generating' || b.status === 'pending');
  useEffect(() => {
    if (!hasGenerating) return;
    const timer = setInterval(load, 5000);
    return () => clearInterval(timer);
  }, [hasGenerating, load]);

  const handleDelete = async (e: React.MouseEvent, id: number) => {
    e.stopPropagation();
    if (!confirm('确定删除此简报？')) return;
    await api.briefings.delete(id);
    setPage(1); load();
  };

  const handleRetry = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (generating) return;
    setGenerating(true);
    try {
      await api.briefings.generate();
      setPage(1); await load();
    } catch (err: any) {
      alert('重试失败: ' + err.message);
    } finally {
      setGenerating(false);
    }
  };

  const handleGenerate = async () => {
    if (generating) return;
    setGenerating(true);
    try {
      const r = await api.briefings.generate();
      alert(`生成任务已启动，简报 ID: ${r.briefing_id}。完成后自动刷新列表。`);
      await load(); // 立即刷新列表
    } catch (e: any) {
      alert('生成失败: ' + e.message);
    } finally {
      setGenerating(false);
    }
  };

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-xl font-semibold text-gray-900">简报列表</h2>
        <button onClick={handleGenerate} disabled={generating}
          className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">
          {generating ? <Spinner className="h-4 w-4 border-white border-t-transparent" /> : <Plus className="h-4 w-4" />}
          {generating ? '生成中...' : '生成简报'}
        </button>
      </div>

      {briefings.length === 0 ? (
        <Card>
          <div className="py-12 text-center text-gray-500">
            <p className="mb-2">暂无简报</p>
            <p className="text-xs">点击"生成简报"开始你的第一份内容简报</p>
          </div>
        </Card>
      ) : (
        <div className="space-y-2">
          {briefings.map((b) => (
            <Card
              key={b.id}
              className={`flex items-center justify-between transition-shadow ${
                b.status === 'completed' ? 'cursor-pointer hover:shadow-md' : 'cursor-default'
              } ${b.status === 'failed' ? 'border-red-200 bg-red-50/30' : ''}`}
              onClick={() => b.status === 'completed' && navigate(`/briefings/${b.id}`)}
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="text-xs text-gray-400">#{b.id}</span>
                  <span className="font-medium text-gray-900 truncate">{b.title}</span>
                  <StatusBadge status={b.status} />
                  {(b.status === 'generating' || b.status === 'pending') && b.progress && (
                    <span className="text-xs text-blue-500 animate-pulse">{b.progress}</span>
                  )}
                </div>
                {b.status === 'failed' && b.error_message ? (
                  <p className="mt-1 text-xs text-red-600">{b.error_message}</p>
                ) : (
                  <p className="mt-1 text-xs text-gray-500">
                    {fmtTime(b.generated_at)} · {b.article_count} 篇文章
                  </p>
                )}
              </div>
              <div className="flex items-center gap-1 ml-2">
                {b.status === 'failed' && (
                  <button onClick={(e) => handleRetry(e)}
                    className="rounded p-1.5 text-gray-400 hover:bg-blue-50 hover:text-blue-600"
                    disabled={generating}
                    title="重试">
                    <RefreshCw className="h-4 w-4" />
                  </button>
                )}
                <button onClick={(e) => handleDelete(e, b.id)}
                  className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600"
                  title="删除">
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            </Card>
          ))}
        </div>
      )}

      {/* 分页 */}
      {total > pageSize && (
        <div className="mt-4 flex items-center justify-center gap-2">
          <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1}
            className="rounded px-2 py-1 text-xs text-gray-600 hover:bg-gray-100 disabled:opacity-30">
            上一页
          </button>
          <span className="text-xs text-gray-500">{page} / {Math.ceil(total / pageSize)}</span>
          <button onClick={() => setPage(p => p + 1)} disabled={page * pageSize >= total}
            className="rounded px-2 py-1 text-xs text-gray-600 hover:bg-gray-100 disabled:opacity-30">
            下一页
          </button>
          <input
            type="number" min={1} max={Math.ceil(total / pageSize)}
            className="w-10 rounded border border-gray-200 px-1 py-0.5 text-xs text-center"
            onKeyDown={e => { if (e.key === 'Enter') { const v = parseInt((e.target as HTMLInputElement).value); if (v >= 1 && v <= Math.ceil(total / pageSize)) setPage(v); } }}
          />
        </div>
      )}
    </div>
  );
}

function fmtTime(t: string) { return t?.replace('T', ' ').slice(0, 19) || ''; }

function StatusBadge({ status }: { status: string }) {
  const label = { pending: '待处理', generating: '生成中', completed: '已完成', failed: '失败' }[status] || status;
  const color = status === 'completed' ? 'green' : status === 'failed' ? 'red' : status === 'generating' ? 'blue' : 'gray';
  return <Badge color={color}>{label}</Badge>;
}
