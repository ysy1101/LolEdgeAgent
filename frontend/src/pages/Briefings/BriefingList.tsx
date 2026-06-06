import React, { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router';
import { api } from '../../lib/api';
import type { Briefing } from '../../types';
import { Card, Badge, Spinner } from '../../components/ui';
import { Plus, Trash2 } from 'lucide-react';

export default function BriefingList() {
  const [briefings, setBriefings] = useState<Briefing[]>([]);
  const [generating, setGenerating] = useState(false);
  const navigate = useNavigate();

  const load = useCallback(async () => {
    try {
      const r = await api.briefings.list(1, 50);
      setBriefings(r.items);
    } catch { /* */ }
  }, []);
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
    load();
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
              className="flex cursor-pointer items-center justify-between hover:shadow-md transition-shadow"
              onClick={() => navigate(`/briefings/${b.id}`)}
            >
              <div>
                <div className="flex items-center gap-2">
                  <span className="text-xs text-gray-400">#{b.id}</span>
                  <span className="font-medium text-gray-900">{b.title}</span>
                  <StatusBadge status={b.status} />
                </div>
                <p className="mt-1 text-xs text-gray-500">
                  {b.generated_at} · {b.article_count} 篇文章
                </p>
              </div>
              <button onClick={(e) => handleDelete(e, b.id)}
                className="rounded p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600"
                title="删除">
                <Trash2 className="h-4 w-4" />
              </button>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const label = { pending: '待处理', generating: '生成中', completed: '已完成', failed: '失败' }[status] || status;
  const color = status === 'completed' ? 'green' : status === 'failed' ? 'red' : status === 'generating' ? 'blue' : 'gray';
  return <Badge color={color}>{label}</Badge>;
}
