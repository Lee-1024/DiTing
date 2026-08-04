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
