package nfs

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

func setupNfsActionsToolWithMocks(nfsActionsSvc *MockNfsActionsService) *NfsActionsTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{NfsActions: nfsActionsSvc}, nil
	}
	return NewNfsActionsTool(client)
}

func TestNfsActionsTool_resizeFileShare(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.NfsAction{ID: "1", Status: "in-progress", Type: "resize", ResourceID: "nfs-abc"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockNfsActionsService)
		expectError bool
	}{
		{
			name: "Successful resize",
			args: map[string]any{
				"ShareID":       "nfs-abc",
				"SizeGibibytes": float64(100),
			},
			mockSetup: func(m *MockNfsActionsService) {
				m.EXPECT().
					Resize(gomock.Any(), "nfs-abc", uint64(100), "").
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "Missing ShareID",
			args: map[string]any{
				"SizeGibibytes": float64(100),
			},
			expectError: true,
		},
		{
			name: "Empty ShareID",
			args: map[string]any{
				"ShareID":       "",
				"SizeGibibytes": float64(100),
			},
			expectError: true,
		},
		{
			name: "Missing SizeGibibytes",
			args: map[string]any{
				"ShareID": "nfs-abc",
			},
			expectError: true,
		},
		{
			name: "SizeGibibytes below minimum",
			args: map[string]any{
				"ShareID":       "nfs-abc",
				"SizeGibibytes": float64(10),
			},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{
				"ShareID":       "nfs-abc",
				"SizeGibibytes": float64(100),
			},
			mockSetup: func(m *MockNfsActionsService) {
				m.EXPECT().
					Resize(gomock.Any(), "nfs-abc", uint64(100), "").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockNfsActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupNfsActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.resizeFileShare(context.Background(), req)

			if tc.expectError {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out godo.NfsAction
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, testAction.ID, out.ID)
			require.Equal(t, testAction.Type, out.Type)
			require.Equal(t, testAction.ResourceID, out.ResourceID)
		})
	}
}

func TestNfsActionsTool_snapshotFileShare(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.NfsAction{ID: "2", Status: "in-progress", Type: "snapshot", ResourceID: "nfs-abc"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockNfsActionsService)
		expectError bool
	}{
		{
			name: "Successful snapshot",
			args: map[string]any{
				"ShareID":      "nfs-abc",
				"SnapshotName": "my-snap",
			},
			mockSetup: func(m *MockNfsActionsService) {
				m.EXPECT().
					Snapshot(gomock.Any(), "nfs-abc", "my-snap", "").
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "Successful snapshot without region",
			args: map[string]any{
				"ShareID":      "nfs-abc",
				"SnapshotName": "my-snap",
			},
			mockSetup: func(m *MockNfsActionsService) {
				m.EXPECT().
					Snapshot(gomock.Any(), "nfs-abc", "my-snap", "").
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "Missing ShareID",
			args: map[string]any{
				"SnapshotName": "my-snap",
			},
			expectError: true,
		},
		{
			name: "Missing SnapshotName",
			args: map[string]any{
				"ShareID": "nfs-abc",
			},
			expectError: true,
		},
		{
			name: "Empty SnapshotName",
			args: map[string]any{
				"ShareID":      "nfs-abc",
				"SnapshotName": "",
			},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{
				"ShareID":      "nfs-abc",
				"SnapshotName": "my-snap",
			},
			mockSetup: func(m *MockNfsActionsService) {
				m.EXPECT().
					Snapshot(gomock.Any(), "nfs-abc", "my-snap", "").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockNfsActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupNfsActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.snapshotFileShare(context.Background(), req)

			if tc.expectError {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out godo.NfsAction
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, testAction.ID, out.ID)
			require.Equal(t, testAction.Type, out.Type)
			require.Equal(t, testAction.ResourceID, out.ResourceID)
		})
	}
}

func TestNfsActionsTool_attachFileShare(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.NfsAction{ID: "3", Status: "in-progress", Type: "attach", ResourceID: "nfs-abc"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockNfsActionsService)
		expectError bool
	}{
		{
			name: "Successful attach",
			args: map[string]any{
				"ShareID": "nfs-abc",
				"VpcID":   "vpc-123",
			},
			mockSetup: func(m *MockNfsActionsService) {
				m.EXPECT().
					Attach(gomock.Any(), "nfs-abc", "vpc-123", "").
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "Successful attach without region",
			args: map[string]any{
				"ShareID": "nfs-abc",
				"VpcID":   "vpc-123",
			},
			mockSetup: func(m *MockNfsActionsService) {
				m.EXPECT().
					Attach(gomock.Any(), "nfs-abc", "vpc-123", "").
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "Missing ShareID",
			args: map[string]any{
				"VpcID": "vpc-123",
			},
			expectError: true,
		},
		{
			name: "Missing VpcID",
			args: map[string]any{
				"ShareID": "nfs-abc",
			},
			expectError: true,
		},
		{
			name: "Empty VpcID",
			args: map[string]any{
				"ShareID": "nfs-abc",
				"VpcID":   "",
			},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{
				"ShareID": "nfs-abc",
				"VpcID":   "vpc-123",
			},
			mockSetup: func(m *MockNfsActionsService) {
				m.EXPECT().
					Attach(gomock.Any(), "nfs-abc", "vpc-123", "").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockNfsActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupNfsActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.attachFileShare(context.Background(), req)

			if tc.expectError {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out godo.NfsAction
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, testAction.ID, out.ID)
			require.Equal(t, testAction.Type, out.Type)
			require.Equal(t, testAction.ResourceID, out.ResourceID)
		})
	}
}

func TestNfsActionsTool_detachFileShare(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.NfsAction{ID: "4", Status: "in-progress", Type: "detach", ResourceID: "nfs-abc"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockNfsActionsService)
		expectError bool
	}{
		{
			name: "Successful detach",
			args: map[string]any{
				"ShareID": "nfs-abc",
				"VpcID":   "vpc-123",
			},
			mockSetup: func(m *MockNfsActionsService) {
				m.EXPECT().
					Detach(gomock.Any(), "nfs-abc", "vpc-123", "").
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "Successful detach without region",
			args: map[string]any{
				"ShareID": "nfs-abc",
				"VpcID":   "vpc-123",
			},
			mockSetup: func(m *MockNfsActionsService) {
				m.EXPECT().
					Detach(gomock.Any(), "nfs-abc", "vpc-123", "").
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "Missing ShareID",
			args: map[string]any{
				"VpcID": "vpc-123",
			},
			expectError: true,
		},
		{
			name: "Missing VpcID",
			args: map[string]any{
				"ShareID": "nfs-abc",
			},
			expectError: true,
		},
		{
			name: "Empty VpcID",
			args: map[string]any{
				"ShareID": "nfs-abc",
				"VpcID":   "",
			},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{
				"ShareID": "nfs-abc",
				"VpcID":   "vpc-123",
			},
			mockSetup: func(m *MockNfsActionsService) {
				m.EXPECT().
					Detach(gomock.Any(), "nfs-abc", "vpc-123", "").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockNfsActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupNfsActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.detachFileShare(context.Background(), req)

			if tc.expectError {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out godo.NfsAction
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, testAction.ID, out.ID)
			require.Equal(t, testAction.Type, out.Type)
			require.Equal(t, testAction.ResourceID, out.ResourceID)
		})
	}
}

func TestNfsActionsTool_reassignFileShare(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.NfsAction{ID: "5", Status: "in-progress", Type: "reassign", ResourceID: "nfs-abc"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockNfsActionsService)
		expectError bool
	}{
		{
			name: "Successful reassign",
			args: map[string]any{
				"ShareID":  "nfs-abc",
				"OldVpcID": "vpc-old",
				"NewVpcID": "vpc-new",
			},
			mockSetup: func(m *MockNfsActionsService) {
				m.EXPECT().
					Reassign(gomock.Any(), "nfs-abc", "vpc-old", "vpc-new").
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "Missing ShareID",
			args: map[string]any{
				"OldVpcID": "vpc-old",
				"NewVpcID": "vpc-new",
			},
			expectError: true,
		},
		{
			name: "Missing OldVpcID",
			args: map[string]any{
				"ShareID":  "nfs-abc",
				"NewVpcID": "vpc-new",
			},
			expectError: true,
		},
		{
			name: "Missing NewVpcID",
			args: map[string]any{
				"ShareID":  "nfs-abc",
				"OldVpcID": "vpc-old",
			},
			expectError: true,
		},
		{
			name: "Empty NewVpcID",
			args: map[string]any{
				"ShareID":  "nfs-abc",
				"OldVpcID": "vpc-old",
				"NewVpcID": "",
			},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{
				"ShareID":  "nfs-abc",
				"OldVpcID": "vpc-old",
				"NewVpcID": "vpc-new",
			},
			mockSetup: func(m *MockNfsActionsService) {
				m.EXPECT().
					Reassign(gomock.Any(), "nfs-abc", "vpc-old", "vpc-new").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockNfsActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupNfsActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.reassignFileShare(context.Background(), req)

			if tc.expectError {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out godo.NfsAction
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, testAction.ID, out.ID)
			require.Equal(t, testAction.Type, out.Type)
			require.Equal(t, testAction.ResourceID, out.ResourceID)
		})
	}
}

func TestNfsActionsTool_switchPerformanceTier(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testAction := &godo.NfsAction{ID: "6", Status: "in-progress", Type: "switch_performance_tier", ResourceID: "nfs-abc"}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockNfsActionsService)
		expectError bool
	}{
		{
			name: "Successful switch",
			args: map[string]any{
				"ShareID":         "nfs-abc",
				"PerformanceTier": "premium",
			},
			mockSetup: func(m *MockNfsActionsService) {
				m.EXPECT().
					SwitchPerformanceTier(gomock.Any(), "nfs-abc", "premium").
					Return(testAction, nil, nil).
					Times(1)
			},
		},
		{
			name: "Missing ShareID",
			args: map[string]any{
				"PerformanceTier": "premium",
			},
			expectError: true,
		},
		{
			name: "Missing PerformanceTier",
			args: map[string]any{
				"ShareID": "nfs-abc",
			},
			expectError: true,
		},
		{
			name: "Empty PerformanceTier",
			args: map[string]any{
				"ShareID":         "nfs-abc",
				"PerformanceTier": "",
			},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{
				"ShareID":         "nfs-abc",
				"PerformanceTier": "premium",
			},
			mockSetup: func(m *MockNfsActionsService) {
				m.EXPECT().
					SwitchPerformanceTier(gomock.Any(), "nfs-abc", "premium").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockActions := NewMockNfsActionsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockActions)
			}
			tool := setupNfsActionsToolWithMocks(mockActions)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.switchPerformanceTier(context.Background(), req)

			if tc.expectError {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out godo.NfsAction
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, testAction.ID, out.ID)
			require.Equal(t, testAction.Type, out.Type)
			require.Equal(t, testAction.ResourceID, out.ResourceID)
		})
	}
}
