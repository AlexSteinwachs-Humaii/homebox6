package services

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gocloud.dev/blob"
	"gocloud.dev/pubsub"

	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/export"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
)

func csvTestService(t *testing.T, conn string) *ExportService {
	t.Helper()
	svc := New(tRepos, WithExportPlumbing(tbus, tClient, config.Storage{}, conn, "sqlite3")).Exports
	t.Cleanup(func() { _ = svc.Shutdown(context.Background()) })
	return svc
}

func TestInventoryCSVArtifact(t *testing.T) {
	ctx := context.Background()
	src, err := tRepos.Groups.GroupCreate(ctx, "CSV-source-"+uuid.NewString(), uuid.Nil)
	require.NoError(t, err)
	locType, err := tRepos.EntityTypes.GetDefault(ctx, src.ID, true)
	require.NoError(t, err)
	itemType, err := tRepos.EntityTypes.GetDefault(ctx, src.ID, false)
	require.NoError(t, err)
	loc, err := tRepos.Entities.Create(ctx, src.ID, repo.EntityCreate{Name: "Garage", EntityTypeID: locType.ID})
	require.NoError(t, err)
	parent, err := tRepos.Entities.Create(ctx, src.ID, repo.EntityCreate{Name: "Box", EntityTypeID: itemType.ID, ParentID: loc.ID})
	require.NoError(t, err)
	tg, err := tRepos.Tags.Create(ctx, src.ID, repo.TagCreate{Name: "Tools"})
	require.NoError(t, err)
	// More than one dashboard page; only an off-screen archived row has a custom
	// field. No search, location or pagination argument exists on the worker.
	for i := 0; i < 35; i++ {
		item, err := tRepos.Entities.Create(ctx, src.ID, repo.EntityCreate{
			Name: fmt.Sprintf("工具, \"quoted\"\n%d", i), Quantity: 2.5, Description: "café\nsecond line", EntityTypeID: itemType.ID, ParentID: parent.ID, TagIDs: []uuid.UUID{tg.ID},
		})
		require.NoError(t, err)
		if i == 34 {
			require.NoError(t, tClient.Entity.UpdateOneID(item.ID).SetArchived(true).SetNotes("notes, \"雪\"\nline").Exec(ctx))
			_, err := tClient.EntityField.Create().SetEntityID(item.ID).SetName("unseen").SetType("text").SetTextValue("雪,\"\ncustom").Save(ctx)
			require.NoError(t, err)
		}
	}
	url := "https://homebox.example"
	expected, err := tSvc.Entities.ExportCSV(ctx, src.ID, url)
	require.NoError(t, err)
	svc := csvTestService(t, "mem://{{ .Topic }}-"+uuid.NewString())
	row, err := svc.EnqueueInventoryCSV(ctx, src.ID, url)
	require.NoError(t, err)
	require.Equal(t, "pending", row.Status)
	var stagesMu sync.Mutex
	var stages []int
	tClient.Export.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			em := m.(*ent.ExportMutation)
			mid, hasID := em.ID()
			result, err := next.Mutate(ctx, m)
			if hasID && mid == row.ID && err == nil {
				if pct, ok := em.Progress(); ok {
					stagesMu.Lock()
					stages = append(stages, pct)
					stagesMu.Unlock()
				}
			}
			return result, err
		})
	})
	// ZIP backup/import workers must not consume CSV jobs either.
	svc.RunExport(ctx, row.ID, src.ID)
	svc.RunImport(ctx, src.ID, uuid.New(), row.ID)
	// A delivery claiming another collection cannot start this job.
	svc.RunInventoryCSV(ctx, row.ID, tGroup.ID, url)
	stored, err := tRepos.Exports.GetInventoryCSV(ctx, src.ID, row.ID)
	require.NoError(t, err)
	require.Equal(t, "pending", stored.Status)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Go(func() { svc.RunInventoryCSV(ctx, row.ID, src.ID, url) })
	}
	wg.Wait()
	stored, err = tRepos.Exports.GetInventoryCSV(ctx, src.ID, row.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", stored.Status)
	require.Equal(t, 100, stored.Progress)
	require.Equal(t, []int{10, 60, 80, 100}, stages)
	require.Equal(t, fmt.Sprintf("%s/exports/%s.csv", src.ID, row.ID), stored.ArtifactPath)
	bucket, err := blob.OpenBucket(ctx, tRepos.Attachments.GetConnString())
	require.NoError(t, err)
	defer bucket.Close()
	key := tRepos.Attachments.GetFullPath(stored.ArtifactPath)
	defer bucket.Delete(ctx, key)
	data, err := bucket.ReadAll(ctx, key)
	require.NoError(t, err)
	require.True(t, utf8.Valid(data))
	require.EqualValues(t, len(data), stored.SizeBytes)
	actual, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	require.NoError(t, err)
	require.Equal(t, expected[0], actual[0])
	require.ElementsMatch(t, expected[1:], actual[1:]) // serializer DB order isn't contractual
	require.Len(t, actual, 38)                         // header, location, parent, all 35 items
	require.Contains(t, actual[0], "HB.field.unseen")
	require.NotContains(t, actual[0], "HB.purchase_time")
	svc.RunInventoryCSV(ctx, row.ID, src.ID, "https://wrong.example")
	again, err := bucket.ReadAll(ctx, key)
	require.NoError(t, err)
	require.Equal(t, data, again) // terminal delivery never overwrites an artifact
}

func TestInventoryCSVEmptyAndPublication(t *testing.T) {
	ctx := context.Background()
	src, err := tRepos.Groups.GroupCreate(ctx, "CSV-empty-"+uuid.NewString(), uuid.Nil)
	require.NoError(t, err)
	conn := "mem://{{ .Topic }}-" + uuid.NewString()
	svc := csvTestService(t, conn)
	_, err = svc.openTopic(ctx, TopicInventoryCSV)
	require.NoError(t, err)
	// Use the same cached connection as the production subscriber.
	sub, err := pubsub.OpenSubscription(ctx, strings.ReplaceAll(conn, "{{ .Topic }}", TopicInventoryCSV))
	require.NoError(t, err)
	defer sub.Shutdown(ctx)
	row, err := svc.EnqueueInventoryCSV(ctx, src.ID, "https://trusted.example")
	require.NoError(t, err)
	msg, err := sub.Receive(ctx)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"group_id": src.ID.String(), "export_id": row.ID.String(), "hb_url": "https://trusted.example"}, msg.Metadata)
	msg.Ack()
	stored, err := tRepos.Exports.GetInventoryCSV(ctx, src.ID, row.ID)
	require.NoError(t, err)
	require.Equal(t, "pending", stored.Status)
	svc.RunInventoryCSV(ctx, row.ID, src.ID, msg.Metadata["hb_url"])
	stored, err = tRepos.Exports.GetInventoryCSV(ctx, src.ID, row.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", stored.Status)
	bucket, err := blob.OpenBucket(ctx, tRepos.Attachments.GetConnString())
	require.NoError(t, err)
	defer bucket.Close()
	key := tRepos.Attachments.GetFullPath(stored.ArtifactPath)
	defer bucket.Delete(ctx, key)
	data, err := bucket.ReadAll(ctx, key)
	require.NoError(t, err)
	rows, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Contains(t, rows[0], "HB.name")
}

func TestInventoryCSVFailures(t *testing.T) {
	ctx := context.Background()
	t.Run("publication", func(t *testing.T) {
		svc := csvTestService(t, "unsupported://{{ .Topic }}")
		row, err := svc.EnqueueInventoryCSV(ctx, tGroup.ID, "https://homebox.example")
		require.Error(t, err)
		stored, err := tRepos.Exports.GetInventoryCSV(ctx, tGroup.ID, row.ID)
		require.NoError(t, err)
		require.Equal(t, "failed", stored.Status)
		require.Contains(t, stored.Error, "failed to enqueue")
	})
	for _, stage := range []string{"preparation", "upload", "completion"} {
		t.Run(stage, func(t *testing.T) {
			// Isolate injected storage/DB failures from the shared test services.
			db, err := ent.Open("sqlite3", "file:"+uuid.NewString()+"?mode=memory&cache=shared&_fk=1&_time_format=sqlite")
			require.NoError(t, err)
			defer db.Close()
			require.NoError(t, db.Schema.Create(ctx))
			storage := config.Storage{PrefixPath: "/", ConnString: tRepos.Attachments.GetConnString()}
			if stage == "upload" {
				storage.ConnString = "unsupported://bucket"
			}
			repos := repo.New(db, tbus, storage, "mem://{{ .Topic }}", config.Thumbnail{})
			g, err := repos.Groups.GroupCreate(ctx, "failure", uuid.Nil)
			require.NoError(t, err)
			svc := New(repos, WithExportPlumbing(tbus, db, storage, "mem://{{ .Topic }}-"+uuid.NewString(), "sqlite3")).Exports
			defer svc.Shutdown(ctx)
			row, err := svc.EnqueueInventoryCSV(ctx, g.ID, "")
			require.NoError(t, err)
			if stage == "preparation" {
				db.Entity.Intercept(ent.InterceptFunc(func(next ent.Querier) ent.Querier {
					return ent.QuerierFunc(func(ctx context.Context, query ent.Query) (ent.Value, error) {
						return nil, errors.New("injected preparation failure")
					})
				}))
			}
			if stage == "completion" {
				db.Export.Use(func(next ent.Mutator) ent.Mutator {
					return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
						if status, ok := m.(*ent.ExportMutation).Status(); ok && status == export.StatusCompleted {
							return nil, errors.New("injected completion failure")
						}
						return next.Mutate(ctx, m)
					})
				})
			}
			svc.RunInventoryCSV(ctx, row.ID, g.ID, "")
			stored, err := repos.Exports.GetInventoryCSV(ctx, g.ID, row.ID)
			require.NoError(t, err)
			require.Equal(t, "failed", stored.Status)
			require.Less(t, stored.Progress, 100)
			require.NotEmpty(t, stored.Error)
			require.Empty(t, stored.ArtifactPath)
			if stage == "completion" {
				bucket, err := blob.OpenBucket(ctx, storage.ConnString)
				require.NoError(t, err)
				defer bucket.Close()
				exists, err := bucket.Exists(ctx, repos.Attachments.GetFullPath(fmt.Sprintf("%s/exports/%s.csv", g.ID, row.ID)))
				require.NoError(t, err)
				require.False(t, exists)
			}
			// Retry preserves failure history and creates an independent pending job.
			retry, err := svc.EnqueueInventoryCSV(ctx, g.ID, "")
			require.NoError(t, err)
			require.NotEqual(t, row.ID, retry.ID)
			require.Equal(t, "pending", retry.Status)
		})
	}
}
