package repo

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/export"

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

func TestInventoryCSVClaimAndRetention(t *testing.T) {
	ctx := context.Background()
	old := time.Now().Add(-8 * 24 * time.Hour)
	csv, err := tClient.Export.Create().SetGroupID(tGroup.ID).SetKind(export.KindInventoryCsv).SetCreatedAt(old).Save(ctx)
	require.NoError(t, err)
	defer tRepos.Exports.Delete(ctx, tGroup.ID, csv.ID)
	backup, err := tClient.Export.Create().SetGroupID(tGroup.ID).SetCreatedAt(old).Save(ctx)
	require.NoError(t, err)
	defer tRepos.Exports.Delete(ctx, tGroup.ID, backup.ID)
	claimed, err := tRepos.Exports.ClaimInventoryCSV(ctx, uuid.New(), csv.ID)
	require.NoError(t, err)
	require.False(t, claimed)
	claimed, err = tRepos.Exports.ClaimInventoryCSV(ctx, tGroup.ID, backup.ID)
	require.NoError(t, err)
	require.False(t, claimed)
	stale, err := tRepos.Exports.ListOlderThan(ctx, time.Now().Add(-7*24*time.Hour))
	require.NoError(t, err)
	for _, row := range stale {
		require.NotEqual(t, csv.ID, row.ID)
	}
	var winners atomic.Int32
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Go(func() {
			won, err := tRepos.Exports.ClaimInventoryCSV(ctx, tGroup.ID, csv.ID)
			if won {
				winners.Add(1)
			}
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.EqualValues(t, 1, winners.Load())
	row, err := tRepos.Exports.GetInventoryCSV(ctx, tGroup.ID, csv.ID)
	require.NoError(t, err)
	require.Equal(t, "running", row.Status)
	stale, err = tRepos.Exports.ListOlderThan(ctx, time.Now().Add(-7*24*time.Hour))
	require.NoError(t, err)
	for _, row := range stale {
		require.NotEqual(t, csv.ID, row.ID)
	}
	// Terminal CSV jobs join the existing seven-day sweep just like ZIPs.
	for _, status := range []export.Status{export.StatusFailed, export.StatusCompleted} {
		require.NoError(t, tClient.Export.UpdateOneID(csv.ID).SetStatus(status).Exec(ctx))
		stale, err = tRepos.Exports.ListOlderThan(ctx, time.Now().Add(-7*24*time.Hour))
		require.NoError(t, err)
		ids := make([]uuid.UUID, len(stale))
		for i, row := range stale {
			ids[i] = row.ID
		}
		require.Contains(t, ids, csv.ID)
		require.Contains(t, ids, backup.ID)
		claimed, err := tRepos.Exports.ClaimInventoryCSV(ctx, tGroup.ID, csv.ID)
		require.NoError(t, err)
		require.False(t, claimed)
	}
}
