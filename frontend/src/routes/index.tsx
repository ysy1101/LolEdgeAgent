import { createBrowserRouter, Navigate } from 'react-router';
import DashboardLayout from '../layouts/DashboardLayout';
import BriefingList from '../pages/Briefings/BriefingList';
import BriefingDetail from '../pages/Briefings/BriefingDetail';
import SourceList from '../pages/Sources/SourceList';
import Preferences from '../pages/Preferences';
import Chat from '../pages/Chat';
import NotFound from '../pages/NotFound';

export const router = createBrowserRouter([
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
  { path: '*', element: <NotFound /> },
]);
