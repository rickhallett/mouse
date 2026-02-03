package tools

import "strings"

type Policy struct {
	allow    map[string]struct{}
	deny     map[string]struct{}
	allowAll bool
}

func NewPolicy(allowList, denyList []string) *Policy {
	policy := &Policy{
		allow: make(map[string]struct{}),
		deny:  make(map[string]struct{}),
	}
	for _, item := range allowList {
		normal := normalize(item)
		if normal == "" {
			continue
		}
		policy.allow[normal] = struct{}{}
	}
	for _, item := range denyList {
		normal := normalize(item)
		if normal == "" {
			continue
		}
		policy.deny[normal] = struct{}{}
	}
	if len(policy.allow) == 0 {
		policy.allowAll = true
	}
	return policy
}

func (p *Policy) Allowed(tool string) bool {
	if p == nil {
		return false
	}
	name := normalize(tool)
	if name == "" {
		return false
	}
	if _, blocked := p.deny[name]; blocked {
		return false
	}
	if p.allowAll {
		return true
	}
	_, ok := p.allow[name]
	return ok
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
