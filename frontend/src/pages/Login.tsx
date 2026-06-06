import { useState } from 'react';
import { useNavigate } from 'react-router';
import { Zap } from 'lucide-react';

export default function Login() {
  const [tab, setTab] = useState<'login' | 'register'>('login');
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const endpoint = tab === 'login' ? '/api/v1/auth/login' : '/api/v1/auth/register';
      const body: Record<string, string> = { username, password };
      if (tab === 'register') body.email = email;

      const res = await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const json = await res.json();
      if (json.code !== 0) {
        setError(json.message);
        return;
      }
      localStorage.setItem('token', json.data.token);
      localStorage.setItem('user', JSON.stringify(json.data.user));
      navigate('/chat');
    } catch {
      setError('网络错误');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50">
      <div className="w-full max-w-sm rounded-xl bg-white p-8 shadow-sm">
        <div className="mb-6 flex items-center justify-center gap-2">
          <Zap className="h-6 w-6 text-blue-600" />
          <span className="text-lg font-semibold text-gray-900">LolEdgeAgent</span>
        </div>

        <div className="mb-4 flex border-b border-gray-200">
          <button
            className={`flex-1 pb-2 text-sm font-medium ${tab === 'login' ? 'border-b-2 border-blue-600 text-blue-600' : 'text-gray-500'}`}
            onClick={() => setTab('login')}>登录</button>
          <button
            className={`flex-1 pb-2 text-sm font-medium ${tab === 'register' ? 'border-b-2 border-blue-600 text-blue-600' : 'text-gray-500'}`}
            onClick={() => setTab('register')}>注册</button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-3">
          <input className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
            placeholder="用户名" autoComplete="username"
            value={username} onChange={e => setUsername(e.target.value)} required />
          {tab === 'register' && (
            <input className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
              placeholder="邮箱（可选）" type="email" autoComplete="email"
              value={email} onChange={e => setEmail(e.target.value)} />
          )}
          <input className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
            placeholder="密码" type="password" autoComplete={tab === 'register' ? 'new-password' : 'current-password'}
            value={password} onChange={e => setPassword(e.target.value)} required />

          {error && <p className="text-xs text-red-600">{error}</p>}

          <button type="submit" disabled={loading}
            className="w-full rounded-lg bg-blue-600 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50">
            {loading ? '请稍候...' : tab === 'login' ? '登录' : '注册'}
          </button>
        </form>
      </div>
    </div>
  );
}
