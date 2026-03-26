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

func setupTriggerToolWithMock(mockFunctions godo.FunctionsService) *TriggerTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Functions: mockFunctions,
		}, nil
	}
	return NewTriggerTool(client)
}

func TestTriggerTool_list(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testTriggers := []godo.FunctionsTrigger{
		{Name: "trigger-1", Function: "pkg/func1", Type: "SCHEDULED"},
		{Name: "trigger-2", Function: "pkg/func2", Type: "SCHEDULED"},
	}

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
				m.EXPECT().ListTriggers(gomock.Any(), "uuid-1").Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"NamespaceID": "uuid-1"},
			mockSetup: func(m *MockFunctionsService) {
				m.EXPECT().ListTriggers(gomock.Any(), "uuid-1").Return(testTriggers, nil, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockFunctionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupTriggerToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
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
			require.Contains(t, textContent.Text, "trigger-1")
			require.Contains(t, textContent.Text, "trigger-2")
		})
	}
}

func TestTriggerTool_get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testTrigger := &godo.FunctionsTrigger{Name: "my-trigger", Function: "pkg/func1", Type: "SCHEDULED"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockFunctionsService)
		expectError bool
	}{
		{
			name:        "missing NamespaceID",
			args:        map[string]any{"TriggerName": "my-trigger"},
			expectError: true,
		},
		{
			name:        "missing TriggerName",
			args:        map[string]any{"NamespaceID": "uuid-1"},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"NamespaceID": "uuid-1", "TriggerName": "my-trigger"},
			mockSetup: func(m *MockFunctionsService) {
				m.EXPECT().GetTrigger(gomock.Any(), "uuid-1", "my-trigger").Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"NamespaceID": "uuid-1", "TriggerName": "my-trigger"},
			mockSetup: func(m *MockFunctionsService) {
				m.EXPECT().GetTrigger(gomock.Any(), "uuid-1", "my-trigger").Return(testTrigger, nil, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockFunctionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupTriggerToolWithMock(mock)
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
			require.Contains(t, textContent.Text, "my-trigger")
		})
	}
}

func TestTriggerTool_create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testTrigger := &godo.FunctionsTrigger{Name: "my-trigger", Function: "pkg/func1", Type: "SCHEDULED", IsEnabled: true}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockFunctionsService)
		expectError bool
	}{
		{
			name:        "missing NamespaceID",
			args:        map[string]any{"Name": "t", "Function": "f", "Type": "SCHEDULED"},
			expectError: true,
		},
		{
			name:        "missing Name",
			args:        map[string]any{"NamespaceID": "uuid-1", "Function": "f", "Type": "SCHEDULED"},
			expectError: true,
		},
		{
			name:        "missing Function",
			args:        map[string]any{"NamespaceID": "uuid-1", "Name": "t", "Type": "SCHEDULED"},
			expectError: true,
		},
		{
			name:        "missing Type",
			args:        map[string]any{"NamespaceID": "uuid-1", "Name": "t", "Function": "f"},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"NamespaceID": "uuid-1", "Name": "my-trigger", "Function": "pkg/func1", "Type": "SCHEDULED", "IsEnabled": true},
			mockSetup: func(m *MockFunctionsService) {
				m.EXPECT().CreateTrigger(gomock.Any(), "uuid-1", gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"NamespaceID": "uuid-1", "Name": "my-trigger", "Function": "pkg/func1", "Type": "SCHEDULED", "IsEnabled": true},
			mockSetup: func(m *MockFunctionsService) {
				m.EXPECT().CreateTrigger(gomock.Any(), "uuid-1", gomock.Any()).DoAndReturn(
					func(_ context.Context, nsID string, req *godo.FunctionsTriggerCreateRequest) (*godo.FunctionsTrigger, *godo.Response, error) {
						require.Equal(t, "my-trigger", req.Name)
						require.Equal(t, "pkg/func1", req.Function)
						require.Equal(t, "SCHEDULED", req.Type)
						require.True(t, req.IsEnabled)
						return testTrigger, nil, nil
					},
				)
			},
		},
		{
			name: "success with cron",
			args: map[string]any{"NamespaceID": "uuid-1", "Name": "my-trigger", "Function": "pkg/func1", "Type": "SCHEDULED", "IsEnabled": true, "Cron": "*/5 * * * *"},
			mockSetup: func(m *MockFunctionsService) {
				m.EXPECT().CreateTrigger(gomock.Any(), "uuid-1", gomock.Any()).DoAndReturn(
					func(_ context.Context, nsID string, req *godo.FunctionsTriggerCreateRequest) (*godo.FunctionsTrigger, *godo.Response, error) {
						require.NotNil(t, req.ScheduledDetails)
						require.Equal(t, "*/5 * * * *", req.ScheduledDetails.Cron)
						return testTrigger, nil, nil
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
			tool := setupTriggerToolWithMock(mock)
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

func TestTriggerTool_update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	testTrigger := &godo.FunctionsTrigger{Name: "my-trigger", Function: "pkg/func1", Type: "SCHEDULED", IsEnabled: false}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockFunctionsService)
		expectError bool
	}{
		{
			name:        "missing NamespaceID",
			args:        map[string]any{"TriggerName": "my-trigger"},
			expectError: true,
		},
		{
			name:        "missing TriggerName",
			args:        map[string]any{"NamespaceID": "uuid-1"},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"NamespaceID": "uuid-1", "TriggerName": "my-trigger", "IsEnabled": false},
			mockSetup: func(m *MockFunctionsService) {
				m.EXPECT().UpdateTrigger(gomock.Any(), "uuid-1", "my-trigger", gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"NamespaceID": "uuid-1", "TriggerName": "my-trigger", "IsEnabled": false},
			mockSetup: func(m *MockFunctionsService) {
				m.EXPECT().UpdateTrigger(gomock.Any(), "uuid-1", "my-trigger", gomock.Any()).DoAndReturn(
					func(_ context.Context, nsID, triggerName string, req *godo.FunctionsTriggerUpdateRequest) (*godo.FunctionsTrigger, *godo.Response, error) {
						require.NotNil(t, req.IsEnabled)
						require.False(t, *req.IsEnabled)
						return testTrigger, nil, nil
					},
				)
			},
		},
		{
			name: "success with cron update",
			args: map[string]any{"NamespaceID": "uuid-1", "TriggerName": "my-trigger", "Cron": "0 * * * *"},
			mockSetup: func(m *MockFunctionsService) {
				m.EXPECT().UpdateTrigger(gomock.Any(), "uuid-1", "my-trigger", gomock.Any()).DoAndReturn(
					func(_ context.Context, nsID, triggerName string, req *godo.FunctionsTriggerUpdateRequest) (*godo.FunctionsTrigger, *godo.Response, error) {
						require.NotNil(t, req.ScheduledDetails)
						require.Equal(t, "0 * * * *", req.ScheduledDetails.Cron)
						return testTrigger, nil, nil
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
			tool := setupTriggerToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.update(context.Background(), req)
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

func TestTriggerTool_delete(t *testing.T) {
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
			args:        map[string]any{"TriggerName": "my-trigger"},
			expectError: true,
		},
		{
			name:        "missing TriggerName",
			args:        map[string]any{"NamespaceID": "uuid-1"},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"NamespaceID": "uuid-1", "TriggerName": "my-trigger"},
			mockSetup: func(m *MockFunctionsService) {
				m.EXPECT().DeleteTrigger(gomock.Any(), "uuid-1", "my-trigger").Return(nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"NamespaceID": "uuid-1", "TriggerName": "my-trigger"},
			mockSetup: func(m *MockFunctionsService) {
				m.EXPECT().DeleteTrigger(gomock.Any(), "uuid-1", "my-trigger").Return(nil, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockFunctionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupTriggerToolWithMock(mock)
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
