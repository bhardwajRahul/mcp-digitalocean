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

func setupVolumeToolWithMocks(storage *MockStorageService) *VolumeTool {
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Storage: storage,
		}, nil
	}
	return NewVolumeTool(client)
}

func TestVolumeTool_createVolume(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testVolume := &godo.Volume{
		Name:          "test-volume",
		SizeGigaBytes: int64(10),
		Region:        &godo.Region{Slug: "nyc1", Name: "New York 1"},
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockStorageService)
		expectError bool
	}{
		{
			name: "Successful create",
			args: map[string]any{
				"Name":          "test-volume",
				"SizeGigaBytes": float64(10),
				"Region":        "nyc1",
			},
			mockSetup: func(m *MockStorageService) {
				m.EXPECT().
					CreateVolume(gomock.Any(), &godo.VolumeCreateRequest{
						Name:          "test-volume",
						SizeGigaBytes: int64(10),
						Region:        "nyc1",
					}).
					Return(testVolume, nil, nil).
					Times(1)
			},
			expectError: false,
		},
		{
			name: "Missing Name",
			args: map[string]any{
				"SizeGigaBytes": float64(10),
				"Region":        "nyc1",
			},
			expectError: true,
		},
		{
			name: "Missing SizeGigaBytes",
			args: map[string]any{
				"Name":   "test-volume",
				"Region": "nyc1",
			},
			expectError: true,
		},
		{
			name: "Missing Region",
			args: map[string]any{
				"Name":          "test-volume",
				"SizeGigaBytes": float64(10),
			},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{
				"Name":          "test-volume",
				"SizeGigaBytes": float64(10),
				"Region":        "nyc1",
			},
			mockSetup: func(m *MockStorageService) {
				m.EXPECT().
					CreateVolume(gomock.Any(), &godo.VolumeCreateRequest{
						Name:          "test-volume",
						SizeGigaBytes: int64(10),
						Region:        "nyc1",
					}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockVolumes := NewMockStorageService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockVolumes)
			}
			tool := setupVolumeToolWithMocks(mockVolumes)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createVolume(context.Background(), req)

			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
		})
	}

}

func TestVolumeTool_deleteVolume(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockStorageService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful delete",
			args: map[string]any{
				"ID": "123",
			},
			mockSetup: func(m *MockStorageService) {
				m.EXPECT().
					DeleteVolume(gomock.Any(), "123").
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectError: false,
			expectText:  "Volume deleted successfully",
		},
		{
			name:        "Missing ID",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{
				"ID": "123",
			},
			mockSetup: func(m *MockStorageService) {
				m.EXPECT().
					DeleteVolume(gomock.Any(), "123").
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockVolumes := NewMockStorageService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockVolumes)
			}
			tool := setupVolumeToolWithMocks(mockVolumes)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteVolume(context.Background(), req)

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

func TestVolumeTool_getVolumeByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testVolume := &godo.Volume{
		ID:            "vol-123",
		Name:          "test-volume",
		SizeGigaBytes: 10,
		Region:        &godo.Region{Slug: "nyc1"},
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockStorageService)
		expectError bool
	}{
		{
			name: "Successful get",
			args: map[string]any{"ID": "vol-123"},
			mockSetup: func(m *MockStorageService) {
				m.EXPECT().
					GetVolume(gomock.Any(), "vol-123").
					Return(testVolume, nil, nil).
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
			args: map[string]any{"ID": "vol-123"},
			mockSetup: func(m *MockStorageService) {
				m.EXPECT().
					GetVolume(gomock.Any(), "vol-123").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockStorage := NewMockStorageService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockStorage)
			}
			tool := setupVolumeToolWithMocks(mockStorage)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getVolumeByID(context.Background(), req)

			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out godo.Volume
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, testVolume.ID, out.ID)
			require.Equal(t, testVolume.Name, out.Name)
			require.Equal(t, testVolume.SizeGigaBytes, out.SizeGigaBytes)
			require.NotNil(t, out.Region)
			require.Equal(t, testVolume.Region.Slug, out.Region.Slug)
		})
	}
}

func TestVolumeTool_listVolumes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	vols := []godo.Volume{
		{ID: "vol-1", Name: "a", SizeGigaBytes: 10},
		{ID: "vol-2", Name: "b", SizeGigaBytes: 20},
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockStorageService)
		expectError bool
		wantIDs     []string
	}{
		{
			name: "Successful list with defaults",
			args: map[string]any{},
			mockSetup: func(m *MockStorageService) {
				m.EXPECT().
					ListVolumes(gomock.Any(), &godo.ListVolumeParams{
						ListOptions: &godo.ListOptions{Page: 1, PerPage: 50},
					}).
					Return(vols, nil, nil).
					Times(1)
			},
			wantIDs: []string{"vol-1", "vol-2"},
		},
		{
			name: "Successful list with filters and pagination",
			args: map[string]any{
				"Name":    "myvol",
				"Region":  "nyc1",
				"Page":    float64(2),
				"PerPage": float64(100),
			},
			mockSetup: func(m *MockStorageService) {
				m.EXPECT().
					ListVolumes(gomock.Any(), &godo.ListVolumeParams{
						Name:   "myvol",
						Region: "nyc1",
						ListOptions: &godo.ListOptions{
							Page:    2,
							PerPage: 100,
						},
					}).
					Return(vols, nil, nil).
					Times(1)
			},
			wantIDs: []string{"vol-1", "vol-2"},
		},
		{
			name: "PerPage capped at max",
			args: map[string]any{
				"PerPage": float64(500),
			},
			mockSetup: func(m *MockStorageService) {
				m.EXPECT().
					ListVolumes(gomock.Any(), &godo.ListVolumeParams{
						ListOptions: &godo.ListOptions{Page: 1, PerPage: 200},
					}).
					Return([]godo.Volume{}, nil, nil).
					Times(1)
			},
			wantIDs: nil,
		},
		{
			name: "API error",
			args: map[string]any{},
			mockSetup: func(m *MockStorageService) {
				m.EXPECT().
					ListVolumes(gomock.Any(), &godo.ListVolumeParams{
						ListOptions: &godo.ListOptions{Page: 1, PerPage: 50},
					}).
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockStorage := NewMockStorageService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockStorage)
			}
			tool := setupVolumeToolWithMocks(mockStorage)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.listVolumes(context.Background(), req)

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

func TestVolumeTool_createSnapshot(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testSnapshot := &godo.Snapshot{
		ID:   "snap-123",
		Name: "my-snapshot",
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockStorageService)
		expectError bool
	}{
		{
			name: "Successful create snapshot",
			args: map[string]any{
				"VolumeID": "vol-123",
				"Name":     "my-snapshot",
				"Tags":     []any{"tag1", "tag2"},
			},
			mockSetup: func(m *MockStorageService) {
				m.EXPECT().
					CreateSnapshot(gomock.Any(), &godo.SnapshotCreateRequest{
						VolumeID: "vol-123",
						Name:     "my-snapshot",
						Tags:     []string{"tag1", "tag2"},
					}).
					Return(testSnapshot, nil, nil).
					Times(1)
			},
			expectError: false,
		},
		{
			name: "Missing VolumeID",
			args: map[string]any{
				"Name": "something",
			},
			expectError: true,
		},
		{
			name: "Missing Name",
			args: map[string]any{
				"VolumeID": "vol-123",
			},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{
				"VolumeID": "vol-123",
				"Name":     "snapfail",
			},
			mockSetup: func(m *MockStorageService) {
				m.EXPECT().
					CreateSnapshot(gomock.Any(), &godo.SnapshotCreateRequest{
						VolumeID: "vol-123",
						Name:     "snapfail",
						Tags:     nil,
					}).
					Return(nil, nil, errors.New("api error")).Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockStorage := NewMockStorageService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockStorage)
			}
			tool := setupVolumeToolWithMocks(mockStorage)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createSnapshot(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var out godo.Snapshot
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, testSnapshot.ID, out.ID)
		})
	}
}

func TestVolumeTool_listSnapshots(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testSnapshots := []godo.Snapshot{
		{ID: "snap-1", Name: "snap1"},
		{ID: "snap-2", Name: "snap2"},
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockStorageService)
		expectIDs   []string
		expectError bool
	}{
		{
			name: "Successful list snapshots",
			args: map[string]any{
				"VolumeID": "vol-123",
			},
			mockSetup: func(m *MockStorageService) {
				m.EXPECT().
					ListSnapshots(gomock.Any(), "vol-123", &godo.ListOptions{Page: 1, PerPage: 50}).
					Return(testSnapshots, nil, nil).
					Times(1)
			},
			expectIDs: []string{"snap-1", "snap-2"},
		},
		{
			name:        "Missing VolumeID",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{
				"VolumeID": "bad-id",
			},
			mockSetup: func(m *MockStorageService) {
				m.EXPECT().
					ListSnapshots(gomock.Any(), "bad-id", &godo.ListOptions{Page: 1, PerPage: 50}).
					Return(nil, nil, errors.New("api error")).Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockStorage := NewMockStorageService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockStorage)
			}
			tool := setupVolumeToolWithMocks(mockStorage)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.listSnapshots(context.Background(), req)
			if tc.expectError {
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			var snaps []map[string]any
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &snaps))
			require.Len(t, snaps, len(tc.expectIDs))
			for i, id := range tc.expectIDs {
				require.Equal(t, id, snaps[i]["id"])
			}
		})
	}
}

func TestVolumeTool_deleteSnapshot(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockStorageService)
		expectError bool
		expectText  string
	}{
		{
			name: "Successful delete",
			args: map[string]any{
				"ID": "snap-123",
			},
			mockSetup: func(m *MockStorageService) {
				m.EXPECT().
					DeleteSnapshot(gomock.Any(), "snap-123").
					Return(&godo.Response{}, nil).
					Times(1)
			},
			expectError: false,
			expectText:  "Snapshot deleted successfully",
		},
		{
			name:        "Missing ID",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "API error",
			args: map[string]any{
				"ID": "snap-123",
			},
			mockSetup: func(m *MockStorageService) {
				m.EXPECT().
					DeleteSnapshot(gomock.Any(), "snap-123").
					Return(nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockStorage := NewMockStorageService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockStorage)
			}
			tool := setupVolumeToolWithMocks(mockStorage)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteSnapshot(context.Background(), req)

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

func TestVolumeTool_getSnapshotByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testSnapshot := &godo.Snapshot{
		ID:   "snap-abc",
		Name: "test-snap",
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockStorageService)
		expectError bool
	}{
		{
			name: "Successful get",
			args: map[string]any{"ID": "snap-abc"},
			mockSetup: func(m *MockStorageService) {
				m.EXPECT().
					GetSnapshot(gomock.Any(), "snap-abc").
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
			mockSetup: func(m *MockStorageService) {
				m.EXPECT().
					GetSnapshot(gomock.Any(), "snap-abc").
					Return(nil, nil, errors.New("api error")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockStorage := NewMockStorageService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mockStorage)
			}
			tool := setupVolumeToolWithMocks(mockStorage)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getSnapshotByID(context.Background(), req)

			if tc.expectError {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			var out godo.Snapshot
			require.NoError(t, json.Unmarshal([]byte(resp.Content[0].(mcp.TextContent).Text), &out))
			require.Equal(t, testSnapshot.ID, out.ID)
			require.Equal(t, testSnapshot.Name, out.Name)
		})
	}
}
