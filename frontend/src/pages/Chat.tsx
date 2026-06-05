import React, { useState } from 'react';
import type { Article } from '../types';
import { Spinner } from '../components/ui';
import { Send } from 'lucide-react';

interface Message {
  role: 'user' | 'assistant';
  content: string;
  articles?: Article[];
}

export default function Chat() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSend = async () => {
    const q = input.trim();
    if (!q || loading) return;
    setInput('');
    setLoading(true);
    setMessages((prev) => [...prev, { role: 'user', content: q }]);

    try {
      const res = await fetch('/api/v1/ask', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ question: q, top_k: 5 }),
      });
      const json = await res.json();
      const answer = json.data?.answer || '回答失败';
      const articles = json.data?.articles || [];
      setMessages((prev) => [...prev, { role: 'assistant', content: answer, articles }]);
    } catch {
      setMessages((prev) => [...prev, { role: 'assistant', content: '请求失败，请检查服务是否运行。' }]);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="mx-auto flex h-[calc(100vh-7rem)] max-w-3xl flex-col">
      <div className="flex-1 overflow-y-auto space-y-4 pr-2">
        {messages.length === 0 && (
          <div className="pt-20 text-center text-gray-400">
            <p className="mb-2 text-lg">基于已采集文章的知识问答</p>
            <p className="text-sm">示例：最近有哪些关于 Go 语言的文章？</p>
          </div>
        )}
        {messages.map((m, i) => (
          <div key={i} className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}>
            <div className={`max-w-[80%] rounded-xl px-4 py-2.5 text-sm ${
              m.role === 'user'
                ? 'bg-blue-600 text-white'
                : 'bg-gray-100 text-gray-900'
            }`}>
              <p className="whitespace-pre-wrap">{m.content}</p>
              {m.articles && m.articles.length > 0 && (
                <div className="mt-2 border-t border-gray-200 pt-2">
                  <p className="mb-1 text-xs text-gray-500">参考文章：</p>
                  {m.articles.map((a) => (
                    <a key={a.id} href={a.url} target="_blank" rel="noreferrer"
                      className="block text-xs text-blue-600 hover:underline truncate">
                      {a.title}
                    </a>
                  ))}
                </div>
              )}
            </div>
          </div>
        ))}
        {loading && (
          <div className="flex justify-center">
            <Spinner className="h-5 w-5" />
          </div>
        )}
      </div>

      <div className="mt-3 flex gap-2 border-t border-gray-200 pt-3">
        <input
          className="flex-1 rounded-lg border border-gray-300 px-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          placeholder="输入问题，基于已采集文章回答..."
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend(); } }}
        />
        <button onClick={handleSend} disabled={loading}
          className="rounded-lg bg-blue-600 px-4 py-2 text-white hover:bg-blue-700 disabled:opacity-50">
          <Send className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
}
