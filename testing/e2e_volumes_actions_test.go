//go:build integration

package testing

import (
	"testing"

	"github.com/digitalocean/godo"
	"github.com/stretchr/testify/require"
)

func TestVolumeActionTools(t *testing.T) {
	t.Parallel()

	d := CreateTestDroplet(t, "mcp-e2e-vol-actions")
	v := CreateTestVolume(t, "mcp-e2e-vol-actions")
	require.Equal(t, d.Region.Slug, v.Region.Slug, "droplet and volume must be in the same region to attach")

	t.Logf("[Action] Attaching volume %s to droplet %d...", v.ID, d.ID)
	attachFinal := triggerVolumeActionAndWait(t, "volume-attach", map[string]any{
		"VolumeID":  v.ID,
		"DropletID": float64(d.ID),
	}, v.ID)

	t.Logf("[Get] Volume after attach:")
	attached := callTool[godo.Volume](t, "volume-get", map[string]any{"VolumeID": v.ID})
	require.Contains(t, attached.DropletIDs, d.ID, "volume should list droplet after attach")
	t.Logf("      ID: %s", attached.ID)
	t.Logf("      DropletIDs: %v", attached.DropletIDs)

	t.Logf("[Get] Volume action by ID:")
	gotAttach := callTool[godo.Action](t, "volume-action-get", map[string]any{
		"VolumeID": v.ID,
		"ActionID": float64(attachFinal.ID),
	})
	require.Equal(t, attachFinal.ID, gotAttach.ID)
	require.Equal(t, attachFinal.Type, gotAttach.Type)
	t.Logf("      ID: %d", gotAttach.ID)
	t.Logf("      Type: %s", gotAttach.Type)
	t.Logf("      Status: %s", gotAttach.Status)

	t.Logf("[List] Volume actions:")
	actions := callTool[[]godo.Action](t, "volume-action-list", map[string]any{
		"VolumeID": v.ID,
		"Page":     float64(1),
		"PerPage":  float64(50),
	})
	requireFoundInList(t, actions, func(a godo.Action) bool { return a.ID == attachFinal.ID }, "attach action")
	t.Logf("      Count: %d", len(actions))

	t.Logf("[Action] Detaching volume %s from droplet %d...", v.ID, d.ID)
	triggerVolumeActionAndWait(t, "volume-detach", map[string]any{
		"VolumeID":  v.ID,
		"DropletID": float64(d.ID),
	}, v.ID)

	t.Logf("[Get] Volume after detach:")
	detached := callTool[godo.Volume](t, "volume-get", map[string]any{"VolumeID": v.ID})
	require.NotContains(t, detached.DropletIDs, d.ID, "volume should not list droplet after detach")
	t.Logf("      ID: %s", detached.ID)
	t.Logf("      DropletIDs: %v", detached.DropletIDs)

	newSize := float64(defaultVolumeSize + 1)
	t.Logf("[Action] Resizing volume %s to %.0f GiB (region %s)...", v.ID, newSize, v.Region.Slug)
	triggerVolumeActionAndWait(t, "volume-resize", map[string]any{
		"VolumeID":      v.ID,
		"SizeGigaBytes": newSize,
		"Region":        v.Region.Slug,
	}, v.ID)

	t.Logf("[Get] Volume after resize:")
	resized := callTool[godo.Volume](t, "volume-get", map[string]any{"VolumeID": v.ID})
	require.GreaterOrEqual(t, resized.SizeGigaBytes, int64(newSize), "volume size should reflect resize")
	t.Logf("      ID: %s", resized.ID)
	t.Logf("      SizeGigaBytes: %d", resized.SizeGigaBytes)
}
