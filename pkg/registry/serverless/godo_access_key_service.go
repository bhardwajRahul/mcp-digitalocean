package serverless

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
)

// godoAccessKeyService implements AccessKeyService by calling godo's
// FunctionsService. It fetches the namespace (for ApiHost) and creates
// an access key (for credentials) in a single CreateAccessKey call.
type godoAccessKeyService struct {
	client func(ctx context.Context) (*godo.Client, error)
}

// NewGodoAccessKeyService returns an AccessKeyService backed by godo.
func NewGodoAccessKeyService(client func(ctx context.Context) (*godo.Client, error)) AccessKeyService {
	return &godoAccessKeyService{client: client}
}

func (s *godoAccessKeyService) CreateAccessKey(ctx context.Context, namespace string, req *AccessKeyCreateRequest) (*AccessKey, error) {
	client, err := s.client(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	ns, _, err := client.Functions.GetNamespace(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}
	if ns.ApiHost == "" {
		return nil, fmt.Errorf("namespace %s is missing api_host", namespace)
	}

	ak, _, err := client.Functions.CreateAccessKey(ctx, namespace, &godo.FunctionsAccessKeyCreateRequest{
		Name:      req.Name,
		ExpiresIn: req.ExpiresIn,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create access key: %w", err)
	}

	return &AccessKey{
		ID:        ak.ID,
		Secret:    ak.Secret,
		APIHost:   ns.ApiHost,
		ExpiresAt: ak.ExpiresAt,
		Name:      ak.Name,
	}, nil
}
