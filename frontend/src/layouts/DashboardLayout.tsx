import { useLocation, Outlet } from 'react-router';
import { Sidebar } from '../components/layout/Sidebar';
import { Header } from '../components/layout/Header';

const titles: Record<string, string> = {
  '/briefings': '简报',
  '/sources': '内容源管理',
  '/preferences': '偏好设置',
};

export default function DashboardLayout() {
  const { pathname } = useLocation();
  const title = titles[pathname] || '简报';

  return (
    <div className="flex h-screen bg-white">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header title={title} />
        <main className="flex-1 overflow-y-auto p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
