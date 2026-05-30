import { Link } from 'react-router';

export default function NotFound() {
  return (
    <div className="flex h-screen flex-col items-center justify-center gap-4">
      <h1 className="text-4xl font-semibold text-gray-900">404</h1>
      <p className="text-gray-500">页面不存在</p>
      <Link to="/briefings" className="text-blue-600 hover:underline">
        返回首页
      </Link>
    </div>
  );
}
