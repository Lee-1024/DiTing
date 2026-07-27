interface HostIdentity {
  hostName?: string;
  nodeName?: string;
  hostId?: string;
}

const containerLikeIdPattern = /^[a-f0-9]{12,64}$/i;

export function isContainerLikeId(value?: string) {
  const text = value?.trim();
  return Boolean(text && containerLikeIdPattern.test(text));
}

export function displayHostIdentity(identity: HostIdentity, fallback = '-') {
  const candidates = [identity.hostName, identity.nodeName, identity.hostId];
  return candidates.find((value) => value && !isContainerLikeId(value)) || fallback;
}
