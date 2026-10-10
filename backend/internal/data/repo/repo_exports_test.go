package repo

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
)

func TestInventoryCSVJobs(t *testing.T) {
	ctx := context.Background()
	gid := tGroup.ID
	other, err := tRepos.Groups.GroupCreate(ctx, "csv-other", uuid.Nil)
	require.NoError(t, err)
	csv, err := tRepos.Exports.CreateInventoryCSV(ctx, gid)
	require.NoError(t, err)
	backup, err := tRepos.Exports.Create(ctx, gid)
	require.NoError(t, err)
	restore, err := tRepos.Exports.CreateImport(ctx, gid, gid.String()+"/imports/input.zip", 12)
	require.NoError(t, err)
	require.Equal(t, "inventory_csv", csv.Kind)
	require.Equal(t, "pending", csv.Status)
	require.Zero(t, csv.Progress)
	for _, job := range []ExportOut{csv, backup, restore} {
		defer func(id uuid.UUID) { _, _ = tRepos.Exports.Delete(ctx, gid, id) }(job.ID)
	}
	rows, err := tRepos.Exports.ListInventoryCSV(ctx, gid)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, csv.ID, rows[0].ID)
	rows, err = tRepos.Exports.ListByGroup(ctx, gid)
	require.NoError(t, err)
	ids := make([]uuid.UUID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	require.Contains(t, ids, backup.ID)
	require.Contains(t, ids, restore.ID)
	for _, row := range rows {
		require.NotEqual(t, "inventory_csv", row.Kind)
	}
	_, err = tRepos.Exports.GetInventoryCSV(ctx, other.ID, csv.ID)
	require.True(t, ent.IsNotFound(err))
	_, err = tRepos.Exports.GetInventoryCSV(ctx, gid, backup.ID)
	require.True(t, ent.IsNotFound(err))
	_, err = tRepos.Exports.GetBackup(ctx, gid, csv.ID)
	require.True(t, ent.IsNotFound(err))
	require.Error(t, tRepos.Exports.SetRunning(ctx, other.ID, csv.ID))
	require.NoError(t, tRepos.Exports.SetRunning(ctx, gid, csv.ID))
	require.NoError(t, tRepos.Exports.SetProgress(ctx, gid, csv.ID, 45))
	csv, err = tRepos.Exports.GetInventoryCSV(ctx, gid, csv.ID)
	require.NoError(t, err)
	require.Equal(t, "running", csv.Status)
	require.Equal(t, 45, csv.Progress)
	key := gid.String() + "/exports/" + csv.ID.String() + ".csv"
	require.NoError(t, tRepos.Exports.SetCompleted(ctx, gid, csv.ID, key, 123))
	csv, err = tRepos.Exports.GetInventoryCSV(ctx, gid, csv.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", csv.Status)
	require.Equal(t, 100, csv.Progress)
	require.Equal(t, key, csv.ArtifactPath)
	require.EqualValues(t, 123, csv.SizeBytes)
	require.NoError(t, tRepos.Exports.SetFailed(ctx, gid, csv.ID, "test failure"))
	csv, err = tRepos.Exports.GetInventoryCSV(ctx, gid, csv.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", csv.Status)
	require.Equal(t, "test failure", csv.Error)
	rows, err = tRepos.Exports.ListInventoryCSV(ctx, other.ID)
	require.NoError(t, err)
	require.Empty(t, rows)
}
