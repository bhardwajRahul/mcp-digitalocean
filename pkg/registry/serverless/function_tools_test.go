package serverless

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	middleware "mcp-digitalocean/internal"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type mockAccessKeySvc struct {
	createFn func(ctx context.Context, namespace string, req *AccessKeyCreateRequest) (*AccessKey, error)
}

func (m *mockAccessKeySvc) CreateAccessKey(ctx context.Context, namespace string, req *AccessKeyCreateRequest) (*AccessKey, error) {
	return m.createFn(ctx, namespace, req)
}

func setupFunctionToolWithMockAndServer(t *testing.T, mockFunctions godo.FunctionsService, handler http.HandlerFunc) *FunctionTool {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	mockFunctions.(*MockFunctionsService).EXPECT().
		GetNamespace(gomock.Any(), gomock.Any()).
		Return(&godo.FunctionsNamespace{
			ApiHost: ts.URL,
			Key:     "test-uuid:test-secret",
			UUID:    "ns-uuid",
		}, nil, nil).
		AnyTimes()

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{
			Functions: mockFunctions,
		}, nil
	}

	tool := NewFunctionTool(context.Background(), client)
	tool.httpClient = ts.Client()
	return tool
}

func TestFunctionTool_list(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		args         map[string]any
		owHandler    http.HandlerFunc
		mockSetup    func(*MockFunctionsService)
		expectError  bool
		skipOWServer bool
	}{
		{
			name:         "missing NamespaceID",
			args:         map[string]any{},
			expectError:  true,
			skipOWServer: true,
		},
		{
			name: "namespace lookup failure",
			args: map[string]any{"NamespaceID": "ns-uuid"},
			mockSetup: func(m *MockFunctionsService) {
				m.EXPECT().GetNamespace(gomock.Any(), "ns-uuid").Return(nil, nil, errors.New("not found"))
			},
			expectError:  true,
			skipOWServer: true,
		},
		{
			name: "openwhisk api error",
			args: map[string]any{"NamespaceID": "ns-uuid"},
			owHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"internal error"}`))
			},
		},
		{
			name: "success",
			args: map[string]any{"NamespaceID": "ns-uuid"},
			owHandler: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodGet, r.Method)
				require.Contains(t, r.URL.Path, "/api/v1/namespaces/_/actions")
				json.NewEncoder(w).Encode([]map[string]string{{"name": "hello"}, {"name": "goodbye"}})
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipOWServer {
				mock := NewMockFunctionsService(ctrl)
				if tc.mockSetup != nil {
					tc.mockSetup(mock)
				}
				client := func(ctx context.Context) (*godo.Client, error) {
					return &godo.Client{Functions: mock}, nil
				}
				tool := NewFunctionTool(context.Background(), client)
				req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
				resp, _ := tool.list(context.Background(), req)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			mock := NewMockFunctionsService(ctrl)
			tool := setupFunctionToolWithMockAndServer(t, mock, tc.owHandler)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.list(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			if tc.name == "openwhisk api error" {
				require.True(t, resp.IsError)
			} else {
				require.False(t, resp.IsError)
				textContent, ok := resp.Content[0].(mcp.TextContent)
				require.True(t, ok)
				require.Contains(t, textContent.Text, "hello")
			}
		})
	}
}

func TestFunctionTool_get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		args         map[string]any
		owHandler    http.HandlerFunc
		expectError  bool
		skipOWServer bool
	}{
		{
			name:         "missing NamespaceID",
			args:         map[string]any{"FunctionName": "hello"},
			expectError:  true,
			skipOWServer: true,
		},
		{
			name:         "missing FunctionName",
			args:         map[string]any{"NamespaceID": "ns-uuid"},
			expectError:  true,
			skipOWServer: true,
		},
		{
			name: "success",
			args: map[string]any{"NamespaceID": "ns-uuid", "FunctionName": "hello"},
			owHandler: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodGet, r.Method)
				require.Contains(t, r.URL.Path, "/actions/hello")
				json.NewEncoder(w).Encode(map[string]string{"name": "hello", "kind": "nodejs:18"})
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipOWServer {
				mock := NewMockFunctionsService(ctrl)
				client := func(ctx context.Context) (*godo.Client, error) {
					return &godo.Client{Functions: mock}, nil
				}
				tool := NewFunctionTool(context.Background(), client)
				req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
				resp, _ := tool.get(context.Background(), req)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			mock := NewMockFunctionsService(ctrl)
			tool := setupFunctionToolWithMockAndServer(t, mock, tc.owHandler)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.get(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			textContent, ok := resp.Content[0].(mcp.TextContent)
			require.True(t, ok)
			require.Contains(t, textContent.Text, "hello")
		})
	}
}

func TestFunctionTool_create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		args         map[string]any
		owHandler    http.HandlerFunc
		expectError  bool
		skipOWServer bool
	}{
		{
			name:         "missing NamespaceID",
			args:         map[string]any{"FunctionName": "hello", "Kind": "nodejs:18", "Code": "function main() {}"},
			expectError:  true,
			skipOWServer: true,
		},
		{
			name:         "missing FunctionName",
			args:         map[string]any{"NamespaceID": "ns-uuid", "Kind": "nodejs:18", "Code": "function main() {}"},
			expectError:  true,
			skipOWServer: true,
		},
		{
			name:         "missing Kind",
			args:         map[string]any{"NamespaceID": "ns-uuid", "FunctionName": "hello", "Code": "function main() {}"},
			expectError:  true,
			skipOWServer: true,
		},
		{
			name:         "missing Code",
			args:         map[string]any{"NamespaceID": "ns-uuid", "FunctionName": "hello", "Kind": "nodejs:18"},
			expectError:  true,
			skipOWServer: true,
		},
		{
			name: "success",
			args: map[string]any{
				"NamespaceID":  "ns-uuid",
				"FunctionName": "hello",
				"Kind":         "nodejs:18",
				"Code":         "function main(params) { return {body: 'Hello'}; }",
			},
			owHandler: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPut, r.Method)
				require.Contains(t, r.URL.Path, "/actions/hello")
				require.Contains(t, r.URL.RawQuery, "overwrite=true")

				var body map[string]interface{}
				json.NewDecoder(r.Body).Decode(&body)
				exec := body["exec"].(map[string]interface{})
				require.Equal(t, "nodejs:18", exec["kind"])
				require.Contains(t, exec["code"].(string), "Hello")

				json.NewEncoder(w).Encode(map[string]string{"name": "hello"})
			},
		},
		{
			name: "success with web export",
			args: map[string]any{
				"NamespaceID":  "ns-uuid",
				"FunctionName": "hello",
				"Kind":         "python:3.11",
				"Code":         "def main(params): return {'body': 'hi'}",
				"WebExport":    true,
			},
			owHandler: func(w http.ResponseWriter, r *http.Request) {
				var body map[string]interface{}
				json.NewDecoder(r.Body).Decode(&body)
				annotations := body["annotations"].([]interface{})
				require.Len(t, annotations, 2)
				json.NewEncoder(w).Encode(map[string]string{"name": "hello"})
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipOWServer {
				mock := NewMockFunctionsService(ctrl)
				client := func(ctx context.Context) (*godo.Client, error) {
					return &godo.Client{Functions: mock}, nil
				}
				tool := NewFunctionTool(context.Background(), client)
				req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
				resp, _ := tool.create(context.Background(), req)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			mock := NewMockFunctionsService(ctrl)
			tool := setupFunctionToolWithMockAndServer(t, mock, tc.owHandler)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.create(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
		})
	}
}

func TestFunctionTool_deleteFn(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		args         map[string]any
		owHandler    http.HandlerFunc
		expectError  bool
		skipOWServer bool
	}{
		{
			name:         "missing NamespaceID",
			args:         map[string]any{"FunctionName": "hello"},
			expectError:  true,
			skipOWServer: true,
		},
		{
			name:         "missing FunctionName",
			args:         map[string]any{"NamespaceID": "ns-uuid"},
			expectError:  true,
			skipOWServer: true,
		},
		{
			name: "success",
			args: map[string]any{"NamespaceID": "ns-uuid", "FunctionName": "hello"},
			owHandler: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodDelete, r.Method)
				require.Contains(t, r.URL.Path, "/actions/hello")
				json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipOWServer {
				mock := NewMockFunctionsService(ctrl)
				client := func(ctx context.Context) (*godo.Client, error) {
					return &godo.Client{Functions: mock}, nil
				}
				tool := NewFunctionTool(context.Background(), client)
				req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
				resp, _ := tool.deleteFn(context.Background(), req)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			mock := NewMockFunctionsService(ctrl)
			tool := setupFunctionToolWithMockAndServer(t, mock, tc.owHandler)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteFn(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.False(t, resp.IsError)
			textContent, ok := resp.Content[0].(mcp.TextContent)
			require.True(t, ok)
			require.Contains(t, textContent.Text, "deleted successfully")
		})
	}
}

func TestFunctionTool_invoke(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		args         map[string]any
		owHandler    http.HandlerFunc
		expectError  bool
		skipOWServer bool
	}{
		{
			name:         "missing NamespaceID",
			args:         map[string]any{"FunctionName": "hello"},
			expectError:  true,
			skipOWServer: true,
		},
		{
			name:         "missing FunctionName",
			args:         map[string]any{"NamespaceID": "ns-uuid"},
			expectError:  true,
			skipOWServer: true,
		},
		{
			name: "success without params",
			args: map[string]any{"NamespaceID": "ns-uuid", "FunctionName": "hello"},
			owHandler: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPost, r.Method)
				require.Contains(t, r.URL.Path, "/actions/hello")
				require.Contains(t, r.URL.RawQuery, "blocking=true")
				require.Contains(t, r.URL.RawQuery, "result=true")
				json.NewEncoder(w).Encode(map[string]string{"payload": "Hello World"})
			},
		},
		{
			name: "success with params",
			args: map[string]any{
				"NamespaceID":  "ns-uuid",
				"FunctionName": "hello",
				"Params":       map[string]interface{}{"name": "John"},
			},
			owHandler: func(w http.ResponseWriter, r *http.Request) {
				var params map[string]interface{}
				json.NewDecoder(r.Body).Decode(&params)
				require.Equal(t, "John", params["name"])
				json.NewEncoder(w).Encode(map[string]string{"payload": "Hello John"})
			},
		},
		{
			name: "openwhisk api error",
			args: map[string]any{"NamespaceID": "ns-uuid", "FunctionName": "hello"},
			owHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error":"action not found"}`))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipOWServer {
				mock := NewMockFunctionsService(ctrl)
				client := func(ctx context.Context) (*godo.Client, error) {
					return &godo.Client{Functions: mock}, nil
				}
				tool := NewFunctionTool(context.Background(), client)
				req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
				resp, _ := tool.invoke(context.Background(), req)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			mock := NewMockFunctionsService(ctrl)
			tool := setupFunctionToolWithMockAndServer(t, mock, tc.owHandler)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.invoke(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			if tc.name == "openwhisk api error" {
				require.True(t, resp.IsError)
			} else {
				require.False(t, resp.IsError)
				textContent, ok := resp.Content[0].(mcp.TextContent)
				require.True(t, ok)
				require.Contains(t, textContent.Text, "Hello")
			}
		})
	}
}

func TestFunctionTool_listActivations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		args         map[string]any
		owHandler    http.HandlerFunc
		expectError  bool
		skipOWServer bool
	}{
		{
			name:         "missing NamespaceID",
			args:         map[string]any{},
			expectError:  true,
			skipOWServer: true,
		},
		{
			name: "success",
			args: map[string]any{"NamespaceID": "ns-uuid"},
			owHandler: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodGet, r.Method)
				require.Contains(t, r.URL.Path, "/api/v1/namespaces/_/activations")
				json.NewEncoder(w).Encode([]map[string]string{{"activationId": "act-1"}, {"activationId": "act-2"}})
			},
		},
		{
			name: "success with function name filter",
			args: map[string]any{"NamespaceID": "ns-uuid", "FunctionName": "hello"},
			owHandler: func(w http.ResponseWriter, r *http.Request) {
				require.Contains(t, r.URL.RawQuery, "name=hello")
				json.NewEncoder(w).Encode([]map[string]string{{"activationId": "act-1", "name": "hello"}})
			},
		},
		{
			name: "openwhisk api error",
			args: map[string]any{"NamespaceID": "ns-uuid"},
			owHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"internal error"}`))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipOWServer {
				mock := NewMockFunctionsService(ctrl)
				client := func(ctx context.Context) (*godo.Client, error) {
					return &godo.Client{Functions: mock}, nil
				}
				tool := NewFunctionTool(context.Background(), client)
				req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
				resp, _ := tool.listActivations(context.Background(), req)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			mock := NewMockFunctionsService(ctrl)
			tool := setupFunctionToolWithMockAndServer(t, mock, tc.owHandler)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.listActivations(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			if tc.name == "openwhisk api error" {
				require.True(t, resp.IsError)
			} else {
				require.False(t, resp.IsError)
				textContent, ok := resp.Content[0].(mcp.TextContent)
				require.True(t, ok)
				require.Contains(t, textContent.Text, "act-1")
			}
		})
	}
}

func TestFunctionTool_getActivation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		args         map[string]any
		owHandler    http.HandlerFunc
		expectError  bool
		skipOWServer bool
	}{
		{
			name:         "missing NamespaceID",
			args:         map[string]any{"ActivationID": "act-1"},
			expectError:  true,
			skipOWServer: true,
		},
		{
			name:         "missing ActivationID",
			args:         map[string]any{"NamespaceID": "ns-uuid"},
			expectError:  true,
			skipOWServer: true,
		},
		{
			name: "success",
			args: map[string]any{"NamespaceID": "ns-uuid", "ActivationID": "act-1"},
			owHandler: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodGet, r.Method)
				require.Contains(t, r.URL.Path, "/activations/act-1")
				require.NotContains(t, r.URL.Path, "/logs")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"activationId": "act-1",
					"name":         "hello",
					"statusCode":   0,
					"duration":     42,
				})
			},
		},
		{
			name: "not found",
			args: map[string]any{"NamespaceID": "ns-uuid", "ActivationID": "bad-id"},
			owHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error":"The requested resource does not exist."}`))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipOWServer {
				mock := NewMockFunctionsService(ctrl)
				client := func(ctx context.Context) (*godo.Client, error) {
					return &godo.Client{Functions: mock}, nil
				}
				tool := NewFunctionTool(context.Background(), client)
				req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
				resp, _ := tool.getActivation(context.Background(), req)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			mock := NewMockFunctionsService(ctrl)
			tool := setupFunctionToolWithMockAndServer(t, mock, tc.owHandler)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getActivation(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			if tc.name == "not found" {
				require.True(t, resp.IsError)
			} else {
				require.False(t, resp.IsError)
				textContent, ok := resp.Content[0].(mcp.TextContent)
				require.True(t, ok)
				require.Contains(t, textContent.Text, "act-1")
			}
		})
	}
}

func TestFunctionTool_getActivationLogs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name         string
		args         map[string]any
		owHandler    http.HandlerFunc
		expectError  bool
		skipOWServer bool
	}{
		{
			name:         "missing NamespaceID",
			args:         map[string]any{"ActivationID": "act-1"},
			expectError:  true,
			skipOWServer: true,
		},
		{
			name:         "missing ActivationID",
			args:         map[string]any{"NamespaceID": "ns-uuid"},
			expectError:  true,
			skipOWServer: true,
		},
		{
			name: "success",
			args: map[string]any{"NamespaceID": "ns-uuid", "ActivationID": "act-1"},
			owHandler: func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodGet, r.Method)
				require.Contains(t, r.URL.Path, "/activations/act-1/logs")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"logs": []string{"2026-03-25T00:00:00Z stdout: Hello World"},
				})
			},
		},
		{
			name: "not found",
			args: map[string]any{"NamespaceID": "ns-uuid", "ActivationID": "bad-id"},
			owHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error":"The requested resource does not exist."}`))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.skipOWServer {
				mock := NewMockFunctionsService(ctrl)
				client := func(ctx context.Context) (*godo.Client, error) {
					return &godo.Client{Functions: mock}, nil
				}
				tool := NewFunctionTool(context.Background(), client)
				req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
				resp, _ := tool.getActivationLogs(context.Background(), req)
				require.NotNil(t, resp)
				require.True(t, resp.IsError)
				return
			}

			mock := NewMockFunctionsService(ctrl)
			tool := setupFunctionToolWithMockAndServer(t, mock, tc.owHandler)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getActivationLogs(context.Background(), req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			if tc.name == "not found" {
				require.True(t, resp.IsError)
			} else {
				require.False(t, resp.IsError)
				textContent, ok := resp.Content[0].(mcp.TextContent)
				require.True(t, ok)
				require.Contains(t, textContent.Text, "Hello World")
			}
		})
	}
}

func TestResolveNamespace_AccessKeyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := middleware.WithAuthKey(context.Background(), "Bearer test-token-123")
	callCount := 0

	svc := &mockAccessKeySvc{
		createFn: func(_ context.Context, namespace string, req *AccessKeyCreateRequest) (*AccessKey, error) {
			callCount++
			require.Equal(t, "fn-ns-1", namespace)
			require.Equal(t, accessKeyClientName, req.Name)
			require.Equal(t, "1d", req.ExpiresIn)
			return &AccessKey{
				ID:        "dof_v1_abc123",
				Secret:    "dof_v1_abc123:secret456",
				APIHost:   "https://faas.example.com",
				ExpiresAt: time.Now().Add(2 * time.Hour),
				Name:      req.Name,
			}, nil
		},
	}

	mock := NewMockFunctionsService(ctrl)
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{Functions: mock}, nil
	}

	tool := NewFunctionTool(context.Background(), client, WithAccessKeyService(svc))

	info, err := tool.resolveNamespace(ctx, "fn-ns-1")
	require.NoError(t, err)
	require.Equal(t, "https://faas.example.com", info.apiHost)
	require.Equal(t, "dof_v1_abc123:dof_v1_abc123:secret456", info.key)
	require.Equal(t, 1, callCount)

	info2, err := tool.resolveNamespace(ctx, "fn-ns-1")
	require.NoError(t, err)
	require.Equal(t, info.key, info2.key)
	require.Equal(t, 1, callCount, "second call should use cache, not create a new key")
}

func TestResolveNamespace_AccessKeyFallsBackToLegacy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := middleware.WithAuthKey(context.Background(), "Bearer test-token-123")

	svc := &mockAccessKeySvc{
		createFn: func(_ context.Context, _ string, _ *AccessKeyCreateRequest) (*AccessKey, error) {
			return nil, errors.New("access key service unavailable")
		},
	}

	mock := NewMockFunctionsService(ctrl)
	mock.EXPECT().GetNamespace(gomock.Any(), "fn-ns-1").Return(&godo.FunctionsNamespace{
		ApiHost: "https://legacy.example.com",
		UUID:    "legacy-uuid",
		Key:     "legacy-key",
	}, nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{Functions: mock}, nil
	}

	tool := NewFunctionTool(context.Background(), client, WithAccessKeyService(svc))

	info, err := tool.resolveNamespace(ctx, "fn-ns-1")
	require.NoError(t, err)
	require.Equal(t, "https://legacy.example.com", info.apiHost)
	require.Equal(t, "legacy-uuid:legacy-key", info.key)
}

func TestResolveNamespace_NoAccessKeyService_UsesLegacy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockFunctionsService(ctrl)
	mock.EXPECT().GetNamespace(gomock.Any(), "fn-ns-1").Return(&godo.FunctionsNamespace{
		ApiHost: "https://legacy.example.com",
		UUID:    "legacy-uuid",
		Key:     "legacy-key",
	}, nil, nil)

	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{Functions: mock}, nil
	}

	tool := NewFunctionTool(context.Background(), client)

	info, err := tool.resolveNamespace(context.Background(), "fn-ns-1")
	require.NoError(t, err)
	require.Equal(t, "https://legacy.example.com", info.apiHost)
	require.Equal(t, "legacy-uuid:legacy-key", info.key)
}

func TestResolveNamespace_CachePerUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	callCount := 0
	svc := &mockAccessKeySvc{
		createFn: func(_ context.Context, _ string, _ *AccessKeyCreateRequest) (*AccessKey, error) {
			callCount++
			id := fmt.Sprintf("dof_v1_%d", callCount)
			return &AccessKey{
				ID:        id,
				Secret:    id + ":secret",
				APIHost:   "https://faas.example.com",
				ExpiresAt: time.Now().Add(2 * time.Hour),
			}, nil
		},
	}

	mock := NewMockFunctionsService(ctrl)
	client := func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{Functions: mock}, nil
	}

	tool := NewFunctionTool(context.Background(), client, WithAccessKeyService(svc))

	ctx1 := middleware.WithAuthKey(context.Background(), "Bearer user-token-1")
	ctx2 := middleware.WithAuthKey(context.Background(), "Bearer user-token-2")

	_, err := tool.resolveNamespace(ctx1, "fn-ns-1")
	require.NoError(t, err)
	require.Equal(t, 1, callCount)

	_, err = tool.resolveNamespace(ctx2, "fn-ns-1")
	require.NoError(t, err)
	require.Equal(t, 2, callCount, "different user should not share cached key")

	_, err = tool.resolveNamespace(ctx1, "fn-ns-1")
	require.NoError(t, err)
	require.Equal(t, 2, callCount, "same user should reuse cached key")
}
