import { useEffect, useState } from 'react';
import type { Article } from '../../types';

export default function ArticleList() {
  const [articles, setArticles] = useState<Article[]>([]);
  const [dates, setDates] = useState<string[]>([]);
  const [selectedDate, setSelectedDate] = useState('');
  const [total, setTotal] = useState(0);

  const token = () => localStorage.getItem('token') || '';
  const headers = () => ({ 'Content-Type': 'application/json', Authorization: `Bearer ${token()}` });

  // 加载可用日期
  useEffect(() => {
    fetch('/api/v1/articles/dates', { headers: headers() })
      .then(r => r.json())
      .then(json => {
        if (json.code === 0 && json.data?.length > 0) {
          setDates(json.data);
          setSelectedDate(json.data[0]); // 默认最新日期
        }
      });
  }, []);

  // 加载选中日期的文章
  useEffect(() => {
    if (!selectedDate) return;
    const dateParam = `date=${selectedDate}`;
    fetch(`/api/v1/articles?${dateParam}&limit=100`, { headers: headers() })
      .then(r => r.json())
      .then(json => {
        if (json.code === 0) {
          setArticles(json.data.items);
          setTotal(json.data.total);
        }
      });
  }, [selectedDate]);

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-gray-900">
          文章库
          <span className="ml-2 text-sm font-normal text-gray-400">
            {selectedDate && `— ${selectedDate} · ${total} 篇`}
          </span>
        </h2>
      </div>

      {/* 日期导航 */}
      {dates.length > 0 && (
        <div className="flex gap-1 mb-4 overflow-x-auto pb-1">
          {dates.slice(0, 14).map(d => (
            <button
              key={d}
              onClick={() => setSelectedDate(d)}
              className={`whitespace-nowrap rounded-full px-3 py-1 text-xs font-medium transition-colors ${
                selectedDate === d
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
              }`}
            >
              {d.slice(5)} {/* 显示 MM-DD */}
            </button>
          ))}
        </div>
      )}

      {/* 文章列表 */}
      {articles.length === 0 ? (
        <div className="py-12 text-center text-gray-400">
          <p>{selectedDate ? '当天暂无文章' : '选择日期查看文章'}</p>
        </div>
      ) : (
        <div className="space-y-2">
          {articles.map(a => (
            <a
              key={a.id}
              href={a.url}
              target="_blank"
              rel="noreferrer"
              className="block rounded-lg border border-gray-200 p-3 hover:border-blue-300 hover:bg-blue-50/30 transition-colors"
            >
              <div className="flex items-start gap-2">
                <span className="text-xs text-gray-400 mt-0.5">#{a.id}</span>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-gray-900 truncate">{a.title}</p>
                  {a.description && (
                    <p className="text-xs text-gray-500 mt-0.5 line-clamp-2">{a.description}</p>
                  )}
                </div>
                {a.relevance_score > 0 && (
                  <span className="text-xs text-gray-400 shrink-0">{a.relevance_score.toFixed(1)}</span>
                )}
              </div>
            </a>
          ))}
        </div>
      )}
    </div>
  );
}
