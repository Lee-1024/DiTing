export const severityOptions = [
  { value: 'info', label: '提示' },
  { value: 'low', label: '低危' },
  { value: 'medium', label: '中危' },
  { value: 'high', label: '高危' },
  { value: 'critical', label: '严重' },
];

export const eventTypeOptions = [
  { value: 'process_exec', label: '进程执行' },
  { value: 'process_exit', label: '进程退出' },
  { value: 'process_kprobe', label: '内核探针' },
  { value: 'file_access', label: '文件访问' },
  { value: 'network_connect', label: '网络连接' },
];

export const ruleFieldOptions = [
  { value: 'cmdline', label: '命令行' },
  { value: 'process_name', label: '进程名' },
  { value: 'username', label: '执行用户' },
  { value: 'login_username', label: '登录用户' },
  { value: 'host_id', label: 'Host ID' },
  { value: 'host_name', label: '主机名' },
  { value: 'node_name', label: '节点名' },
  { value: 'namespace', label: 'Namespace' },
  { value: 'pod_name', label: 'Pod' },
  { value: 'container_id', label: '容器 ID' },
  { value: 'binary_path', label: '二进制路径' },
  { value: 'file_path', label: '文件路径' },
  { value: 'file_operation', label: '文件操作' },
  { value: 'dst_ip', label: '目标 IP' },
  { value: 'dst_port', label: '目标端口' },
  { value: 'protocol', label: '网络协议' },
  { value: 'domain', label: '域名' },
  { value: 'event_type', label: '事件类型' },
];

export const ruleOperatorOptions = [
  { value: 'contains', label: '包含' },
  { value: 'eq', label: '等于' },
  { value: 'neq', label: '不等于' },
  { value: 'prefix', label: '前缀匹配' },
  { value: 'suffix', label: '后缀匹配' },
  { value: 'in', label: '属于列表' },
  { value: 'regex', label: '正则匹配' },
];

// optionLabel 处理 option Label 相关逻辑。
export function optionLabel(options: Array<{ value: string; label: string }>, value?: string) {
  if (!value) {
    return '';
  }
  return options.find((option) => option.value === value)?.label ?? value;
}

// severityLabel 处理 severity Label 相关逻辑。
export function severityLabel(value?: string) {
  return optionLabel(severityOptions, value);
}

// eventTypeLabel 处理 event Type Label 相关逻辑。
export function eventTypeLabel(value?: string) {
  return optionLabel(eventTypeOptions, value);
}

// ruleFieldLabel 处理 rule Field Label 相关逻辑。
export function ruleFieldLabel(value?: string) {
  return optionLabel(ruleFieldOptions, value);
}

// ruleOperatorLabel 处理 rule Operator Label 相关逻辑。
export function ruleOperatorLabel(value?: string) {
  return optionLabel(ruleOperatorOptions, value);
}
