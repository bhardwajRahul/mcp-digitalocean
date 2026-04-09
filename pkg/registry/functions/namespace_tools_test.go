package functions

import (
	"context"
	"errors"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupNamespaceTool(mock *MockFunctionsService) *NamespaceTool {
	return NewNamespaceTool(mockClient(mock))
}

func TestNamespaceTool_ListNamespaces(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockFunctionsService(ctrl)
	tool := setupNamespaceTool(mock)

	tests := []struct {
		name        string
		setup       func()
		expectError bool
	}{
		{
			name: "success",
			setup: func() {
				mock.EXPECT().ListNamespaces(gomock.Any()).
					Return([]godo.FunctionsNamespace{
						{Namespace: "ns-1", Label: "test-1", Region: "nyc1"},
						{Namespace: "ns-2", Label: "test-2", Region: "sfo1"},
					}, nil, nil)
			},
		},
		{
			name: "API error",
			setup: func() {
				mock.EXPECT().ListNamespaces(gomock.Any()).
					Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			resp, err := tool.listNamespaces(context.Background(), mcp.CallToolRequest{})
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			content := resp.Content[0].(mcp.TextContent).Text
			require.Contains(t, content, "ns-1")
			require.Contains(t, content, "ns-2")
		})
	}
}

func TestNamespaceTool_GetNamespace(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockFunctionsService(ctrl)
	tool := setupNamespaceTool(mock)

	tests := []struct {
		name        string
		args        map[string]interface{}
		setup       func()
		expectError bool
	}{
		{
			name: "success",
			args: map[string]interface{}{"NamespaceID": "ns-uuid-1"},
			setup: func() {
				mock.EXPECT().GetNamespace(gomock.Any(), "ns-uuid-1").
					Return(&godo.FunctionsNamespace{
						Namespace: "ns-uuid-1", Label: "my-ns", Region: "nyc1", ApiHost: "https://faas.example.com",
					}, nil, nil)
			},
		},
		{
			name:        "missing namespace ID",
			args:        map[string]interface{}{},
			setup:       func() {},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]interface{}{"NamespaceID": "ns-uuid-1"},
			setup: func() {
				mock.EXPECT().GetNamespace(gomock.Any(), "ns-uuid-1").
					Return(nil, nil, errors.New("not found"))
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			req := mcp.CallToolRequest{}
			req.Params.Arguments = tc.args
			resp, err := tool.getNamespace(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			content := resp.Content[0].(mcp.TextContent).Text
			require.Contains(t, content, "ns-uuid-1")
		})
	}
}

func TestNamespaceTool_CreateNamespace(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockFunctionsService(ctrl)
	tool := setupNamespaceTool(mock)

	tests := []struct {
		name        string
		args        map[string]interface{}
		setup       func()
		expectError bool
	}{
		{
			name: "success",
			args: map[string]interface{}{"Label": "my-ns", "Region": "nyc1"},
			setup: func() {
				mock.EXPECT().CreateNamespace(gomock.Any(), &godo.FunctionsNamespaceCreateRequest{
					Label: "my-ns", Region: "nyc1",
				}).Return(&godo.FunctionsNamespace{
					Namespace: "ns-new", Label: "my-ns", Region: "nyc1",
				}, nil, nil)
			},
		},
		{
			name:        "missing label",
			args:        map[string]interface{}{"Region": "nyc1"},
			setup:       func() {},
			expectError: true,
		},
		{
			name:        "missing region",
			args:        map[string]interface{}{"Label": "my-ns"},
			setup:       func() {},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]interface{}{"Label": "my-ns", "Region": "nyc1"},
			setup: func() {
				mock.EXPECT().CreateNamespace(gomock.Any(), gomock.Any()).
					Return(nil, nil, errors.New("quota exceeded"))
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			req := mcp.CallToolRequest{}
			req.Params.Arguments = tc.args
			resp, err := tool.createNamespace(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			content := resp.Content[0].(mcp.TextContent).Text
			require.Contains(t, content, "ns-new")
		})
	}
}

func TestNamespaceTool_DeleteNamespace(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockFunctionsService(ctrl)
	tool := setupNamespaceTool(mock)

	tests := []struct {
		name        string
		args        map[string]interface{}
		setup       func()
		expectError bool
	}{
		{
			name: "success",
			args: map[string]interface{}{"NamespaceID": "ns-uuid-1"},
			setup: func() {
				mock.EXPECT().DeleteNamespace(gomock.Any(), "ns-uuid-1").
					Return(nil, nil)
			},
		},
		{
			name:        "missing namespace ID",
			args:        map[string]interface{}{},
			setup:       func() {},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]interface{}{"NamespaceID": "ns-uuid-1"},
			setup: func() {
				mock.EXPECT().DeleteNamespace(gomock.Any(), "ns-uuid-1").
					Return(nil, errors.New("forbidden"))
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup()
			req := mcp.CallToolRequest{}
			req.Params.Arguments = tc.args
			resp, err := tool.deleteNamespace(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			content := resp.Content[0].(mcp.TextContent).Text
			require.Contains(t, content, "deleted successfully")
		})
	}
}

func TestNamespaceTool_ListAccessKeys(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockFunctionsService(ctrl)
	tool := setupNamespaceTool(mock)

	mock.EXPECT().ListAccessKeys(gomock.Any(), "ns-uuid-1").
		Return([]godo.FunctionsAccessKey{
			{ID: "key-1", Name: "my-key"},
		}, nil, nil)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"NamespaceID": "ns-uuid-1"}
	resp, err := tool.listAccessKeys(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	content := resp.Content[0].(mcp.TextContent).Text
	require.Contains(t, content, "key-1")
}

func TestNamespaceTool_CreateAccessKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockFunctionsService(ctrl)
	tool := setupNamespaceTool(mock)

	mock.EXPECT().CreateAccessKey(gomock.Any(), "ns-uuid-1", &godo.FunctionsAccessKeyCreateRequest{
		Name: "test-key", ExpiresIn: "24h",
	}).Return(&godo.FunctionsAccessKey{
		ID: "new-key-id", Secret: "new-secret", Name: "test-key",
	}, nil, nil)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"NamespaceID": "ns-uuid-1",
		"Name":        "test-key",
		"ExpiresIn":   "24h",
	}
	resp, err := tool.createAccessKey(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	content := resp.Content[0].(mcp.TextContent).Text
	require.Contains(t, content, "new-key-id")
}

func TestNamespaceTool_DeleteAccessKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockFunctionsService(ctrl)
	tool := setupNamespaceTool(mock)

	mock.EXPECT().DeleteAccessKey(gomock.Any(), "ns-uuid-1", "key-to-delete").
		Return(nil, nil)

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"NamespaceID": "ns-uuid-1",
		"KeyID":       "key-to-delete",
	}
	resp, err := tool.deleteAccessKey(context.Background(), req)
	require.NoError(t, err)
	require.False(t, resp.IsError)
	content := resp.Content[0].(mcp.TextContent).Text
	require.Contains(t, content, "deleted successfully")
}
