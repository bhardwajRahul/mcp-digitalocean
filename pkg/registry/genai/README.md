# GenAI Evaluation Tools

This package provides MCP tools for managing and running evaluation workflows in DigitalOcean's GenAI platform.

## Overview

The evaluation tools enable users to:
- List available evaluation metrics
- Manage evaluation datasets (upload CSV files)
- Create and update evaluation test cases
- Run evaluations against agent deployments
- Monitor evaluation run status

## Tools

### Atomic Tools (One API Call Each)

#### `genai-list-evaluation-metrics`
Lists all available evaluation metrics that can be used in test cases.

**Arguments:** None

**Returns:** JSON object with array of metrics and metadata

```json
{
  "metrics": [
    {
      "metric_uuid": "...",
      "metric_name": "correctness",
      "metric_type": "...",
      "category": "METRIC_CATEGORY_CORRECTNESS",
      ...
    }
  ],
  "count": 5
}
```

#### `genai-list-evaluation-test-cases`
Lists evaluation test cases for a specific workspace.

**Arguments:**
- `workspace_uuid` (string, optional): Workspace UUID
- `agent_workspace_name` (string, optional): Workspace name

At least one of `workspace_uuid` or `agent_workspace_name` must be provided.

**Returns:** JSON object with array of test cases

```json
{
  "test_cases": [
    {
      "test_case_uuid": "...",
      "name": "my_test",
      "description": "...",
      ...
    }
  ],
  "count": 2
}
```

#### `genai-create-evaluation-dataset`
Creates an evaluation dataset by uploading a CSV file. The file is validated to ensure:
- File extension is `.csv`
- Contains a `query` column
- All query column values are valid JSON objects

**Arguments:**
- `name` (string, required): Name for the dataset
- `file_path` (string, required): Path to the CSV file to upload

**Returns:** JSON object with dataset UUID and metadata

```json
{
  "dataset_uuid": "...",
  "name": "my_dataset",
  "file_size": 1024
}
```

#### `genai-create-evaluation-test-case`
Creates a new evaluation test case.

**Arguments:**
- `name` (string, required): Name of the test case
- `description` (string, optional): Description
- `dataset_uuid` (string, required): Dataset UUID to use
- `metrics` (array of strings, optional): Metric UUIDs to include
- `workspace_uuid` (string, optional): Workspace UUID
- `agent_workspace_name` (string, optional): Workspace name

At least one of `workspace_uuid` or `agent_workspace_name` must be provided.

**Returns:** JSON object with test case UUID

```json
{
  "test_case_uuid": "...",
  "name": "my_test",
  "dataset_uuid": "..."
}
```

#### `genai-update-evaluation-test-case`
Updates an existing evaluation test case.

**Arguments:**
- `test_case_uuid` (string, required): Test case UUID to update
- `name` (string, optional): New name
- `description` (string, optional): New description
- `dataset_uuid` (string, optional): New dataset UUID
- `metrics` (array of strings, optional): New metric UUIDs

**Returns:** JSON object with test case UUID and new version

```json
{
  "test_case_uuid": "...",
  "version": 2
}
```

#### `genai-run-evaluation-test-case`
Runs an evaluation test case against specified agent deployments.

**Arguments:**
- `test_case_uuid` (string, required): Test case UUID to run
- `agent_deployment_names` (array of strings, required): Deployment names to evaluate
- `run_name` (string, required): Name for this evaluation run

**Returns:** JSON object with evaluation run UUIDs

```json
{
  "evaluation_run_uuids": ["uuid1", "uuid2"],
  "count": 2
}
```

#### `genai-get-evaluation-run`
Gets the status and results of an evaluation run.

**Arguments:**
- `evaluation_run_uuid` (string, required): Evaluation run UUID

**Returns:** JSON object with full evaluation run details

```json
{
  "evaluation_run": {
    "evaluation_run_uuid": "...",
    "status": "EVALUATION_RUN_SUCCESSFUL",
    "run_level_metric_results": [
      {
        "metric_name": "correctness",
        "number_value": 0.95,
        "reasoning": "..."
      }
    ],
    ...
  }
}
```

### High-Level Orchestrated Tool

#### `genai-run-evaluation-workflow`
Runs a complete end-to-end evaluation workflow. This tool orchestrates all the steps:
1. Validates the dataset CSV
2. Lists available metrics and filters by category
3. Uploads the dataset
4. Creates or updates the test case
5. Runs the evaluation
6. Polls for results until completion (with configurable timeout)

This tool is ideal for users unfamiliar with the multi-step evaluation process, as it handles all orchestration internally.

**Arguments:**
- `dataset_file_path` (string, required): Path to CSV evaluation dataset
- `workspace_name` (string, required): Agent workspace name
- `test_case_name` (string, required): Name for the test case
- `agent_deployment_names` (array of strings, required): Deployment names to evaluate
- `run_name` (string, required): Name for the evaluation run
- `description` (string, optional): Test case description
- `metric_categories` (array of strings, optional): Filter by metric categories (e.g., `"METRIC_CATEGORY_CORRECTNESS"`, `"METRIC_CATEGORY_SAFETY_AND_SECURITY"`). If empty, all metrics are used.
- `timeout_seconds` (number, optional): Timeout for polling results (default: 300 seconds)
- `poll_interval_seconds` (number, optional): Interval between status polls (default: 5 seconds)

**Returns:** JSON object with complete workflow results

```json
{
  "dataset_uuid": "...",
  "test_case_uuid": "...",
  "evaluation_run_uuid": "...",
  "status": "EVALUATION_RUN_SUCCESSFUL",
  "metric_results": [
    {
      "metric_name": "correctness",
      "number_value": 0.95,
      "reasoning": "..."
    }
  ],
  "duration_seconds": 45.3,
  "error_message": null
}
```

## Dataset CSV Format

Evaluation datasets must be CSV files with:
- A `query` column containing JSON objects as strings
- Additional columns for ground truth, expected outputs, etc. (optional)

Example:
```csv
query,expected_output
"{\"question\": \"What is 2+2?\"}",4
"{\"question\": \"What is the capital of France?\"}","Paris"
```

## Workflow Example

### Using Atomic Tools (Step-by-Step)

```
1. List metrics to see available options:
   genai-list-evaluation-metrics

2. Upload your dataset:
   genai-create-evaluation-dataset
     name: "my_dataset"
     file_path: "/path/to/queries.csv"

3. Create a test case:
   genai-create-evaluation-test-case
     name: "test_my_agent"
     description: "Testing agent correctness"
     dataset_uuid: "<uuid from step 2>"
     metrics: ["<metric_uuid_1>", "<metric_uuid_2>"]
     agent_workspace_name: "my_workspace"

4. Run the evaluation:
   genai-run-evaluation-test-case
     test_case_uuid: "<uuid from step 3>"
     agent_deployment_names: ["my_agent_deployment"]
     run_name: "run_1"

5. Poll for results:
   genai-get-evaluation-run
     evaluation_run_uuid: "<uuid from step 4>"
```

### Using the Orchestrated Workflow Tool (All-in-One)

```
genai-run-evaluation-workflow
  dataset_file_path: "/path/to/queries.csv"
  workspace_name: "my_workspace"
  test_case_name: "test_my_agent"
  agent_deployment_names: ["my_agent_deployment"]
  run_name: "run_1"
  description: "Testing agent correctness"
  metric_categories: ["METRIC_CATEGORY_CORRECTNESS"]
  timeout_seconds: 300
  poll_interval_seconds: 5
```

## CSV Validation

The CSV dataset is validated to ensure:
1. File has `.csv` extension
2. File contains a `query` column
3. All `query` column values are valid JSON
4. File is readable and not empty

If validation fails, a detailed error message is returned describing the issue.

## Error Handling

All tools return structured error messages. Errors from API calls are wrapped with context about which step failed:

```json
{
  "error": "failed to create evaluation dataset: service error"
}
```

For workflow tool, errors include the step number:
```
"step 4: failed to create presigned URL: ..."
"step 7: evaluation polling timed out"
```

## Metric Categories

Available metric categories (when filtering in workflow tool):
- `METRIC_CATEGORY_CORRECTNESS`: Correctness and accuracy metrics
- `METRIC_CATEGORY_USER_OUTCOMES`: User satisfaction and engagement metrics
- `METRIC_CATEGORY_SAFETY_AND_SECURITY`: Safety and security related metrics
- `METRIC_CATEGORY_CONTEXT_QUALITY`: Context and retrieval quality metrics
- `METRIC_CATEGORY_MODEL_FIT`: Model fit and performance metrics

## Evaluation Run Status Values

- `EVALUATION_RUN_QUEUED`: Run is waiting to start
- `EVALUATION_RUN_RUNNING`: Run is currently executing
- `EVALUATION_RUN_RUNNING_DATASET`: Processing dataset
- `EVALUATION_RUN_EVALUATING_RESULTS`: Evaluating metric results
- `EVALUATION_RUN_SUCCESSFUL`: Run completed successfully
- `EVALUATION_RUN_PARTIALLY_SUCCESSFUL`: Some metrics were evaluated, others failed
- `EVALUATION_RUN_FAILED`: Run failed completely
- `EVALUATION_RUN_CANCELLED`: Run was cancelled

Terminal statuses: `SUCCESSFUL`, `FAILED`, `CANCELLED`, `PARTIALLY_SUCCESSFUL`
