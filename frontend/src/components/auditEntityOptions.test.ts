import { describe, expect, it } from 'vitest';
import { buildHostOptions, buildUserOptions } from './auditEntityOptions';

describe('buildUserOptions', () => {
  it('removes empty and duplicate usernames while preserving order', () => {
    expect(buildUserOptions([
      { username: ' root ' },
      { username: '' },
      { username: 'root' },
      { username: 'ubuntu' },
    ])).toEqual([
      { value: 'root', label: 'root' },
      { value: 'ubuntu', label: 'ubuntu' },
    ]);
  });
});

describe('buildHostOptions', () => {
  it('uses host ID, node name, then host name as the submitted value', () => {
    expect(buildHostOptions([
      { hostId: 'id-1', hostName: 'web-1', nodeName: 'node-1' },
      { hostName: 'web-2', nodeName: 'node-2' },
      { hostName: 'web-3' },
    ])).toEqual([
      { value: 'id-1', label: 'web-1 / node-1 / id-1' },
      { value: 'node-2', label: 'web-2 / node-2' },
      { value: 'web-3', label: 'web-3' },
    ]);
  });

  it('removes repeated labels, empty identities, and duplicate values', () => {
    expect(buildHostOptions([
      { hostId: 'id-1', hostName: 'same', nodeName: 'same' },
      { hostId: 'id-1', hostName: 'duplicate' },
      { hostName: '   ' },
    ])).toEqual([
      { value: 'id-1', label: 'same / id-1' },
    ]);
  });
});
