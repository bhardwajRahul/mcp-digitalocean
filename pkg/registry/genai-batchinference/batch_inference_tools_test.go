package genaibatchinference

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupToolWithMock(mock godo.BatchInferenceService) *BatchInferenceTool {
	return NewBatchInferenceTool(func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{BatchInference: mock}, nil
	})
}

func setupToolWithClientError() *BatchInferenceTool {
	return NewBatchInferenceTool(func(ctx context.Context) (*godo.Client, error) {
		return nil, errors.New("auth failed")
	})
}

var (
	testFileResponse = &godo.CreateBatchFileResponse{
		FileID:    "b7d562e0-49ae-43f7-820d-6a37a2f05435",
		UploadURL: "https://spaces.example.com/presigned-upload",
		ExpiresAt: "2026-04-21T17:49:27.536939587Z",
	}

	expiresAt       = "2026-04-22T17:41:57Z"
	testBatchQueued = &godo.Batch{
		BatchID:          "11f13da9-64a4-fe5b-8567-a23ae3abd3e2",
		Provider:         "anthropic",
		FileID:           "b7d562e0-49ae-43f7-820d-6a37a2f05435",
		CompletionWindow: "24h",
		Status:           "queued",
		RequestID:        "postman-1776793316",
		RequestCounts:    &godo.BatchRequestCounts{Total: 0, Completed: 0, Failed: 0},
		ResultAvailable:  false,
		CreatedAt:        "2026-04-21T17:41:57Z",
		UpdatedAt:        "2026-04-21T17:41:57Z",
		ExpiresAt:        &expiresAt,
	}

	testBatchCompleted = &godo.Batch{
		BatchID:          "11f13da9-64a4-fe5b-8567-a23ae3abd3e2",
		Provider:         "anthropic",
		FileID:           "b7d562e0-49ae-43f7-820d-6a37a2f05435",
		CompletionWindow: "24h",
		Status:           "completed",
		RequestID:        "postman-1776793316",
		RequestCounts:    &godo.BatchRequestCounts{Total: 2, Completed: 2, Failed: 0},
		ResultAvailable:  true,
		CreatedAt:        "2026-04-21T17:41:57Z",
		UpdatedAt:        "2026-04-21T17:44:26Z",
	}

	cancelledAt        = "2026-04-21T18:10:00Z"
	testBatchCancelled = &godo.Batch{
		BatchID:           "11f13dad-0b91-104b-8567-a23ae3abd3e2",
		Provider:          "anthropic",
		FileID:            "b7d562e0-49ae-43f7-820d-6a37a2f05435",
		CompletionWindow:  "24h",
		Status:            "cancelled",
		RequestID:         "postman-cancel-test",
		RequestCounts:     &godo.BatchRequestCounts{Total: 2, Completed: 0, Failed: 0},
		ResultAvailable:   false,
		CancelRequestedAt: &cancelledAt,
		CreatedAt:         "2026-04-21T18:05:00Z",
		UpdatedAt:         "2026-04-21T18:10:00Z",
	}

	testResults = &godo.BatchResultsResponse{
		OutputFileID: "msgbatch_015UytMT8cCLD8332Hb3BhNe",
		Download: godo.BatchResultsDownload{
			PresignedURL: "https://spaces.example.com/presigned-download",
			ExpiresAt:    "2026-04-21T18:07:47.228188268Z",
		},
	}

	testListResponse = &godo.ListBatchesResponse{
		Edges: []godo.BatchEdge{
			{
				Cursor: "eyJjIjoiMjAyNi0wNC0yMVQxNzo0MTo1N1oiLCJpIjoyMH0=",
				Node: godo.Batch{
					BatchID:         "11f13da9-64a4-fe5b-8567-a23ae3abd3e2",
					Status:          "completed",
					Provider:        "anthropic",
					ResultAvailable: true,
					RequestCounts:   &godo.BatchRequestCounts{Total: 2, Completed: 2, Failed: 0},
				},
			},
		},
		PageInfo: godo.BatchPageInfo{
			HasNextPage: true,
			EndCursor:   "eyJjIjoiMjAyNi0wNC0yMVQxNzo0MTo1N1oiLCJpIjoyMH0=",
		},
	}
)

func TestBatchInferenceTool_createFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockBatchInferenceService)
		expectError bool
	}{
		{
			name:        "missing FileName",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name:        "empty FileName",
			args:        map[string]any{"FileName": ""},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"FileName": "input.jsonl"},
			mockSetup: func(m *MockBatchInferenceService) {
				m.EXPECT().CreatePresignedUploadURL(gomock.Any(), gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"FileName": "input.jsonl"},
			mockSetup: func(m *MockBatchInferenceService) {
				m.EXPECT().CreatePresignedUploadURL(gomock.Any(), &godo.CreateBatchFileRequest{
					FileName: "input.jsonl",
				}).Return(testFileResponse, &godo.Response{}, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockBatchInferenceService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createFile(context.Background(), req)

			if tc.expectError {
				if err != nil {
					return
				}
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			tc2, ok := resp.Content[0].(mcp.TextContent)
			require.True(t, ok)

			var result godo.CreateBatchFileResponse
			err = json.Unmarshal([]byte(tc2.Text), &result)
			require.NoError(t, err)
			require.Equal(t, testFileResponse.FileID, result.FileID)
			require.Equal(t, testFileResponse.UploadURL, result.UploadURL)
			require.Equal(t, testFileResponse.ExpiresAt, result.ExpiresAt)
		})
	}
}

func TestBatchInferenceTool_uploadInputFile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockBatchInferenceService)
		expectError bool
	}{
		{
			name:        "missing UploadURL",
			args:        map[string]any{"Content": "test"},
			expectError: true,
		},
		{
			name:        "missing Content",
			args:        map[string]any{"UploadURL": "https://example.com/upload"},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"UploadURL": "https://example.com/upload", "Content": "test"},
			mockSetup: func(m *MockBatchInferenceService) {
				m.EXPECT().UploadInputFile(gomock.Any(), "https://example.com/upload", gomock.Any()).Return(nil, errors.New("upload failed"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{
				"UploadURL": "https://example.com/upload",
				"Content":   `{"custom_id":"req-1","method":"POST","url":"/v1/messages","body":{}}`,
			},
			mockSetup: func(m *MockBatchInferenceService) {
				m.EXPECT().UploadInputFile(gomock.Any(), "https://example.com/upload", gomock.Any()).
					DoAndReturn(func(_ context.Context, url string, content io.Reader) (*godo.Response, error) {
						data, err := io.ReadAll(content)
						require.NoError(t, err)
						require.Contains(t, string(data), "custom_id")
						return &godo.Response{}, nil
					})
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockBatchInferenceService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.uploadInputFile(context.Background(), req)

			if tc.expectError {
				if err != nil {
					return
				}
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

func TestBatchInferenceTool_createJob(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockBatchInferenceService)
		expectError bool
	}{
		{
			name:        "missing Provider",
			args:        map[string]any{"FileID": "file-123", "CompletionWindow": "24h"},
			expectError: true,
		},
		{
			name:        "missing FileID",
			args:        map[string]any{"Provider": "openai", "CompletionWindow": "24h"},
			expectError: true,
		},
		{
			name:        "missing CompletionWindow",
			args:        map[string]any{"Provider": "openai", "FileID": "file-123"},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{
				"Provider":         "openai",
				"FileID":           "file-123",
				"CompletionWindow": "24h",
				"Endpoint":         "/v1/chat/completions",
			},
			mockSetup: func(m *MockBatchInferenceService) {
				m.EXPECT().CreateJob(gomock.Any(), gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success openai",
			args: map[string]any{
				"Provider":         "openai",
				"FileID":           "b7d562e0-49ae-43f7-820d-6a37a2f05435",
				"CompletionWindow": "24h",
				"RequestID":        "postman-1776793316",
				"Endpoint":         "/v1/chat/completions",
			},
			mockSetup: func(m *MockBatchInferenceService) {
				m.EXPECT().CreateJob(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, req *godo.CreateBatchRequest) (*godo.Batch, *godo.Response, error) {
						require.Equal(t, "openai", req.Provider)
						require.Equal(t, "24h", req.CompletionWindow)
						require.Equal(t, "postman-1776793316", req.RequestID)
						require.Equal(t, "b7d562e0-49ae-43f7-820d-6a37a2f05435", req.FileID)
						require.Equal(t, "/v1/chat/completions", req.Endpoint)
						return testBatchQueued, &godo.Response{}, nil
					})
			},
		},
		{
			name: "success anthropic",
			args: map[string]any{
				"Provider":         "anthropic",
				"FileID":           "b7d562e0-49ae-43f7-820d-6a37a2f05435",
				"CompletionWindow": "24h",
			},
			mockSetup: func(m *MockBatchInferenceService) {
				m.EXPECT().CreateJob(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, req *godo.CreateBatchRequest) (*godo.Batch, *godo.Response, error) {
						require.Equal(t, "anthropic", req.Provider)
						require.Empty(t, req.Endpoint)
						return testBatchQueued, &godo.Response{}, nil
					})
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockBatchInferenceService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createJob(context.Background(), req)

			if tc.expectError {
				if err != nil {
					return
				}
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

func TestBatchInferenceTool_getJob(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockBatchInferenceService)
		expectError bool
	}{
		{
			name:        "missing BatchID",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"BatchID": "batch-uuid-1"},
			mockSetup: func(m *MockBatchInferenceService) {
				m.EXPECT().GetJob(gomock.Any(), "batch-uuid-1").Return(nil, nil, errors.New("not found"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"BatchID": "11f13da9-64a4-fe5b-8567-a23ae3abd3e2"},
			mockSetup: func(m *MockBatchInferenceService) {
				m.EXPECT().GetJob(gomock.Any(), "11f13da9-64a4-fe5b-8567-a23ae3abd3e2").Return(testBatchCompleted, &godo.Response{}, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockBatchInferenceService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getJob(context.Background(), req)

			if tc.expectError {
				if err != nil {
					return
				}
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			tc2, ok := resp.Content[0].(mcp.TextContent)
			require.True(t, ok)

			var result godo.Batch
			err = json.Unmarshal([]byte(tc2.Text), &result)
			require.NoError(t, err)
			require.Equal(t, testBatchCompleted.BatchID, result.BatchID)
			require.Equal(t, testBatchCompleted.Status, result.Status)
		})
	}
}

func TestBatchInferenceTool_getJobResults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockBatchInferenceService)
		expectError bool
	}{
		{
			name:        "missing BatchID",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "api error (not ready)",
			args: map[string]any{"BatchID": "batch-uuid-1"},
			mockSetup: func(m *MockBatchInferenceService) {
				m.EXPECT().GetJobResult(gomock.Any(), "batch-uuid-1").Return(nil, nil, errors.New("FAILED_PRECONDITION: results not ready"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"BatchID": "11f13da9-64a4-fe5b-8567-a23ae3abd3e2"},
			mockSetup: func(m *MockBatchInferenceService) {
				m.EXPECT().GetJobResult(gomock.Any(), "11f13da9-64a4-fe5b-8567-a23ae3abd3e2").Return(testResults, &godo.Response{}, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockBatchInferenceService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getJobResults(context.Background(), req)

			if tc.expectError {
				if err != nil {
					return
				}
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			tc2, ok := resp.Content[0].(mcp.TextContent)
			require.True(t, ok)

			var result godo.BatchResultsResponse
			err = json.Unmarshal([]byte(tc2.Text), &result)
			require.NoError(t, err)
			require.Equal(t, testResults.OutputFileID, result.OutputFileID)
			require.Equal(t, testResults.Download.PresignedURL, result.Download.PresignedURL)
		})
	}
}

func TestBatchInferenceTool_cancelJob(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockBatchInferenceService)
		expectError bool
	}{
		{
			name:        "missing BatchID",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"BatchID": "batch-uuid-1"},
			mockSetup: func(m *MockBatchInferenceService) {
				m.EXPECT().CancelJob(gomock.Any(), "batch-uuid-1").Return(nil, nil, errors.New("not cancellable"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"BatchID": "11f13dad-0b91-104b-8567-a23ae3abd3e2"},
			mockSetup: func(m *MockBatchInferenceService) {
				m.EXPECT().CancelJob(gomock.Any(), "11f13dad-0b91-104b-8567-a23ae3abd3e2").Return(testBatchCancelled, &godo.Response{}, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockBatchInferenceService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.cancelJob(context.Background(), req)

			if tc.expectError {
				if err != nil {
					return
				}
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			tc2, ok := resp.Content[0].(mcp.TextContent)
			require.True(t, ok)

			var result godo.Batch
			err = json.Unmarshal([]byte(tc2.Text), &result)
			require.NoError(t, err)
			require.Equal(t, testBatchCancelled.BatchID, result.BatchID)
			require.Equal(t, "cancelled", result.Status)
			require.NotNil(t, result.CancelRequestedAt)
		})
	}
}

func TestBatchInferenceTool_listJobs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockBatchInferenceService)
		expectError bool
	}{
		{
			name: "api error",
			args: map[string]any{},
			mockSetup: func(m *MockBatchInferenceService) {
				m.EXPECT().ListJobs(gomock.Any(), gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success with filters",
			args: map[string]any{"Status": "completed", "Limit": float64(10), "After": "cursor-0"},
			mockSetup: func(m *MockBatchInferenceService) {
				m.EXPECT().ListJobs(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, opts *godo.ListBatchesOptions) (*godo.ListBatchesResponse, *godo.Response, error) {
						require.Equal(t, "completed", opts.Status)
						require.Equal(t, 10, opts.Limit)
						require.Equal(t, "cursor-0", opts.After)
						return testListResponse, &godo.Response{}, nil
					})
			},
		},
		{
			name: "success no filters",
			args: map[string]any{},
			mockSetup: func(m *MockBatchInferenceService) {
				m.EXPECT().ListJobs(gomock.Any(), gomock.Any()).Return(testListResponse, &godo.Response{}, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockBatchInferenceService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.listJobs(context.Background(), req)

			if tc.expectError {
				if err != nil {
					return
				}
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)

			tc2, ok := resp.Content[0].(mcp.TextContent)
			require.True(t, ok)

			var result godo.ListBatchesResponse
			err = json.Unmarshal([]byte(tc2.Text), &result)
			require.NoError(t, err)
			require.Len(t, result.Edges, 1)
			require.True(t, result.PageInfo.HasNextPage)
		})
	}
}

func TestBatchInferenceTool_Tools(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockBatchInferenceService(ctrl)
	tool := setupToolWithMock(mock)

	tools := tool.Tools()
	require.Len(t, tools, 7)

	toolNames := make(map[string]bool)
	for _, st := range tools {
		toolNames[st.Tool.Name] = true
	}

	require.True(t, toolNames["genai-batch-inference-create-file"])
	require.True(t, toolNames["genai-batch-inference-upload-file"])
	require.True(t, toolNames["genai-batch-inference-create"])
	require.True(t, toolNames["genai-batch-inference-get"])
	require.True(t, toolNames["genai-batch-inference-get-results"])
	require.True(t, toolNames["genai-batch-inference-cancel"])
	require.True(t, toolNames["genai-batch-inference-list"])
}

func TestBatchInferenceTool_clientError(t *testing.T) {
	tool := setupToolWithClientError()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"BatchID":          "batch-uuid-1",
		"FileName":         "input.jsonl",
		"Provider":         "openai",
		"CompletionWindow": "24h",
		"FileID":           "file-abc-123",
		"Endpoint":         "/v1/chat/completions",
	}}}

	handlers := []struct {
		name string
		fn   func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
	}{
		{"createFile", tool.createFile},
		{"uploadInputFile", tool.uploadInputFile},
		{"createJob", tool.createJob},
		{"getJob", tool.getJob},
		{"getJobResults", tool.getJobResults},
		{"cancelJob", tool.cancelJob},
		{"listJobs", tool.listJobs},
	}

	for _, h := range handlers {
		t.Run(h.name, func(t *testing.T) {
			_, err := h.fn(context.Background(), req)
			require.Error(t, err)
			require.Contains(t, err.Error(), "auth failed")
		})
	}
}
