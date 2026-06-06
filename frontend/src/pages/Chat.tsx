import React, { useEffect, useRef, useState } from 'react';
import type { Article } from '../types';
import { Spinner } from '../components/ui';
import { Send, Plus, Trash2 } from 'lucide-react';

interface Message { role: 'user' | 'assistant'; content: string; articles?: Article[] }

interface Conv { id: number; title: string }

export default function Chat() {
  const [convs, setConvs] = useState<Conv[]>([]);
  const [convId, setConvId] = useState<number | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);

  const token = () => localStorage.getItem('token') || '';
  const headers = () => ({ 'Content-Type': 'application/json', Authorization: `Bearer ${token()}` });

  // 加载对话列表
  const loadConvs = async () => {
    const res = await fetch('/api/v1/conversations', { headers: headers() });
    const json = await res.json();
    if (json.code === 0) setConvs(json.data);
  };
  useEffect(() => { loadConvs(); }, []);

  // 新建对话
  const newConv = async () => {
    const res = await fetch('/api/v1/conversations', { method: 'POST', headers: headers() });
    const json = await res.json();
    if (json.code === 0) {
      setConvId(json.data.id);
      setMessages([]);
      loadConvs();
    }
  };

  // 切换对话
  const switchConv = async (id: number) => {
    setConvId(id);
    const res = await fetch(`/api/v1/conversations/${id}/messages`, { headers: headers() });
    const json = await res.json();
    if (json.code === 0) {
      setMessages(json.data.map((m: any) => ({
        role: m.role,
        content: m.content,
        articles: m.role === 'assistant' ? tryParseArticles(m.content) : undefined,
      })));
    }
  };

  // 删除对话
  const deleteConv = async (e: React.MouseEvent, id: number) => {
    e.stopPropagation();
    if (!confirm('删除此对话？')) return;
    await fetch(`/api/v1/conversations/${id}`, { method: 'DELETE', headers: headers() });
    if (convId === id) { setConvId(null); setMessages([]); }
    loadConvs();
  };

  // 发送消息
  const send = async () => {
    const q = input.trim();
    if (!q || loading || !convId) return;
    setInput('');
    setLoading(true);

    const userMsg: Message = { role: 'user', content: q };
    setMessages(prev => [...prev, userMsg]);

    // 保存用户消息
    await fetch(`/api/v1/conversations/${convId}/messages`, {
      method: 'POST', headers: headers(),
      body: JSON.stringify({ role: 'user', content: q }),
    });

    // 构建历史上下文（最近 10 轮）
    const history = messages.slice(-20).map(m => ({
      role: m.role,
      content: m.content,
    }));

    try {
      const res = await fetch('/api/v1/agent/chat', {
        method: 'POST', headers: headers(),
        body: JSON.stringify({ message: q, history }),
      });
      const json = await res.json();
      const answer = json.data?.content || json.data?.Content || '回答失败';

      // 保存 AI 回复
      await fetch(`/api/v1/conversations/${convId}/messages`, {
        method: 'POST', headers: headers(),
        body: JSON.stringify({ role: 'assistant', content: answer }),
      });

      const aiMsg: Message = { role: 'assistant', content: answer };
      setMessages(prev => [...prev, aiMsg]);
    } catch {
      setMessages(prev => [...prev, { role: 'assistant', content: '请求失败' }]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { bottomRef.current?.scrollIntoView({ behavior: 'smooth' }); }, [messages]);

  return (
    <div className="flex h-[calc(100vh-7rem)]">
      {/* 对话列表侧边 */}
      <div className="w-48 border-r border-gray-200 pr-3 mr-3 overflow-y-auto">
        <button onClick={newConv} className="flex w-full items-center gap-1 rounded-lg px-2 py-1.5 text-xs font-medium text-blue-600 hover:bg-blue-50 mb-2">
          <Plus className="h-3 w-3" /> 新对话
        </button>
        {convs.map(c => (
          <div key={c.id}
            onClick={() => switchConv(c.id)}
            className={`group flex cursor-pointer items-center justify-between rounded-lg px-2 py-1.5 text-xs ${
              convId === c.id ? 'bg-blue-50 text-blue-700' : 'text-gray-600 hover:bg-gray-100'
            }`}>
            <span className="truncate flex-1">{c.title}</span>
            <button onClick={(e) => deleteConv(e, c.id)}
              className="ml-1 hidden group-hover:block text-gray-400 hover:text-red-500">
              <Trash2 className="h-3 w-3" />
            </button>
          </div>
        ))}
      </div>

      {/* 消息区 */}
      <div className="flex flex-1 flex-col">
        <div className="flex-1 overflow-y-auto space-y-4 pr-2">
          {messages.length === 0 && (
            <div className="pt-20 text-center text-gray-400">
              <p className="text-lg">基于已采集文章的知识问答</p>
            </div>
          )}
          {messages.map((m, i) => (
            <div key={i} className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}>
              <div className={`max-w-[80%] rounded-xl px-4 py-2.5 text-sm ${
                m.role === 'user' ? 'bg-blue-600 text-white' : 'bg-gray-100 text-gray-900'
              }`}>
                <p className="whitespace-pre-wrap">{m.content}</p>
                {m.articles && m.articles.length > 0 && (
                  <div className="mt-2 border-t border-gray-200 pt-2">
                    <p className="mb-1 text-xs text-gray-500">参考文章：</p>
                    {m.articles.map(a => (
                      <a key={a.id} href={a.url} target="_blank" rel="noreferrer"
                        className="block text-xs text-blue-600 hover:underline truncate">{a.title}</a>
                    ))}
                  </div>
                )}
              </div>
            </div>
          ))}
          {loading && <div className="flex justify-center"><Spinner className="h-5 w-5" /></div>}
          <div ref={bottomRef} />
        </div>

        <div className="mt-3 flex gap-2 border-t border-gray-200 pt-3">
          <input className="flex-1 rounded-lg border border-gray-300 px-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            placeholder="输入问题..." value={input} onChange={e => setInput(e.target.value)}
            onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send(); } }} />
          <button onClick={send} disabled={loading}
            className="rounded-lg bg-blue-600 px-4 py-2 text-white hover:bg-blue-700 disabled:opacity-50">
            <Send className="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  );
}

function tryParseArticles(_content: string): Article[] | undefined {
  return undefined;
}
