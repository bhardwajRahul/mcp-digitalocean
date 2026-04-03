//go:build integration

package testing

import (
	"fmt"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
)

func TestVolumeLifecycle(t *testing.T) {
	t.Parallel()

	newVolume := CreateTestVolume(t, "mcp-e2e-volume")

	getVolume := callTool[godo.Volume](t, "volume-get", map[string]any{
		"VolumeID": newVolume.ID,
	})

	require.Equal(t, newVolume.ID, getVolume.ID)
	t.Logf("[Get] Successfully retrieved volume:")
	t.Logf("      Name: %s", getVolume.Name)
	t.Logf("      ID: %s", getVolume.ID)
	t.Logf("      Region: %s", getVolume.Region.Slug)
	t.Logf("      Size: %d", getVolume.SizeGigaBytes)

	snapshotName := fmt.Sprintf("%s-snap-%d", newVolume.Name, time.Now().Unix())
	t.Logf("[Create] Creating snapshot: %s...", snapshotName)

	newSnapshot := callTool[godo.Snapshot](t, "volume-snapshot-create", map[string]any{
		"VolumeID": newVolume.ID,
		"Name":     snapshotName,
	})
	t.Logf("[Created] Snapshot %s: Name=%s Size=%f", newSnapshot.ID, newSnapshot.Name, newSnapshot.SizeGigaBytes)

	getSnapshot := callTool[godo.Snapshot](t, "volume-snapshot-get", map[string]any{
		"SnapshotID": newSnapshot.ID,
	})
	require.Equal(t, newSnapshot.ID, getSnapshot.ID)
	t.Logf("[Get] Successfully retrieved snapshot:")
	t.Logf("      Name: %s", getSnapshot.Name)
	t.Logf("      ID: %s", getSnapshot.ID)

	t.Logf("[Delete] Deleting snapshot: %s...", snapshotName)

	DeleteResource(t, "volume-snapshot", newSnapshot.ID)

	t.Logf("[Delete] Deleting volume: %s...", newVolume.Name)

	DeleteResource(t, "volume", newVolume.ID)
}
