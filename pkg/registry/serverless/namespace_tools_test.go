package serverless

import (
	"context"
	"errors"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupNamespaceToolWithMock(mockFunctions godo.FunctionsService) *NamespaceTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Functions: mockFunctions,
		}, nil
	}
	return NewNamespaceTool(client)
}

func TestNamespaceTool_list(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testNamespaces := []godo.FunctionsNamespace{
		{Label: "ns1", Region: "nyc1", UUID: "uuid-1"},
		{Label: "ns2", Region: "sfo3", UUID: "uuid-2"},
	}

	tests := []struct {
		name        string
		mockSetup   func(*MockFunctionsService)
		expectError bool
	}{
		{
			name: "api error",
			mockSetup: func(m *MockFunctionsService) {
				m.EXPECT().ListNamespaces(gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			mockSetup: func(m *MockFunctionsService) {
				m.EXPECT().ListNamespaces(gomock.Any()).Return(testNamespaces, nil, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockFunctionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupNamespaceToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{}}}
			resp, err := tool.list(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
			textContent, ok := resp.Content[0].(mcp.TextContent)
			require.True(t, ok)
			require.Contains(t, textContent.Text, "ns1")
			require.Contains(t, textContent.Text, "ns2")
		})
	}
}

func TestNamespaceTool_get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testNamespace := &godo.FunctionsNamespace{Label: "my-ns", Region: "nyc1", UUID: "uuid-1"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockFunctionsService)
		expectError bool
	}{
		{
			name:        "missing NamespaceID",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"NamespaceID": "uuid-1"},
			mockSetup: func(m *MockFunctionsService) {
				m.EXPECT().GetNamespace(gomock.Any(), "uuid-1").Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"NamespaceID": "uuid-1"},
			mockSetup: func(m *MockFunctionsService) {
				m.EXPECT().GetNamespace(gomock.Any(), "uuid-1").Return(testNamespace, nil, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockFunctionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupNamespaceToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.get(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
			textContent, ok := resp.Content[0].(mcp.TextContent)
			require.True(t, ok)
			require.Contains(t, textContent.Text, "my-ns")
		})
	}
}

func TestNamespaceTool_create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testNamespace := &godo.FunctionsNamespace{Label: "my-ns", Region: "nyc1", UUID: "uuid-1"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockFunctionsService)
		expectError bool
	}{
		{
			name:        "missing Label",
			args:        map[string]any{"Region": "nyc1"},
			expectError: true,
		},
		{
			name:        "missing Region",
			args:        map[string]any{"Label": "my-ns"},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"Label": "my-ns", "Region": "nyc1"},
			mockSetup: func(m *MockFunctionsService) {
				m.EXPECT().CreateNamespace(gomock.Any(), gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"Label": "my-ns", "Region": "nyc1"},
			mockSetup: func(m *MockFunctionsService) {
				m.EXPECT().CreateNamespace(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, req *godo.FunctionsNamespaceCreateRequest) (*godo.FunctionsNamespace, *godo.Response, error) {
						require.Equal(t, "my-ns", req.Label)
						require.Equal(t, "nyc1", req.Region)
						return testNamespace, nil, nil
					},
				)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockFunctionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupNamespaceToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.create(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
		})
	}
}

func TestNamespaceTool_delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockFunctionsService)
		expectError bool
	}{
		{
			name:        "missing NamespaceID",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"NamespaceID": "uuid-1"},
			mockSetup: func(m *MockFunctionsService) {
				m.EXPECT().DeleteNamespace(gomock.Any(), "uuid-1").Return(nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"NamespaceID": "uuid-1"},
			mockSetup: func(m *MockFunctionsService) {
				m.EXPECT().DeleteNamespace(gomock.Any(), "uuid-1").Return(nil, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockFunctionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupNamespaceToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.delete(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.NotEmpty(t, resp.Content)
		})
	}
}
