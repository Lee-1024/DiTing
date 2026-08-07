export interface AuditSelectOption {
  value: string;
  label: string;
}

interface AuditUserIdentity {
  username: string;
}

interface AuditHostIdentity {
  hostId?: string;
  hostName?: string;
  nodeName?: string;
}

export function buildUserOptions(items: AuditUserIdentity[]): AuditSelectOption[] {
  const seen = new Set<string>();
  const options: AuditSelectOption[] = [];

  items.forEach((item) => {
    const username = item.username.trim();
    if (!username || seen.has(username)) {
      return;
    }
    seen.add(username);
    options.push({ value: username, label: username });
  });

  return options;
}

export function buildHostOptions(items: AuditHostIdentity[]): AuditSelectOption[] {
  const seen = new Set<string>();
  const options: AuditSelectOption[] = [];

  items.forEach((item) => {
    const hostId = item.hostId?.trim() ?? '';
    const nodeName = item.nodeName?.trim() ?? '';
    const hostName = item.hostName?.trim() ?? '';
    const value = hostId || nodeName || hostName;
    if (!value || seen.has(value)) {
      return;
    }

    const identities = [hostName, nodeName, hostId].filter((identity, index, values) => (
      Boolean(identity) && values.indexOf(identity) === index
    ));
    seen.add(value);
    options.push({ value, label: identities.join(' / ') });
  });

  return options;
}
