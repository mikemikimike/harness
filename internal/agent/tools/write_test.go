package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"charm.land/fantasy"
	"github.com/stretchr/testify/require"
)

type mockFileTrackerService struct{}

func (m mockFileTrackerService) RecordRead(ctx context.Context, sessionID, path string) {}

func (m mockFileTrackerService) LastReadTime(ctx context.Context, sessionID, path string) time.Time {
	return time.Now()
}

func (m mockFileTrackerService) ListReadFiles(ctx context.Context, sessionID string) ([]string, error) {
	return nil, nil
}

func TestWriteToolWritesEmptyNewFile(t *testing.T) {
	t.Parallel()

	workingDir := t.TempDir()
	ctx := context.WithValue(context.Background(), SessionIDContextKey, "test-session")

	tool := NewWriteTool(nil, &mockHistoryService{}, mockFileTrackerService{}, nil, workingDir)

	input, err := json.Marshal(WriteParams{FilePath: "empty.txt", Content: ""})
	require.NoError(t, err)

	resp, err := tool.Run(ctx, fantasy.ToolCall{
		ID:    "test-call",
		Name:  WriteToolName,
		Input: string(input),
	})
	require.NoError(t, err)
	require.False(t, resp.IsError)

	b, err := os.ReadFile(filepath.Join(workingDir, "empty.txt"))
	require.NoError(t, err)
	require.Equal(t, "", string(b))
}

// TestWriteToolUnwritableDirIsErrorResponse: a file whose parent
// directory cannot be created is an error result for the model rather
// than a Go error that aborts the turn.
func TestWriteToolUnwritableDirIsErrorResponse(t *testing.T) {
	t.Parallel()

	workingDir := t.TempDir()
	readOnly := unwritableDir(t, workingDir, "ro")
	ctx := context.WithValue(context.Background(), SessionIDContextKey, "test-session")
	tool := NewWriteTool(nil, &mockHistoryService{}, mockFileTrackerService{}, nil, workingDir)

	resp := runFileTool(t, tool, ctx, WriteParams{FilePath: filepath.Join(readOnly, "sub", "new.txt"), Content: "x"})
	require.True(t, resp.IsError)
	require.Contains(t, resp.Content, "Cannot create the parent directories of")
	require.Contains(t, resp.Content, "permissions")
}
