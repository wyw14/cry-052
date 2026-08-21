package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/wyw14/cry052/internal/domain"
)

type SecretResolver interface {
	ResolveSecret(context.Context, string) ([]byte, error)
}

type HashTransformer struct {
	secrets SecretResolver
}

func NewHashTransformer(secrets SecretResolver) *HashTransformer {
	return &HashTransformer{secrets: secrets}
}

func (*HashTransformer) Kind() domain.StrategyKind { return domain.StrategyHash }

func (t *HashTransformer) Transform(ctx context.Context, input TransformInput) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	secretReference := input.Parameters["secret_reference"]
	if secretReference == "" {
		return nil, fmt.Errorf("hash requires a local secret reference")
	}
	secret, err := t.secrets.ResolveSecret(ctx, secretReference)
	if err != nil {
		return nil, fmt.Errorf("resolve hash secret: %w", err)
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = fmt.Fprint(mac, input.Value)
	digest := hex.EncodeToString(mac.Sum(nil))
	if length, err := positiveInt(input.Parameters, "length", 64); err != nil {
		return nil, err
	} else if length > 0 && length < len(digest) {
		digest = digest[:length]
	}
	return digest, nil
}
