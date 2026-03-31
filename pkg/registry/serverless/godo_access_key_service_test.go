package serverless

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGodoAccessKeyService_CreateAccessKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockFunctionsService(ctrl)

	mock.EXPECT().GetNamespace(gomock.Any(), "fn-ns-1").Return(&godo.FunctionsNamespace{
		Namespace: "fn-ns-1",
		ApiHost:   "https://faas.example.com/",
	}, nil, nil)

	mock.EXPECT().CreateAccessKey(gomock.Any(), "fn-ns-1", &godo.FunctionsAccessKeyCreateRequest{
		Name:      "mcp-server",
		ExpiresIn: "2h",
	}).Return(&godo.FunctionsAccessKey{
		ID:        "dof_v1_abc123",
		Name:      "mcp-server",
		Secret:    "dof_v1_abc123:secret456",
		ExpiresAt: time.Date(2026, 3, 26, 14, 0, 0, 0, time.UTC),
	}, nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{Functions: mock}, nil
	}

	svc := NewGodoAccessKeyService(client)
	ak, err := svc.CreateAccessKey(context.Background(), "fn-ns-1", &AccessKeyCreateRequest{
		Name:      "mcp-server",
		ExpiresIn: "2h",
	})

	require.NoError(t, err)
	require.Equal(t, "dof_v1_abc123", ak.ID)
	require.Equal(t, "dof_v1_abc123:secret456", ak.Secret)
	require.Equal(t, "https://faas.example.com/", ak.APIHost)
	require.Equal(t, "mcp-server", ak.Name)
	require.Equal(t, time.Date(2026, 3, 26, 14, 0, 0, 0, time.UTC), ak.ExpiresAt)
}

func TestGodoAccessKeyService_GetNamespaceFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockFunctionsService(ctrl)
	mock.EXPECT().GetNamespace(gomock.Any(), "fn-ns-1").Return(nil, nil, errors.New("not found"))

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{Functions: mock}, nil
	}

	svc := NewGodoAccessKeyService(client)
	_, err := svc.CreateAccessKey(context.Background(), "fn-ns-1", &AccessKeyCreateRequest{
		Name:      "mcp-server",
		ExpiresIn: "2h",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get namespace")
}

func TestGodoAccessKeyService_CreateAccessKeyFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockFunctionsService(ctrl)
	mock.EXPECT().GetNamespace(gomock.Any(), "fn-ns-1").Return(&godo.FunctionsNamespace{
		Namespace: "fn-ns-1",
		ApiHost:   "https://faas.example.com",
	}, nil, nil)
	mock.EXPECT().CreateAccessKey(gomock.Any(), "fn-ns-1", gomock.Any()).Return(nil, nil, errors.New("quota exceeded"))

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{Functions: mock}, nil
	}

	svc := NewGodoAccessKeyService(client)
	_, err := svc.CreateAccessKey(context.Background(), "fn-ns-1", &AccessKeyCreateRequest{
		Name:      "mcp-server",
		ExpiresIn: "2h",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create access key")
}

func TestGodoAccessKeyService_MissingApiHost(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockFunctionsService(ctrl)
	mock.EXPECT().GetNamespace(gomock.Any(), "fn-ns-1").Return(&godo.FunctionsNamespace{
		Namespace: "fn-ns-1",
		ApiHost:   "",
	}, nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{Functions: mock}, nil
	}

	svc := NewGodoAccessKeyService(client)
	_, err := svc.CreateAccessKey(context.Background(), "fn-ns-1", &AccessKeyCreateRequest{
		Name:      "mcp-server",
		ExpiresIn: "2h",
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "missing api_host")
}
