import { expect, it } from 'vitest';
import { generatePolicy, type PolicyFormValues } from './tetragonPolicy';

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
