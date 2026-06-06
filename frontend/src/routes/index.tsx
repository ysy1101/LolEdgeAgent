import { useEffect, useState } from 'react';
import { createBrowserRouter, Navigate, Outlet } from 'react-router';
import DashboardLayout from '../layouts/DashboardLayout';
import BriefingList from '../pages/Briefings/BriefingList';
import BriefingDetail from '../pages/Briefings/BriefingDetail';
import SourceList from '../pages/Sources/SourceList';
import Preferences from '../pages/Preferences';
import Chat from '../pages/Chat';
import Login from '../pages/Login';
import NotFound from '../pages/NotFound';
import { Spinner } from '../components/ui';

function RequireAuth() {
  const token = localStorage.getItem('token');
  const [verifying, setVerifying] = useState(!!token);
  const [valid, setValid] = useState(false);

  useEffect(() => {
    if (!token) { setVerifying(false); return; }
    fetch('/api/v1/auth/verify', {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then(res => res.json())
      .then(json => { setValid(json.code === 0); setVerifying(false); })
      .catch(() => { setValid(false); setVerifying(false); });
  }, [token]);

  if (!token) return <Navigate to="/login" replace />;
  if (verifying) return <div className="flex h-screen items-center justify-center"><Spinner /></div>;
  if (!valid) {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    return <Navigate to="/login" replace />;
  }
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
