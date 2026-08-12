import { expect, it } from 'vitest';
import { buildCollectorHostOptions, generatePolicy, type PolicyFormValues } from './tetragonPolicy';

const policy: PolicyFormValues = {
  template: 'sensitive_file',
  mode: 'enforce',
  name: 'protect-docker-config',
  enabled: true,
  filePaths: ['/etc/docker/daemon.json'],
  userMatchMode: 'exclude_root',
};

it('renders an AppArmor deployment preview without Tetragon policy fields', () => {
  const preview = generatePolicy(policy);

  expect(preview).toContain('engine: apparmor');
  expect(preview).toContain('profile: diting-sudo');
  expect(preview).toContain('scope: sudo-and-descendants');
  expect(preview).toContain('direct_root: allowed');
  expect(preview).toContain('- /etc/docker/daemon.json');
  expect(preview).not.toContain('TracingPolicy');
  expect(preview).not.toContain('kprobes:');
  expect(preview).not.toContain('Sigkill');
});

it('marks disabled policies in the preview', () => {
  const preview = generatePolicy({ ...policy, mode: 'disabled', enabled: false });

  expect(preview).toContain('enabled: false');
});

it('includes sensitive file operations in preview', () => {
  const preview = generatePolicy({
    template: 'sensitive_file',
    mode: 'enforce',
    name: 'protect',
    filePaths: ['/etc/docker/daemon.json'],
    operations: ['read', 'change'],
  });

  expect(preview).toContain('operations:');
  expect(preview).toContain('  - read');
  expect(preview).toContain('  - change');
});

it('defaults sensitive file operations to change', () => {
  const preview = generatePolicy({
    template: 'sensitive_file',
    mode: 'enforce',
    name: 'protect',
    filePaths: ['/etc/docker/daemon.json'],
  });

  expect(preview).toContain('  - change');
});

it('builds collector host select options and keeps legacy host ids', () => {
  const options = buildCollectorHostOptions([
    { hostId: 'server-002', hostName: 'diting-test-113', status: 'online' },
    { hostId: 'server-001', hostName: '10.40.0.184', status: 'offline' },
  ], ['legacy-host']);

  expect(options).toEqual([
    { value: 'server-001', label: 'server-001 / 10.40.0.184（离线）' },
    { value: 'server-002', label: 'server-002 / diting-test-113（在线）' },
    { value: 'legacy-host', label: 'legacy-host（历史主机）' },
  ]);
});
