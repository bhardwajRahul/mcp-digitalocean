package dedicatedinference

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupToolWithMock(mock godo.DedicatedInferenceService) *DedicatedInferenceTool {
	return NewDedicatedInferenceTool(func(ctx context.Context) (*godo.Client, error) {
		return &godo.Client{DedicatedInference: mock}, nil
	})
}

func setupToolWithClientError() *DedicatedInferenceTool {
	return NewDedicatedInferenceTool(func(ctx context.Context) (*godo.Client, error) {
		return nil, errors.New("auth failed")
	})
}

var testDI = &godo.DedicatedInference{
	ID:      "di-uuid-1234",
	Name:    "my-inference",
	Region:  "nyc2",
	Status:  "ACTIVE",
	VPCUUID: "vpc-uuid",
	Endpoints: &godo.DedicatedInferenceEndpoints{
		PublicEndpointFQDN:  "my-inference.public.example.com",
		PrivateEndpointFQDN: "my-inference.private.example.com",
	},
	DeploymentSpec: &godo.DedicatedInferenceDeployment{
		Version:              1,
		EnablePublicEndpoint: true,
		ModelDeployments: []*godo.DedicatedInferenceModelDeployment{
			{
				ModelID:       "model-deploy-uuid-1",
				ModelSlug:     "deepseek-ai/DeepSeek-R1",
				ModelProvider: "hugging_face",
				Accelerators: []*godo.DedicatedInferenceAccelerator{
					{AcceleratorSlug: "gpu-h100x8", Scale: 1, Type: "prefill_decode"},
				},
			},
		},
	},
	CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	UpdatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
}

var testToken = &godo.DedicatedInferenceToken{
	ID:        "token-uuid-1234",
	Name:      "default",
	Value:     "secret-token-value",
	IsManaged: false,
	CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
}

func TestDedicatedInferenceTool_create(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDedicatedInferenceService)
		expectError bool
		checkResult func(*testing.T, *mcp.CallToolResult)
	}{
		{
			name:        "missing Name",
			args:        map[string]any{"Region": "nyc2"},
			expectError: true,
		},
		{
			name:        "missing Region",
			args:        map[string]any{"Name": "my-inference"},
			expectError: true,
		},
		{
			name:        "missing ModelDeployments",
			args:        map[string]any{"Name": "my-inference", "Region": "nyc2"},
			expectError: true,
		},
		{
			name: "malformed deployment entry skipped",
			args: map[string]any{
				"Name":   "my-inference",
				"Region": "nyc2",
				"ModelDeployments": []any{
					"not-a-map",
				},
			},
			expectError: true,
		},
		{
			name: "malformed accelerator entry skipped",
			args: map[string]any{
				"Name":   "my-inference",
				"Region": "nyc2",
				"ModelDeployments": []any{
					map[string]any{
						"ModelSlug":     "deepseek-ai/DeepSeek-R1",
						"ModelProvider": "hugging_face",
						"Accelerators":  []any{"not-a-map"},
					},
				},
			},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, req *godo.DedicatedInferenceCreateRequest) (*godo.DedicatedInference, *godo.DedicatedInferenceToken, *godo.Response, error) {
						require.Len(t, req.Spec.ModelDeployments, 1)
						require.Empty(t, req.Spec.ModelDeployments[0].Accelerators)
						return testDI, testToken, &godo.Response{}, nil
					})
			},
		},
		{
			name: "invalid scale skipped",
			args: map[string]any{
				"Name":   "my-inference",
				"Region": "nyc2",
				"ModelDeployments": []any{
					map[string]any{
						"ModelSlug":     "deepseek-ai/DeepSeek-R1",
						"ModelProvider": "hugging_face",
						"Accelerators": []any{
							map[string]any{"AcceleratorSlug": "gpu-h100x8", "Scale": "not-a-number", "Type": "prefill_decode"},
						},
					},
				},
			},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, req *godo.DedicatedInferenceCreateRequest) (*godo.DedicatedInference, *godo.DedicatedInferenceToken, *godo.Response, error) {
						require.Empty(t, req.Spec.ModelDeployments[0].Accelerators)
						return testDI, testToken, &godo.Response{}, nil
					})
			},
		},
		{
			name: "zero scale skipped",
			args: map[string]any{
				"Name":   "my-inference",
				"Region": "nyc2",
				"ModelDeployments": []any{
					map[string]any{
						"ModelSlug":     "deepseek-ai/DeepSeek-R1",
						"ModelProvider": "hugging_face",
						"Accelerators": []any{
							map[string]any{"AcceleratorSlug": "gpu-h100x8", "Scale": float64(0), "Type": "prefill_decode"},
						},
					},
				},
			},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, req *godo.DedicatedInferenceCreateRequest) (*godo.DedicatedInference, *godo.DedicatedInferenceToken, *godo.Response, error) {
						require.Empty(t, req.Spec.ModelDeployments[0].Accelerators)
						return testDI, testToken, &godo.Response{}, nil
					})
			},
		},
		{
			name: "ModelID passed through",
			args: map[string]any{
				"Name":   "my-inference",
				"Region": "nyc2",
				"ModelDeployments": []any{
					map[string]any{
						"ModelSlug":     "deepseek-ai/DeepSeek-R1",
						"ModelProvider": "hugging_face",
						"ModelID":       "model-deploy-uuid-1",
						"Accelerators": []any{
							map[string]any{"AcceleratorSlug": "gpu-h100x8", "Scale": float64(1), "Type": "prefill_decode"},
						},
					},
				},
			},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, req *godo.DedicatedInferenceCreateRequest) (*godo.DedicatedInference, *godo.DedicatedInferenceToken, *godo.Response, error) {
						require.Equal(t, "model-deploy-uuid-1", req.Spec.ModelDeployments[0].ModelID)
						return testDI, testToken, &godo.Response{}, nil
					})
			},
		},
		{
			name: "api error",
			args: map[string]any{
				"Name":   "my-inference",
				"Region": "nyc2",
				"ModelDeployments": []any{
					map[string]any{
						"ModelSlug":     "deepseek-ai/DeepSeek-R1",
						"ModelProvider": "hugging_face",
						"Accelerators": []any{
							map[string]any{"AcceleratorSlug": "gpu-h100x8", "Scale": float64(1), "Type": "prefill_decode"},
						},
					},
				},
			},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success with full spec",
			args: map[string]any{
				"Name":                 "my-inference",
				"Region":               "nyc2",
				"EnablePublicEndpoint": true,
				"VPCUUID":              "vpc-uuid",
				"ModelDeployments": []any{
					map[string]any{
						"ModelSlug":     "deepseek-ai/DeepSeek-R1",
						"ModelProvider": "hugging_face",
						"Accelerators": []any{
							map[string]any{"AcceleratorSlug": "gpu-h100x8", "Scale": float64(1), "Type": "prefill_decode"},
						},
					},
				},
				"HuggingFaceToken": "hf_test_token",
			},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, req *godo.DedicatedInferenceCreateRequest) (*godo.DedicatedInference, *godo.DedicatedInferenceToken, *godo.Response, error) {
						require.Equal(t, 1, req.Spec.Version)
						require.Equal(t, "my-inference", req.Spec.Name)
						require.Equal(t, "nyc2", req.Spec.Region)
						require.True(t, req.Spec.EnablePublicEndpoint)
						require.Equal(t, "vpc-uuid", req.Spec.VPC.UUID)
						require.Len(t, req.Spec.ModelDeployments, 1)
						require.Equal(t, "deepseek-ai/DeepSeek-R1", req.Spec.ModelDeployments[0].ModelSlug)
						require.Equal(t, "hf_test_token", req.Secrets.HuggingFaceToken)
						return testDI, testToken, &godo.Response{}, nil
					})
			},
			checkResult: func(t *testing.T, result *mcp.CallToolResult) {
				tc, ok := result.Content[0].(mcp.TextContent)
				require.True(t, ok)

				var resp struct {
					DedicatedInference *godo.DedicatedInference      `json:"dedicated_inference"`
					Token              *godo.DedicatedInferenceToken `json:"token"`
				}
				err := json.Unmarshal([]byte(tc.Text), &resp)
				require.NoError(t, err)
				require.Equal(t, testDI.ID, resp.DedicatedInference.ID)
				require.Equal(t, testDI.Name, resp.DedicatedInference.Name)
				require.Equal(t, testToken.ID, resp.Token.ID)
			},
		},
		{
			name: "success without secrets",
			args: map[string]any{
				"Name":   "my-inference",
				"Region": "nyc2",
				"ModelDeployments": []any{
					map[string]any{
						"ModelSlug":     "meta-llama/Llama-3-8B",
						"ModelProvider": "hugging_face",
						"Accelerators": []any{
							map[string]any{"AcceleratorSlug": "gpu-a100", "Scale": float64(2), "Type": "prefill_decode"},
						},
					},
				},
			},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, req *godo.DedicatedInferenceCreateRequest) (*godo.DedicatedInference, *godo.DedicatedInferenceToken, *godo.Response, error) {
						require.Nil(t, req.Secrets)
						return testDI, testToken, &godo.Response{}, nil
					})
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockDedicatedInferenceService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.createDedicatedInference(context.Background(), req)

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

			if tc.checkResult != nil {
				tc.checkResult(t, resp)
			}
		})
	}
}

func TestDedicatedInferenceTool_get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDedicatedInferenceService)
		expectError bool
	}{
		{
			name:        "missing DedicatedInferenceID",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name:        "empty DedicatedInferenceID",
			args:        map[string]any{"DedicatedInferenceID": ""},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"DedicatedInferenceID": "di-uuid-1234"},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().Get(gomock.Any(), "di-uuid-1234").Return(nil, nil, errors.New("not found"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"DedicatedInferenceID": "di-uuid-1234"},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().Get(gomock.Any(), "di-uuid-1234").Return(testDI, &godo.Response{}, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockDedicatedInferenceService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.getDedicatedInference(context.Background(), req)

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
			require.NotEmpty(t, resp.Content)

			tc2, ok := resp.Content[0].(mcp.TextContent)
			require.True(t, ok)

			var result godo.DedicatedInference
			err = json.Unmarshal([]byte(tc2.Text), &result)
			require.NoError(t, err)
			require.Equal(t, testDI.ID, result.ID)
			require.Equal(t, testDI.Name, result.Name)
			require.Equal(t, testDI.Region, result.Region)
			require.Equal(t, testDI.Status, result.Status)
		})
	}
}

func TestDedicatedInferenceTool_list(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testItems := []godo.DedicatedInferenceListItem{
		{ID: "di-1", Name: "inference-1", Region: "nyc2", Status: "ACTIVE"},
		{ID: "di-2", Name: "inference-2", Region: "tor1", Status: "PROVISIONING"},
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDedicatedInferenceService)
		expectError bool
		expectCount int
		checkResult func(*testing.T, *mcp.CallToolResult)
	}{
		{
			name: "list all",
			args: map[string]any{},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().List(gomock.Any(), gomock.Any()).Return(testItems, &godo.Response{
					Meta: &godo.Meta{Total: 2, Page: 1, Pages: 1},
				}, nil)
			},
			expectCount: 2,
			checkResult: func(t *testing.T, result *mcp.CallToolResult) {
				tc, ok := result.Content[0].(mcp.TextContent)
				require.True(t, ok)
				var resp listResponse
				err := json.Unmarshal([]byte(tc.Text), &resp)
				require.NoError(t, err)
				require.NotNil(t, resp.Meta)
				require.Equal(t, 2, resp.Meta.Total)
			},
		},
		{
			name: "filter by region",
			args: map[string]any{"Region": "nyc2"},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().List(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, opts *godo.DedicatedInferenceListOptions) ([]godo.DedicatedInferenceListItem, *godo.Response, error) {
						require.Equal(t, "nyc2", opts.Region)
						return testItems[:1], &godo.Response{}, nil
					})
			},
			expectCount: 1,
		},
		{
			name: "filter by name",
			args: map[string]any{"Name": "inference-1"},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().List(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, opts *godo.DedicatedInferenceListOptions) ([]godo.DedicatedInferenceListItem, *godo.Response, error) {
						require.Equal(t, "inference-1", opts.Name)
						return testItems[:1], &godo.Response{}, nil
					})
			},
			expectCount: 1,
		},
		{
			name: "with pagination",
			args: map[string]any{"Page": float64(2), "PerPage": float64(10)},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().List(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, opts *godo.DedicatedInferenceListOptions) ([]godo.DedicatedInferenceListItem, *godo.Response, error) {
						require.Equal(t, 2, opts.Page)
						require.Equal(t, 10, opts.PerPage)
						return []godo.DedicatedInferenceListItem{}, &godo.Response{}, nil
					})
			},
			expectCount: 0,
		},
		{
			name: "api error",
			args: map[string]any{},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockDedicatedInferenceService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.listDedicatedInferences(context.Background(), req)

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
			require.NotEmpty(t, resp.Content)

			textContent, ok := resp.Content[0].(mcp.TextContent)
			require.True(t, ok)

			var result listResponse
			err = json.Unmarshal([]byte(textContent.Text), &result)
			require.NoError(t, err)
			require.Len(t, result.Items, tc.expectCount)

			if tc.checkResult != nil {
				tc.checkResult(t, resp)
			}
		})
	}
}

func TestDedicatedInferenceTool_update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	updatedDI := &godo.DedicatedInference{
		ID:     "di-uuid-1234",
		Name:   "updated-inference",
		Region: "nyc2",
		Status: "UPDATING",
	}

	minimalValidModelDeployments := []any{
		map[string]any{
			"ModelSlug":     "deepseek-ai/DeepSeek-R1",
			"ModelProvider": "hugging_face",
			"ModelID":       "model-deploy-uuid-1",
			"Accelerators": []any{
				map[string]any{"AcceleratorSlug": "gpu-h100x8", "Scale": float64(1), "Type": "prefill_decode"},
			},
		},
	}

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDedicatedInferenceService)
		expectError bool
	}{
		{
			name:        "missing DedicatedInferenceID",
			args:        map[string]any{"Name": "new-name"},
			expectError: true,
		},
		{
			name:        "missing ModelDeployments",
			args:        map[string]any{"DedicatedInferenceID": "di-uuid-1234", "Name": "updated-inference"},
			expectError: true,
		},
		{
			name:        "empty ModelDeployments",
			args:        map[string]any{"DedicatedInferenceID": "di-uuid-1234", "ModelDeployments": []any{}},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{
				"DedicatedInferenceID": "di-uuid-1234",
				"Name":                 "new-name",
				"ModelDeployments":     minimalValidModelDeployments,
			},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().Update(gomock.Any(), "di-uuid-1234", gomock.Any()).Return(nil, nil, errors.New("api error"))
			},
			expectError: true,
		},
		{
			name: "success with name update",
			args: map[string]any{
				"DedicatedInferenceID": "di-uuid-1234",
				"Name":                 "updated-inference",
				"ModelDeployments":     minimalValidModelDeployments,
			},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().Update(gomock.Any(), "di-uuid-1234", gomock.Any()).
					DoAndReturn(func(_ context.Context, id string, req *godo.DedicatedInferenceUpdateRequest) (*godo.DedicatedInference, *godo.Response, error) {
						require.Equal(t, "updated-inference", req.Spec.Name)
						return updatedDI, &godo.Response{}, nil
					})
			},
		},
		{
			name: "success with secrets",
			args: map[string]any{
				"DedicatedInferenceID": "di-uuid-1234",
				"HuggingFaceToken":     "hf_new_token",
				"ModelDeployments":     minimalValidModelDeployments,
			},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().Update(gomock.Any(), "di-uuid-1234", gomock.Any()).
					DoAndReturn(func(_ context.Context, id string, req *godo.DedicatedInferenceUpdateRequest) (*godo.DedicatedInference, *godo.Response, error) {
						require.NotNil(t, req.Secrets)
						require.Equal(t, "hf_new_token", req.Secrets.HuggingFaceToken)
						return updatedDI, &godo.Response{}, nil
					})
			},
		},
		{
			name: "success with region update",
			args: map[string]any{
				"DedicatedInferenceID": "di-uuid-1234",
				"Region":               "tor1",
				"ModelDeployments":     minimalValidModelDeployments,
			},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().Update(gomock.Any(), "di-uuid-1234", gomock.Any()).
					DoAndReturn(func(_ context.Context, id string, req *godo.DedicatedInferenceUpdateRequest) (*godo.DedicatedInference, *godo.Response, error) {
						require.Equal(t, "tor1", req.Spec.Region)
						return updatedDI, &godo.Response{}, nil
					})
			},
		},
		{
			name: "success with enable public endpoint",
			args: map[string]any{
				"DedicatedInferenceID": "di-uuid-1234",
				"EnablePublicEndpoint": true,
				"ModelDeployments":     minimalValidModelDeployments,
			},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().Update(gomock.Any(), "di-uuid-1234", gomock.Any()).
					DoAndReturn(func(_ context.Context, id string, req *godo.DedicatedInferenceUpdateRequest) (*godo.DedicatedInference, *godo.Response, error) {
						require.True(t, req.Spec.EnablePublicEndpoint)
						return updatedDI, &godo.Response{}, nil
					})
			},
		},
		{
			name: "success with VPC UUID",
			args: map[string]any{
				"DedicatedInferenceID": "di-uuid-1234",
				"VPCUUID":              "new-vpc-uuid",
				"ModelDeployments":     minimalValidModelDeployments,
			},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().Update(gomock.Any(), "di-uuid-1234", gomock.Any()).
					DoAndReturn(func(_ context.Context, id string, req *godo.DedicatedInferenceUpdateRequest) (*godo.DedicatedInference, *godo.Response, error) {
						require.NotNil(t, req.Spec.VPC)
						require.Equal(t, "new-vpc-uuid", req.Spec.VPC.UUID)
						return updatedDI, &godo.Response{}, nil
					})
			},
		},
		{
			name: "success with model deployments",
			args: map[string]any{
				"DedicatedInferenceID": "di-uuid-1234",
				"ModelDeployments": []any{
					map[string]any{
						"ModelSlug":     "deepseek-ai/DeepSeek-R1",
						"ModelProvider": "hugging_face",
						"ModelID":       "model-deploy-uuid-1",
						"Accelerators": []any{
							map[string]any{"AcceleratorSlug": "gpu-h100x8", "Scale": float64(1), "Type": "prefill_decode"},
						},
					},
				},
			},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().Update(gomock.Any(), "di-uuid-1234", gomock.Any()).
					DoAndReturn(func(_ context.Context, id string, req *godo.DedicatedInferenceUpdateRequest) (*godo.DedicatedInference, *godo.Response, error) {
						require.Len(t, req.Spec.ModelDeployments, 1)
						require.Equal(t, "model-deploy-uuid-1", req.Spec.ModelDeployments[0].ModelID)
						require.Equal(t, "deepseek-ai/DeepSeek-R1", req.Spec.ModelDeployments[0].ModelSlug)
						require.Equal(t, "hugging_face", req.Spec.ModelDeployments[0].ModelProvider)
						require.Len(t, req.Spec.ModelDeployments[0].Accelerators, 1)
						require.Equal(t, uint64(1), req.Spec.ModelDeployments[0].Accelerators[0].Scale)
						require.Equal(t, "prefill_decode", req.Spec.ModelDeployments[0].Accelerators[0].Type)
						return updatedDI, &godo.Response{}, nil
					})
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockDedicatedInferenceService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.updateDedicatedInference(context.Background(), req)

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

			var result godo.DedicatedInference
			err = json.Unmarshal([]byte(tc2.Text), &result)
			require.NoError(t, err)
			require.Equal(t, updatedDI.ID, result.ID)
		})
	}
}

func TestDedicatedInferenceTool_delete(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name        string
		args        map[string]any
		mockSetup   func(*MockDedicatedInferenceService)
		expectError bool
	}{
		{
			name:        "missing DedicatedInferenceID",
			args:        map[string]any{},
			expectError: true,
		},
		{
			name:        "empty DedicatedInferenceID",
			args:        map[string]any{"DedicatedInferenceID": ""},
			expectError: true,
		},
		{
			name: "api error",
			args: map[string]any{"DedicatedInferenceID": "di-uuid-1234"},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().Delete(gomock.Any(), "di-uuid-1234").Return(nil, errors.New("not found"))
			},
			expectError: true,
		},
		{
			name: "success",
			args: map[string]any{"DedicatedInferenceID": "di-uuid-1234"},
			mockSetup: func(m *MockDedicatedInferenceService) {
				m.EXPECT().Delete(gomock.Any(), "di-uuid-1234").Return(&godo.Response{}, nil)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := NewMockDedicatedInferenceService(ctrl)
			if tc.mockSetup != nil {
				tc.mockSetup(mock)
			}
			tool := setupToolWithMock(mock)
			req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: tc.args}}
			resp, err := tool.deleteDedicatedInference(context.Background(), req)

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
			require.Contains(t, tc2.Text, "success")
		})
	}
}

func TestDedicatedInferenceTool_Tools(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mock := NewMockDedicatedInferenceService(ctrl)
	tool := setupToolWithMock(mock)

	tools := tool.Tools()
	require.Len(t, tools, 5)

	toolNames := make(map[string]bool)
	for _, st := range tools {
		toolNames[st.Tool.Name] = true
	}

	require.True(t, toolNames["dedicated-inference-create"])
	require.True(t, toolNames["dedicated-inference-get"])
	require.True(t, toolNames["dedicated-inference-list"])
	require.True(t, toolNames["dedicated-inference-update"])
	require.True(t, toolNames["dedicated-inference-delete"])
}

func TestDedicatedInferenceTool_clientError(t *testing.T) {
	tool := setupToolWithClientError()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"DedicatedInferenceID": "di-uuid-1234",
		"Name":                 "test",
		"Region":               "nyc2",
		"ModelDeployments": []any{
			map[string]any{
				"ModelSlug":     "deepseek-ai/DeepSeek-R1",
				"ModelProvider": "hugging_face",
				"Accelerators": []any{
					map[string]any{"AcceleratorSlug": "gpu-h100x8", "Scale": float64(1), "Type": "prefill_decode"},
				},
			},
		},
	}}}

	handlers := []struct {
		name string
		fn   func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
	}{
		{"create", tool.createDedicatedInference},
		{"get", tool.getDedicatedInference},
		{"list", tool.listDedicatedInferences},
		{"update", tool.updateDedicatedInference},
		{"delete", tool.deleteDedicatedInference},
	}

	for _, h := range handlers {
		t.Run(h.name, func(t *testing.T) {
			_, err := h.fn(context.Background(), req)
			require.Error(t, err)
			require.Contains(t, err.Error(), "auth failed")
		})
	}
}
