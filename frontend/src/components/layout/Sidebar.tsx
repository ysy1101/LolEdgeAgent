import { NavLink } from 'react-router';
import { FileText, Radio, Settings, Zap } from 'lucide-react';

export function Sidebar() {
  return (
    <aside className="flex h-full w-56 flex-col border-r border-gray-200 bg-gray-50">
      <div className="flex h-14 items-center gap-2 border-b border-gray-200 px-4">
        <Zap className="h-5 w-5 text-blue-600" />
        <span className="text-sm font-semibold text-gray-900">LolEdgeAgent</span>
      </div>

      <nav className="flex flex-1 flex-col gap-1 p-3">
        <NavItem to="/briefings" icon={<FileText className="h-4 w-4" />} label="简报" end />
        <NavItem to="/sources" icon={<Radio className="h-4 w-4" />} label="内容源" />
        <NavItem to="/preferences" icon={<Settings className="h-4 w-4" />} label="偏好设置" />
      </nav>
    </aside>
  );
}

function NavItem({ to, icon, label, end }: { to: string; icon: React.ReactNode; label: string; end?: boolean }) {
  return (
    <NavLink
      to={to}
      end={end}
      className={({ isActive }) =>
        `flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
          isActive
            ? 'bg-blue-50 text-blue-700'
            : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900'
        }`
      }
    >
      {icon}
      {label}
    </NavLink>
  );
}
