import { createBrowserRouter, Navigate } from 'react-router-dom';
import MainLayout from '../layouts/MainLayout';
import DashboardPage from '../pages/dashboard/DashboardPage';
import AuditEventsPage from '../pages/audit-events/AuditEventsPage';
import CommandStatsPage from '../pages/commands/CommandStatsPage';
import UserAuditPage from '../pages/users/UserAuditPage';
import HostAuditPage from '../pages/hosts/HostAuditPage';
import HostAssetsPage from '../pages/host-assets/HostAssetsPage';
import RiskEventsPage from '../pages/risks/RiskEventsPage';
import RulesPage from '../pages/rules/RulesPage';
import LoginPage from '../pages/login/LoginPage';
import { getToken } from '../stores/auth';

function RequireAuth({ children }: { children: JSX.Element }) {
  return getToken() ? children : <Navigate to="/login" replace />;
}

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  {
    path: '/',
    element: <RequireAuth><MainLayout /></RequireAuth>,
    children: [
      { index: true, element: <DashboardPage /> },
      { path: 'audit/events', element: <AuditEventsPage /> },
      { path: 'audit/commands', element: <CommandStatsPage /> },
      { path: 'audit/users', element: <UserAuditPage /> },
      { path: 'audit/hosts', element: <HostAuditPage /> },
      { path: 'audit/risks', element: <RiskEventsPage /> },
      { path: 'assets/hosts', element: <HostAssetsPage /> },
      { path: 'rules', element: <RulesPage /> },
      { path: '*', element: <Navigate to="/" replace /> },
    ],
  },
]);
