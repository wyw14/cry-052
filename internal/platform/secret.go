package platform

import (
	"context"
	"fmt"
	"sync"
)

type SecretVault struct {
	mu     sync.RWMutex
	values map[string][]byte
}

func NewSecretVault(values map[string][]byte) *SecretVault {
	copied := make(map[string][]byte, len(values))
	for key, value := range values {
		copied[key] = append([]byte(nil), value...)
	}
	return &SecretVault{values: copied}
}

func (v *SecretVault) ResolveSecret(ctx context.Context, reference string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	value, ok := v.values[reference]
	if !ok {
		return nil, fmt.Errorf("unknown local secret reference")
	}
	return append([]byte(nil), value...), nil
}

func (v *SecretVault) Put(reference string, value []byte) error {
	if reference == "" || len(value) < 16 {
		return fmt.Errorf("secret reference or value is invalid")
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.values[reference] = append([]byte(nil), value...)
	return nil
}
