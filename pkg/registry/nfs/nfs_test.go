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

func setupNfsToolWithMocks(nfsSvc *MockNfsService) *NfsTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{Nfs: nfsSvc}, nil
	}
	return NewNfsTool(client)
}

func TestNfsTool_createFileShare(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testShare := &godo.Nfs{
		ID:              "nfs-abc",
		Name:            "my-share",
		SizeGib:         50,
		Region:          "nyc3",
		PerformanceTier: "standard",
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockNfsService)
		expectError bool
	}{
		{
			name: "Successful create",
			args: map[string]any{
				"Name":          "my-share",
				"SizeGibibytes": float64(50),
				"Region":        "nyc3",
			},
			mockSetup: func(m *MockNfsService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.NfsCreateRequest{
						Name:    "my-share",
						SizeGib: 50,
						Region:  "nyc3",
					}).
					Return(testShare, nil, nil).
					Times(1)
			},
		},
		{
			name: "Successful create with optional fields",
			args: map[string]any{
				"Name":            "my-share",
				"SizeGibibytes":   float64(100),
				"Region":          "nyc3",
				"PerformanceTier": "standard",
				"VpcIds":          []any{"vpc-1", "vpc-2"},
			},
			mockSetup: func(m *MockNfsService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.NfsCreateRequest{
						Name:            "my-share",
						SizeGib:         100,
						Region:          "nyc3",
						PerformanceTier: "standard",
						VpcIDs:          []string{"vpc-1", "vpc-2"},
					}).
					Return(testShare, nil, nil).
					Times(1)
			},
		},
		{
			name: "Missing Name",
			args: map[string]any{
				"SizeGibibytes": float64(50),
				"Region":        "nyc3",
			},
			expectError: true,
		},
		{
			name: "Missing SizeGibibytes",
			args: map[string]any{
				"Name":   "my-share",
				"Region": "nyc3",
			},
			expectError: true,
		},
		{
			name: "SizeGibibytes below minimum",
			args: map[string]any{
				"Name":          "my-share",
				"SizeGibibytes": float64(10),
				"Region":        "nyc3",
			},
			expectError: true,
		},
		{
			name: "Missing Region",
			args: map[string]any{
				"Name":          "my-share",
				"SizeGibibytes": float64(50),
			},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{
				"Name":          "my-share",
				"SizeGibibytes": float64(50),
				"Region":        "nyc3",
			},
			mockSetup: func(m *MockNfsService) {
				m.EXPECT().
					Create(gomock.Any(), &godo.NfsCreateRequest{
						Name:    "my-share",
						SizeGib: 50,
						Region:  "nyc3",
					}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockNfs := NewMockNfsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockNfs)
			}
			tool := setupNfsToolWithMocks(mockNfs)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createFileShare(context.Background(), req)

			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out godo.Nfs
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, testShare.ID, out.ID)
			require.Equal(t, testShare.Name, out.Name)
		})
	}
}

func TestNfsTool_listFileShares(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	shares := []*godo.Nfs{
		{ID: "nfs-1", Name: "share-a", Region: "nyc1"},
		{ID: "nfs-2", Name: "share-b", Region: "nyc1"},
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockNfsService)
		expectError bool
		wantIDs     []string
	}{
		{
			name: "Successful list with defaults",
			args: map[string]any{
				"Region": "nyc1",
			},
			mockSetup: func(m *MockNfsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 20}, "nyc1").
					Return(shares, nil, nil).
					Times(1)
			},
			wantIDs: []string{"nfs-1", "nfs-2"},
		},
		{
			name: "Successful list without region filter",
			args: map[string]any{},
			mockSetup: func(m *MockNfsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 20}, "").
					Return(shares, nil, nil).
					Times(1)
			},
			wantIDs: []string{"nfs-1", "nfs-2"},
		},
		{
			name: "Successful list with pagination",
			args: map[string]any{
				"Region":  "nyc1",
				"Page":    float64(2),
				"PerPage": float64(5),
			},
			mockSetup: func(m *MockNfsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 2, PerPage: 5}, "nyc1").
					Return(shares, nil, nil).
					Times(1)
			},
			wantIDs: []string{"nfs-1", "nfs-2"},
		},
		{
			name: "PerPage capped at max",
			args: map[string]any{
				"Region":  "nyc1",
				"PerPage": float64(999),
			},
			mockSetup: func(m *MockNfsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 50}, "nyc1").
					Return([]*godo.Nfs{}, nil, nil).
					Times(1)
			},
			wantIDs: nil,
		},
		{
			name: "API error",
			args: map[string]any{
				"Region": "nyc1",
			},
			mockSetup: func(m *MockNfsService) {
				m.EXPECT().
					List(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 20}, "nyc1").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockNfs := NewMockNfsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockNfs)
			}
			tool := setupNfsToolWithMocks(mockNfs)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.listFileShares(context.Background(), req)

			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out []map[string]any
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			if len(tc.wantIDs) == 0 {
				require.Empty(t, out)
				return
			}
			require.Len(t, out, len(tc.wantIDs))
			for i, id := range tc.wantIDs {
				require.Equal(t, id, out[i]["id"])
			}
		})
	}
}

func TestNfsTool_getFileShareByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testShare := &godo.Nfs{
		ID:      "nfs-abc",
		Name:    "my-share",
		SizeGib: 50,
		Region:  "nyc3",
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockNfsService)
		expectError bool
	}{
		{
			name: "Successful get",
			args: map[string]any{
				"ID": "nfs-abc",
			},
			mockSetup: func(m *MockNfsService) {
				m.EXPECT().
					Get(gomock.Any(), "nfs-abc", "").
					Return(testShare, nil, nil).
					Times(1)
			},
		},
		{
			name:        "Missing ID",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name:        "Empty ID",
			args:        map[string]any{"ID": ""},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{"ID": "nfs-abc"},
			mockSetup: func(m *MockNfsService) {
				m.EXPECT().
					Get(gomock.Any(), "nfs-abc", "").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockNfs := NewMockNfsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockNfs)
			}
			tool := setupNfsToolWithMocks(mockNfs)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getFileShareByID(context.Background(), req)

			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out godo.Nfs
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, testShare.ID, out.ID)
			require.Equal(t, testShare.Name, out.Name)
			require.Equal(t, testShare.SizeGib, out.SizeGib)
		})
	}
}

func TestNfsTool_deleteFileShare(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockNfsService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful delete",
			args: map[string]any{
				"ID": "nfs-abc",
			},
			mockSetup: func(m *MockNfsService) {
				m.EXPECT().
					Delete(gomock.Any(), "nfs-abc", "").
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "File share deleted successfully",
		},
		{
			name:        "Missing ID",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name:        "Empty ID",
			args:        map[string]any{"ID": ""},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{"ID": "nfs-abc"},
			mockSetup: func(m *MockNfsService) {
				m.EXPECT().
					Delete(gomock.Any(), "nfs-abc", "").
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockNfs := NewMockNfsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockNfs)
			}
			tool := setupNfsToolWithMocks(mockNfs)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteFileShare(context.Background(), req)

			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.Contains(t, resp.Content[0].(mcp.TextContent).Text, tc.expectText)
		})
	}
}

func TestNfsTool_listNfsSnapshots(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	snapshots := []*godo.NfsSnapshot{
		{ID: "snap-1", Name: "snapshot-a", ShareID: "nfs-abc", Region: "nyc3"},
		{ID: "snap-2", Name: "snapshot-b", ShareID: "nfs-abc", Region: "nyc3"},
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockNfsService)
		expectError bool
		wantIDs     []string
	}{
		{
			name: "Successful list with defaults",
			args: map[string]any{
				"Region": "nyc3",
			},
			mockSetup: func(m *MockNfsService) {
				m.EXPECT().
					ListSnapshots(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 20}, "", "nyc3").
					Return(snapshots, nil, nil).
					Times(1)
			},
			wantIDs: []string{"snap-1", "snap-2"},
		},
		{
			name: "Successful list filtered by share ID",
			args: map[string]any{
				"Region":  "nyc3",
				"shareID": "nfs-abc",
			},
			mockSetup: func(m *MockNfsService) {
				m.EXPECT().
					ListSnapshots(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 20}, "nfs-abc", "nyc3").
					Return(snapshots, nil, nil).
					Times(1)
			},
			wantIDs: []string{"snap-1", "snap-2"},
		},
		{
			name: "Successful list with pagination",
			args: map[string]any{
				"Region":  "nyc3",
				"Page":    float64(2),
				"PerPage": float64(10),
			},
			mockSetup: func(m *MockNfsService) {
				m.EXPECT().
					ListSnapshots(gomock.Any(), &godo.ListOptions{Page: 2, PerPage: 10}, "", "nyc3").
					Return([]*godo.NfsSnapshot{}, nil, nil).
					Times(1)
			},
			wantIDs: nil,
		},
		{
			name: "PerPage capped at max",
			args: map[string]any{
				"Region":  "nyc3",
				"PerPage": float64(999),
			},
			mockSetup: func(m *MockNfsService) {
				m.EXPECT().
					ListSnapshots(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 50}, "", "nyc3").
					Return([]*godo.NfsSnapshot{}, nil, nil).
					Times(1)
			},
			wantIDs: nil,
		},
		{
			name: "API error",
			args: map[string]any{
				"Region": "nyc3",
			},
			mockSetup: func(m *MockNfsService) {
				m.EXPECT().
					ListSnapshots(gomock.Any(), &godo.ListOptions{Page: 1, PerPage: 20}, "", "nyc3").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockNfs := NewMockNfsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockNfs)
			}
			tool := setupNfsToolWithMocks(mockNfs)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.listNfsSnapshots(context.Background(), req)

			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out []map[string]any
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			if len(tc.wantIDs) == 0 {
				require.Empty(t, out)
				return
			}
			require.Len(t, out, len(tc.wantIDs))
			for i, id := range tc.wantIDs {
				require.Equal(t, id, out[i]["id"])
			}
		})
	}
}

func TestNfsTool_getNfsSnapshotByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testSnapshot := &godo.NfsSnapshot{
		ID:      "snap-abc",
		Name:    "my-snapshot",
		ShareID: "nfs-abc",
		SizeGib: 50,
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockNfsService)
		expectError bool
	}{
		{
			name: "Successful get",
			args: map[string]any{
				"ID": "snap-abc",
			},
			mockSetup: func(m *MockNfsService) {
				m.EXPECT().
					GetSnapshot(gomock.Any(), "snap-abc", "").
					Return(testSnapshot, nil, nil).
					Times(1)
			},
		},
		{
			name:        "Missing ID",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name:        "Empty ID",
			args:        map[string]any{"ID": ""},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{"ID": "snap-abc"},
			mockSetup: func(m *MockNfsService) {
				m.EXPECT().
					GetSnapshot(gomock.Any(), "snap-abc", "").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockNfs := NewMockNfsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockNfs)
			}
			tool := setupNfsToolWithMocks(mockNfs)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getNfsSnapshotByID(context.Background(), req)

			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out godo.NfsSnapshot
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, testSnapshot.ID, out.ID)
			require.Equal(t, testSnapshot.Name, out.Name)
			require.Equal(t, testSnapshot.ShareID, out.ShareID)
		})
	}
}

func TestNfsTool_deleteNfsSnapshot(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockNfsService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful delete",
			args: map[string]any{
				"ID": "snap-abc",
			},
			mockSetup: func(m *MockNfsService) {
				m.EXPECT().
					DeleteSnapshot(gomock.Any(), "snap-abc", "").
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectText: "NFS snapshot deleted successfully",
		},
		{
			name:        "Missing ID",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name:        "Empty ID",
			args:        map[string]any{"ID": ""},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{"ID": "snap-abc"},
			mockSetup: func(m *MockNfsService) {
				m.EXPECT().
					DeleteSnapshot(gomock.Any(), "snap-abc", "").
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockNfs := NewMockNfsService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockNfs)
			}
			tool := setupNfsToolWithMocks(mockNfs)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteNfsSnapshot(context.Background(), req)

			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			require.Contains(t, resp.Content[0].(mcp.TextContent).Text, tc.expectText)
		})
	}
}
