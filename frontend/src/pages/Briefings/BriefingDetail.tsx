import { useEffect, useState } from 'react';
import { useParams, Link } from 'react-router';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { api } from '../../lib/api';
import type { Briefing } from '../../types';
import { Badge, Spinner } from '../../components/ui';
import { ArrowLeft } from 'lucide-react';

export default function BriefingDetail() {
  const { id } = useParams<{ id: string }>();
  const [briefing, setBriefing] = useState<Briefing | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!id) return;
    api.briefings.get(+id).then((b) => {
      setBriefing(b);
      setLoading(false);
    }).catch(() => setLoading(false));
  }, [id]);

  if (loading) {
    return <div className="flex justify-center py-20"><Spinner className="h-6 w-6" /></div>;
  }

  if (!briefing) {
    return (
      <div className="py-20 text-center text-gray-500">
        <p>简报不存在</p>
        <Link to="/briefings" className="text-blue-600 hover:underline mt-2 inline-block">返回列表</Link>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-4xl">
      <div className="mb-6 flex items-center gap-4">
        <Link to="/briefings" className="text-gray-400 hover:text-gray-600">
          <ArrowLeft className="h-5 w-5" />
        </Link>
        <div>
          <h1 className="text-xl font-semibold text-gray-900">{briefing.title}</h1>
          <p className="text-xs text-gray-500 mt-0.5">
            {briefing.generated_at?.replace('T',' ').slice(0,19)} · {briefing.article_count} 篇文章 · <StatusBadge status={briefing.status} />
          </p>
        </div>
      </div>

      {briefing.progress && parseTimings(briefing.progress) && (
        <details className="mb-4">
          <summary className="text-xs text-gray-500 cursor-pointer hover:text-gray-700">
            管线耗时
          </summary>
          <div className="mt-1 flex gap-3 text-xs text-gray-500">
            {Object.entries(parseTimings(briefing.progress)!).map(([k,v]) => (
              <span key={k}>{k}: <b className="text-gray-700">{v as string}</b></span>
            ))}
          </div>
        </details>
      )}

      <div className="prose max-w-none">
        <ReactMarkdown remarkPlugins={[remarkGfm]}>
          {briefing.content_markdown}
        </ReactMarkdown>
      </div>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const label = { pending: '待处理', generating: '生成中', completed: '已完成', failed: '失败' }[status] || status;
  const color = status === 'completed' ? 'green' : status === 'failed' ? 'red' : status === 'generating' ? 'blue' : 'gray';
  return <Badge color={color}>{label}</Badge>;
}

function parseTimings(progress: string): Record<string,string> | null {
  const idx = progress.indexOf(':');
  if (idx < 0) return null;
  try { return JSON.parse(progress.slice(idx + 1)); } catch { return null; }
}
