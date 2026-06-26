import React, { useEffect, useRef, useState } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import type { Article } from '../types';
import { Spinner } from '../components/ui';
import { Send, Plus, Trash2 } from 'lucide-react';

interface Step { round: number; role: string; content: string; tool?: string }

interface Message { role: 'user' | 'assistant'; content: string; steps?: Step[]; articles?: Article[] }

interface Conv { id: number; title: string }

export default function Chat() {
  const [convs, setConvs] = useState<Conv[]>([]);
  const [convId, setConvId] = useState<number | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const msgContainerRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    const el = msgContainerRef.current;
    if (el) {
      // 等 DOM 渲染完再滚，避免 ReactMarkdown 渲染中滚动不准
      requestAnimationFrame(() => {
        el.scrollTop = el.scrollHeight;
      });
    }
  };

  const token = () => localStorage.getItem('token') || '';
  const headers = () => ({ 'Content-Type': 'application/json', Authorization: `Bearer ${token()}` });

  const loadConvs = async () => {
    const res = await fetch('/api/v1/conversations', { headers: headers() });
    const json = await res.json();
    if (json.code === 0) setConvs(json.data);
  };
  useEffect(() => { loadConvs(); }, []);

  const newConv = async () => {
    const res = await fetch('/api/v1/conversations', { method: 'POST', headers: headers() });
    const json = await res.json();
    if (json.code === 0) {
      setConvId(json.data.id);
      setMessages([]);
      loadConvs();
    }
  };

  const switchConv = async (id: number) => {
    setConvId(id);
    const res = await fetch(`/api/v1/conversations/${id}/messages`, { headers: headers() });
    const json = await res.json();
    if (json.code === 0) {
      const loaded = json.data.map((m: any) => ({ role: m.role, content: m.content }));
      setMessages(loaded);
      // 等 DOM 渲染完再滚到底部
      setTimeout(scrollToBottom, 100);
    }
  };

  const deleteConv = async (e: React.MouseEvent, id: number) => {
    e.stopPropagation();
    if (!confirm('删除此对话？')) return;
    await fetch(`/api/v1/conversations/${id}`, { method: 'DELETE', headers: headers() });
    if (convId === id) { setConvId(null); setMessages([]); }
    loadConvs();
  };

  // 发送消息：先持久化 → 从 DB 取真实历史 → Agent 处理 → 再持久化回复
  const send = async () => {
    const q = input.trim();
    if (!q || loading || !convId) return;
    setInput('');
    setLoading(true);

    // 1. 先把用户消息落 DB（保证失败也能追溯到用户问了什么）
    await fetch(`/api/v1/conversations/${convId}/messages`, {
      method: 'POST', headers: headers(),
      body: JSON.stringify({ role: 'user', content: q }),
    });

    // 2. 从 DB 取完整历史（真实状态，不依赖 React state 时序）
    const msgsRes = await fetch(`/api/v1/conversations/${convId}/messages`, { headers: headers() });
    const msgsJson = await msgsRes.json();
    const allMsgs: any[] = msgsJson.data || [];
    setMessages(allMsgs.map((m: any) => ({ role: m.role, content: m.content })));

    // 3. 历史交给 Agent（最近 20 条，不含刚保存的用户消息）
    const history = allMsgs.slice(0, -1).slice(-20).map((m: any) => ({
      role: m.role,
      content: m.content,
    }));

    try {
      const res = await fetch('/api/v1/agent/chat', {
        method: 'POST', headers: headers(),
        body: JSON.stringify({ message: q, history }),
      });
      const json = await res.json();
      const answer = json.data?.content || '回答失败';
      const steps: Step[] = json.data?.steps || [];

      // 4. 助手回复落 DB
      await fetch(`/api/v1/conversations/${convId}/messages`, {
        method: 'POST', headers: headers(),
        body: JSON.stringify({ role: 'assistant', content: answer }),
      });

      // 5. UI 追加助手消息
      setMessages(prev => [...prev, { role: 'assistant', content: answer, steps }]);
    } catch {
      setMessages(prev => [...prev, { role: 'assistant', content: '请求失败' }]);
    } finally {
      setLoading(false);
    }
  };

  // 消息变化时滚到底部（新消息、切换对话）
  useEffect(() => { scrollToBottom(); }, [messages]);

  return (
    <div className="flex h-[calc(100vh-7rem)]">
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

      <div className="flex flex-1 flex-col">
        <div ref={msgContainerRef} className="flex-1 overflow-y-auto space-y-4 pr-2">
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
                {m.role === 'assistant' ? (
                  <div className="prose prose-sm max-w-none prose-headings:text-gray-900 prose-a:text-blue-600 prose-code:text-pink-600 prose-pre:bg-gray-800 prose-pre:text-gray-100">
                    <ReactMarkdown remarkPlugins={[remarkGfm]}>{m.content}</ReactMarkdown>
                  </div>
                ) : (
                  <p className="whitespace-pre-wrap">{m.content}</p>
                )}
                {m.steps && m.steps.length > 0 && (
                  <details className="mt-2 border-t border-gray-200 pt-2">
                    <summary className="text-xs text-gray-500 cursor-pointer hover:text-gray-700">
                      Agent 思考过程 ({m.steps.length} 步)
                    </summary>
                    <div className="mt-1 space-y-1 max-h-48 overflow-y-auto">
                      {m.steps.map((s, j) => (
                        <div key={j} className={`text-xs rounded px-2 py-1 ${
                          s.role === 'error' ? 'bg-red-50 text-red-600' :
                          s.role === 'tool' ? 'bg-green-50 text-green-700' :
                          s.role === 'tool_call' ? 'bg-yellow-50 text-yellow-700' :
                          'bg-blue-50 text-blue-700'
                        }`}>
                          <span className="font-medium">{s.role}</span>
                          {s.tool && <span className="ml-1 text-gray-500">[{s.tool}]</span>}
                          <span className="ml-1">{s.content}</span>
                        </div>
                      ))}
                    </div>
                  </details>
                )}
              </div>
            </div>
          ))}
          {loading && <div className="flex justify-center"><Spinner className="h-5 w-5" /></div>}
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
