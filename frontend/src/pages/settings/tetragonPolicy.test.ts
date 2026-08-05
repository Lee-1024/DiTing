import { generatePolicy, type PolicyFormValues } from './tetragonPolicy';

function assertIncludes(text: string, expected: string) {
  if (!text.includes(expected)) {
    throw new Error(`expected YAML to include ${expected}`);
  }
}

function assertNotIncludes(text: string, unexpected: string) {
  if (text.includes(unexpected)) {
    throw new Error(`expected YAML not to include ${unexpected}`);
  }
}

const policy: PolicyFormValues = {
  template: 'dangerous_command',
  mode: 'enforce',
  name: 'block-docker-restart',
  enabled: true,
  commands: ['reboot'],
  commandRules: [
    {
      binary: 'systemctl',
      args: ['restart|stop', 'docker|docker.service'],
    },
  ],
  userMatchMode: 'exclude_root',
};

const yaml = generatePolicy(policy);

assertIncludes(yaml, 'call: "sys_execve"');
assertIncludes(yaml, 'values:\n            - "reboot"');
assertIncludes(yaml, '- "diting-enforcement"');
assertIncludes(yaml, '- "diting-blocked-command"');
assertIncludes(yaml, '- index: 1\n        operator: Equal\n        values:\n            - "restart"\n            - "stop"');
assertIncludes(yaml, '- index: 2\n        operator: Equal\n        values:\n            - "docker"\n            - "docker.service"');
assertIncludes(yaml, 'source: "current_task"\n      resolve: "cred.uid.val"');
assertIncludes(yaml, 'matchData:\n      - index: 0\n        operator: NotEqual\n        values:\n        - "0"');
assertIncludes(yaml, 'matchActions:\n      - action: Sigkill');
assertIncludes(yaml, '- "diting-sudo-ancestry"');
assertIncludes(yaml, 'matchParentBinaries:\n      - operator: In\n        values:\n        - "/usr/bin/sudo"\n        - "/bin/sudo"\n        followChildren: true');
assertNotIncludes(yaml, '            - "sudo"');

const sensitiveFileYaml = generatePolicy({
  template: 'sensitive_file',
  mode: 'enforce',
  name: 'block-docker-config',
  enabled: true,
  filePaths: ['/etc/docker/daemon.json'],
  processNames: ['vim'],
  userMatchMode: 'exclude_root',
});

assertIncludes(sensitiveFileYaml, '- "diting-sudo-ancestry"');
assertIncludes(sensitiveFileYaml, 'matchParentBinaries:\n      - operator: In\n        values:\n        - "/usr/bin/sudo"\n        - "/bin/sudo"\n        followChildren: true');
assertNotIncludes(sensitiveFileYaml, '- "diting-sudo-pre-escalation"');
assertNotIncludes(sensitiveFileYaml, '            - "sudo"');

const suspiciousProcessYaml = generatePolicy({
  template: 'suspicious_process',
  mode: 'enforce',
  name: 'block-suspicious-shell',
  enabled: true,
  processNames: ['bash'],
  userMatchMode: 'exclude_root',
});

assertIncludes(suspiciousProcessYaml, '- "diting-sudo-ancestry"');
assertIncludes(suspiciousProcessYaml, 'matchParentBinaries:\n      - operator: In\n        values:\n        - "/usr/bin/sudo"\n        - "/bin/sudo"\n        followChildren: true');
assertIncludes(suspiciousProcessYaml, 'matchData:\n      - index: 0\n        operator: NotEqual\n        values:\n        - "0"');
assertNotIncludes(suspiciousProcessYaml, '            - "sudo"');

const deleteProtectionYaml = generatePolicy({
  template: 'delete_behavior',
  mode: 'enforce',
  name: 'block-test-delete',
  enabled: true,
  filePaths: ['/home/ubuntu/test'],
  processNames: ['rm'],
  userMatchMode: 'exclude_root',
});

assertIncludes(deleteProtectionYaml, 'call: "security_path_unlink"');
assertIncludes(deleteProtectionYaml, 'call: "security_path_rmdir"');
assertIncludes(deleteProtectionYaml, '- "delete-protection"');
assertIncludes(deleteProtectionYaml, '- "diting-enforcement"');
assertIncludes(deleteProtectionYaml, '- "diting-blocked-command"');
assertIncludes(deleteProtectionYaml, 'operator: Equal\n        values:\n            - "/home/ubuntu/test"');
assertIncludes(deleteProtectionYaml, 'matchBinaries:\n      - operator: Postfix\n        values:\n        - "rm"');
assertIncludes(deleteProtectionYaml, 'matchActions:\n      - action: Override\n        argError: -1\n      - action: Sigkill');
