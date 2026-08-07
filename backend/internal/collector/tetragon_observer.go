package collector

import (
	"fmt"
	"strings"
)

const tetragonObserverPolicyName = "diting-apparmor-observer"

// GenerateTetragonObserverPolicy creates a monitor-only policy. AppArmor remains
// the enforcement authority; Tetragon only reports the security hook result.
func GenerateTetragonObserverPolicy(paths []string) (string, error) {
	normalized, err := normalizeAppArmorPaths(paths)
	if err != nil {
		return "", err
	}
	var exact strings.Builder
	var children strings.Builder
	for _, path := range normalized {
		fmt.Fprintf(&exact, "            - %q\n", path)
		fmt.Fprintf(&children, "            - %q\n", strings.TrimSuffix(path, "/")+"/")
	}
	return fmt.Sprintf(`apiVersion: cilium.io/v1alpha1
kind: TracingPolicy
metadata:
  name: diting-apparmor-observer
spec:
  kprobes:
    - call: security_file_open
      syscall: false
      return: true
      args:
        - index: 0
          type: file
      returnArg:
        index: 0
        type: int
      tags:
        - diting-apparmor-observer
      selectors:
        - matchArgs:
            - index: 0
              operator: Equal
              values:
%s          matchActions:
            - action: Post
        - matchArgs:
            - index: 0
              operator: Prefix
              values:
%s          matchActions:
            - action: Post
`, exact.String(), children.String()), nil
}
