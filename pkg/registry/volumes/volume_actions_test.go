package volumes

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupVolumeActionsToolWithMocks(storageActions *MockStorageActionsService) *VolumeActionsTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{StorageActions: storageActions}, nil
	}
	return NewVolumeActionsTool(client)
}

func TestVolumeActionsTool_attachVolume(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.Action{ID: 2001, Status: "completed"}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockStorageActionsService)
		expectError bool
	}{
		{
			name: "Successful attach",
			args: map[string]any{"VolumeID": "123", "DropletID": float64(456)},
			mockSetup: func(m *MockStorageActionsService) {
				m.EXPECT().Attach(gomock.Any(), "123", 456).Return(testAction, nil, nil).Times(1)
			},
			expectError: false,
		},
		{
			name:        "Missing VolumeID",
			args:        map[string]any{"DropletID": float64(456)},
			expectError: true,
		},
		{
			name:        "Missing DropletID",
			args:        map[string]any{"VolumeID": "123"},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{"VolumeID": "123", "DropletID": float64(456)},
			mockSetup: func(m *MockStorageActionsService) {
				m.EXPECT().Attach(gomock.Any(), "123", 456).Return(nil, nil, errors.New("api error")).Times(1)
			},
			expectError: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockStorageActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupVolumeActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.attachVolume(context.Background(), req)

			if tc.expectError {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out godo.Action
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, testAction.ID, out.ID)
			require.Equal(t, testAction.Status, out.Status)
		})
	}
}

func TestVolumeActionsTool_detachVolume(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.Action{ID: 2002, Status: "in-progress", Type: "detach_volume"}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockStorageActionsService)
		expectError bool
	}{
		{
			name: "Successful detach",
			args: map[string]any{"VolumeID": "123", "DropletID": float64(456)},
			mockSetup: func(m *MockStorageActionsService) {
				m.EXPECT().DetachByDropletID(gomock.Any(), "123", 456).Return(testAction, nil, nil).Times(1)
			},
		},
		{
			name:        "Missing VolumeID",
			args:        map[string]any{"DropletID": float64(456)},
			expectError: true,
		},
		{
			name:        "Missing DropletID",
			args:        map[string]any{"VolumeID": "123"},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{"VolumeID": "123", "DropletID": float64(456)},
			mockSetup: func(m *MockStorageActionsService) {
				m.EXPECT().DetachByDropletID(gomock.Any(), "123", 456).Return(nil, nil, errors.New("api error")).Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockStorageActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupVolumeActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.detachVolume(context.Background(), req)

			if tc.expectError {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out godo.Action
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, testAction.ID, out.ID)
			require.Equal(t, testAction.Type, out.Type)
		})
	}
}

func TestVolumeActionsTool_getVolumeAction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.Action{ID: 3001, Status: "completed", Type: "attach_volume"}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockStorageActionsService)
		expectError bool
	}{
		{
			name: "Successful get",
			args: map[string]any{"VolumeID": "vol-123", "ActionID": float64(3001)},
			mockSetup: func(m *MockStorageActionsService) {
				m.EXPECT().Get(gomock.Any(), "vol-123", 3001).Return(testAction, nil, nil).Times(1)
			},
		},
		{
			name:        "Missing VolumeID",
			args:        map[string]any{"ActionID": float64(3001)},
			expectError: true,
		},
		{
			name:        "Missing ActionID",
			args:        map[string]any{"VolumeID": "vol-123"},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{"VolumeID": "vol-123", "ActionID": float64(3001)},
			mockSetup: func(m *MockStorageActionsService) {
				m.EXPECT().Get(gomock.Any(), "vol-123", 3001).Return(nil, nil, errors.New("api error")).Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockStorageActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupVolumeActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getVolumeAction(context.Background(), req)

			if tc.expectError {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out godo.Action
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, testAction.ID, out.ID)
			require.Equal(t, testAction.Status, out.Status)
		})
	}
}

func TestVolumeActionsTool_listVolumeActions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testActions := []godo.Action{
		{ID: 4001, Type: "attach_volume", Status: "completed"},
		{ID: 4002, Type: "resize", Status: "in-progress"},
	}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockStorageActionsService)
		expectError bool
		wantIDs     []float64
	}{
		{
			name: "Successful list defaults",
			args: map[string]any{"VolumeID": "vol-123"},
			mockSetup: func(m *MockStorageActionsService) {
				m.EXPECT().List(gomock.Any(), "vol-123", &godo.ListOptions{Page: 1, PerPage: 50}).Return(testActions, nil, nil).Times(1)
			},
			wantIDs: []float64{4001, 4002},
		},
		{
			name: "Successful list with pagination",
			args: map[string]any{"VolumeID": "vol-123", "Page": float64(2), "PerPage": float64(100)},
			mockSetup: func(m *MockStorageActionsService) {
				m.EXPECT().List(gomock.Any(), "vol-123", &godo.ListOptions{Page: 2, PerPage: 100}).Return(testActions, nil, nil).Times(1)
			},
			wantIDs: []float64{4001, 4002},
		},
		{
			name:        "Missing VolumeID",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{"VolumeID": "vol-123"},
			mockSetup: func(m *MockStorageActionsService) {
				m.EXPECT().List(gomock.Any(), "vol-123", &godo.ListOptions{Page: 1, PerPage: 50}).Return(nil, nil, errors.New("api error")).Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockStorageActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupVolumeActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.listVolumeActions(context.Background(), req)

			if tc.expectError {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out []godo.Action
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Len(t, out, len(tc.wantIDs))
			for i, id := range tc.wantIDs {
				require.Equal(t, int(id), out[i].ID)
			}
		})
	}
}

func TestVolumeActionsTool_resizeVolume(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.Action{ID: 5001, Status: "in-progress", Type: "resize"}
	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockStorageActionsService)
		expectError bool
	}{
		{
			name: "Successful resize",
			args: map[string]any{"VolumeID": "vol-123", "SizeGigaBytes": float64(20), "Region": "nyc1"},
			mockSetup: func(m *MockStorageActionsService) {
				m.EXPECT().Resize(gomock.Any(), "vol-123", 20, "nyc1").Return(testAction, nil, nil).Times(1)
			},
		},
		{
			name:        "Missing VolumeID",
			args:        map[string]any{"SizeGigaBytes": float64(20), "Region": "nyc1"},
			expectError: true,
		},
		{
			name:        "Missing SizeGigaBytes",
			args:        map[string]any{"VolumeID": "vol-123", "Region": "nyc1"},
			expectError: true,
		},
		{
			name:        "Missing Region",
			args:        map[string]any{"VolumeID": "vol-123", "SizeGigaBytes": float64(20)},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{"VolumeID": "vol-123", "SizeGigaBytes": float64(20), "Region": "nyc1"},
			mockSetup: func(m *MockStorageActionsService) {
				m.EXPECT().Resize(gomock.Any(), "vol-123", 20, "nyc1").Return(nil, nil, errors.New("api error")).Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockStorageActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupVolumeActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.resizeVolume(context.Background(), req)

			if tc.expectError {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out godo.Action
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, testAction.ID, out.ID)
			require.Equal(t, testAction.Type, out.Type)
		})
	}
}
