package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gocloud.dev/blob"

	"github.com/sysadminsmedia/homebox/backend/internal/core/services/reporting/eventbus"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/export"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
)

func TestPurgeInventoryCSVArtifacts(t *testing.T) {
	ctx := context.Background()
	db, err := ent.Open("sqlite3", "file:"+uuid.NewString()+"?mode=memory&cache=shared&_fk=1&_time_format=sqlite")
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, db.Schema.Create(ctx))
	storage := config.Storage{ConnString: "file://" + t.TempDir(), PrefixPath: "/"}
	repos := repo.New(db, eventbus.New(), storage, "mem://{{ .Topic }}", config.Thumbnail{})
	g, err := repos.Groups.GroupCreate(ctx, "purge", uuid.Nil)
	require.NoError(t, err)
	bucket, err := blob.OpenBucket(ctx, repos.Attachments.GetConnString())
	require.NoError(t, err)
	defer bucket.Close()
	var terminalIDs, activeIDs []uuid.UUID
	for _, status := range []export.Status{export.StatusPending, export.StatusRunning, export.StatusCompleted, export.StatusFailed} {
		id := uuid.New()
		key := fmt.Sprintf("%s/exports/%s.csv", g.ID, id)
		_, err := db.Export.Create().SetID(id).SetGroupID(g.ID).SetKind(export.KindInventoryCsv).
			SetCreatedAt(time.Now().Add(-8 * 24 * time.Hour)).SetStatus(status).SetArtifactPath(key).Save(ctx)
		require.NoError(t, err)
		require.NoError(t, bucket.WriteAll(ctx, repos.Attachments.GetFullPath(key), []byte("HB.name\n"), nil))
		if status == export.StatusPending || status == export.StatusRunning {
			activeIDs = append(activeIDs, id)
		} else {
			terminalIDs = append(terminalIDs, id)
		}
	}
	purgeStaleExports(ctx, &app{repos: repos})
	for _, id := range terminalIDs {
		_, err := repos.Exports.GetInventoryCSV(ctx, g.ID, id)
		require.True(t, ent.IsNotFound(err))
		exists, err := bucket.Exists(ctx, repos.Attachments.GetFullPath(fmt.Sprintf("%s/exports/%s.csv", g.ID, id)))
		require.NoError(t, err)
		require.False(t, exists)
	}
	for _, id := range activeIDs {
		row, err := repos.Exports.GetInventoryCSV(ctx, g.ID, id)
		require.NoError(t, err)
		exists, err := bucket.Exists(ctx, repos.Attachments.GetFullPath(row.ArtifactPath))
		require.NoError(t, err)
		require.True(t, exists)
	}
}
