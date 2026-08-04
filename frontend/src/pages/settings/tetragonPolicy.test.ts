import { generatePolicy, type PolicyFormValues } from './tetragonPolicy';

function assertIncludes(text: string, expected: string) {
  if (!text.includes(expected)) {
    throw new Error(`expected YAML to include ${expected}`);
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
assertIncludes(yaml, 'source: "current_task"\n      resolve: "loginuid.val"');
assertIncludes(yaml, 'matchData:\n      - index: 0\n        operator: NotEqual\n        values:\n        - "0"');
assertIncludes(yaml, 'matchActions:\n      - action: Sigkill');
assertIncludes(yaml, '- "diting-sudo-pre-escalation"');
assertIncludes(yaml, '- index: 0\n        operator: Postfix\n        values:\n            - "sudo"');
assertIncludes(yaml, '- index: 1\n        operator: Postfix\n        values:\n            - "reboot"');

const sensitiveFileYaml = generatePolicy({
  template: 'sensitive_file',
  mode: 'enforce',
  name: 'block-docker-config',
  enabled: true,
  filePaths: ['/etc/docker/daemon.json'],
  processNames: ['vim'],
  userMatchMode: 'exclude_root',
});

assertIncludes(sensitiveFileYaml, '- "diting-sudo-pre-escalation"');
assertIncludes(sensitiveFileYaml, '- index: 0\n        operator: Postfix\n        values:\n            - "sudo"');
assertIncludes(sensitiveFileYaml, '- index: 1\n        operator: Postfix\n        values:\n            - "vim"');
assertIncludes(sensitiveFileYaml, '- index: 2\n        operator: Equal\n        values:\n            - "/etc/docker/daemon.json"');
assertIncludes(sensitiveFileYaml, 'resolve: "cred.uid.val"');

const suspiciousProcessYaml = generatePolicy({
  template: 'suspicious_process',
  mode: 'enforce',
  name: 'block-suspicious-shell',
  enabled: true,
  processNames: ['bash'],
  userMatchMode: 'exclude_root',
});

assertIncludes(suspiciousProcessYaml, '- "diting-sudo-pre-escalation"');
assertIncludes(suspiciousProcessYaml, '- index: 1\n        operator: Postfix\n        values:\n            - "bash"');
