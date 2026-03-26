package blockstorage

import (
	"context"
	// "encoding/json"
	// "errors"
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
			args: map[string]any{ //these are the arguments that will be passed to the volume tool, WHICH has a mocked storage service
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
