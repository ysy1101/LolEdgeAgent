import { createBrowserRouter, Navigate, Outlet } from 'react-router';
import DashboardLayout from '../layouts/DashboardLayout';
import BriefingList from '../pages/Briefings/BriefingList';
import BriefingDetail from '../pages/Briefings/BriefingDetail';
import SourceList from '../pages/Sources/SourceList';
import Preferences from '../pages/Preferences';
import Chat from '../pages/Chat';
import Login from '../pages/Login';
import NotFound from '../pages/NotFound';

function RequireAuth() {
  const token = localStorage.getItem('token');
  if (!token) return <Navigate to="/login" replace />;
  return <Outlet />;
}

export const router = createBrowserRouter([
  { path: '/login', element: <Login /> },
  {
    element: <RequireAuth />,
    children: [
      {
        path: '/',
        Component: DashboardLayout,
        children: [
          { index: true, element: <Navigate to="/chat" replace /> },
          { path: 'chat', element: <Chat /> },
          { path: 'briefings', element: <BriefingList /> },
          { path: 'briefings/:id', element: <BriefingDetail /> },
          { path: 'sources', element: <SourceList /> },
          { path: 'preferences', element: <Preferences /> },
        ],
      },
    ],
  },
  { path: '*', element: <NotFound /> },
]);
