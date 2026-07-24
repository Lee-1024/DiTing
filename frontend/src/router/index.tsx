import { createBrowserRouter, Navigate } from 'react-router-dom';
import MainLayout from '../layouts/MainLayout';
import DashboardPage from '../pages/dashboard/DashboardPage';
import AuditEventsPage from '../pages/audit-events/AuditEventsPage';
import CommandStatsPage from '../pages/commands/CommandStatsPage';
import UserAuditPage from '../pages/users/UserAuditPage';
import UserManagementPage from '../pages/users/UserManagementPage';
import HostAuditPage from '../pages/hosts/HostAuditPage';
import HostAssetsPage from '../pages/host-assets/HostAssetsPage';
import RiskEventsPage from '../pages/risks/RiskEventsPage';
import RulesPage from '../pages/rules/RulesPage';
import RuleHitsPage from '../pages/rules/RuleHitsPage';
import OperationLogsPage from '../pages/settings/OperationLogsPage';
import CollectorHealthPage from '../pages/settings/CollectorHealthPage';
import CollectorDebugPage from '../pages/settings/CollectorDebugPage';
import CollectorConfigPage from '../pages/settings/CollectorConfigPage';
import TetragonPolicyPage from '../pages/settings/TetragonPolicyPage';
import LoginPage from '../pages/login/LoginPage';
import { getToken } from '../stores/auth';

// RequireAuth 渲染 Require Auth 组件。
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
      { path: 'audit/rules', element: <RuleHitsPage /> },
      { path: 'assets/hosts', element: <HostAssetsPage /> },
      { path: 'rules', element: <RulesPage /> },
      { path: 'settings/users', element: <UserManagementPage /> },
      { path: 'settings/operation-logs', element: <OperationLogsPage /> },
      { path: 'settings/collector-health', element: <CollectorHealthPage /> },
      { path: 'settings/collector-debug', element: <CollectorDebugPage /> },
      { path: 'settings/collector', element: <CollectorConfigPage /> },
      { path: 'settings/tetragon-policies', element: <TetragonPolicyPage /> },
      { path: '*', element: <Navigate to="/" replace /> },
    ],
  },
]);
