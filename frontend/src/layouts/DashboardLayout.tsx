import { Outlet } from 'react-router';
import { Sidebar } from '../components/layout/Sidebar';

export default function DashboardLayout() {
  return (
    <div className="flex h-screen bg-white">
      <Sidebar />
      <div className="flex flex-1 flex-col overflow-hidden">
        <main className="flex-1 overflow-y-auto p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
