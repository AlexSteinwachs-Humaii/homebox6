package services

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gocloud.dev/blob"
	"gocloud.dev/gcerrors"
	"gocloud.dev/pubsub"

	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
)

const TopicInventoryCSV = "inventory_csv"

// EnqueueInventoryCSV stores the authorized collection before publishing. The
// message contains only job identity and the server-resolved HB URL, never
// dashboard query state. Delivery inherits the existing pubsub guarantees:
// mem:// messages do not survive restart; lost/interrupted jobs remain visible
// in storage and a retry creates a new job (no unsafe running-job reclamation). The returned snapshot is pending even if a worker has
// already received the message.
func (s *ExportService) EnqueueInventoryCSV(ctx context.Context, gid uuid.UUID, hbURL string) (repo.ExportOut, error) {
	row, err := s.repos.Exports.CreateInventoryCSV(ctx, gid)
	if err != nil {
		return row, err
	}
	topic, err := s.openTopic(ctx, TopicInventoryCSV)
	if err == nil {
		err = topic.Send(ctx, &pubsub.Message{Metadata: map[string]string{
			"group_id": gid.String(), "export_id": row.ID.String(), "hb_url": hbURL,
		}})
	}
	if err != nil {
		s.failInventoryCSV(gid, row.ID, "", fmt.Errorf("failed to enqueue: %w", err))
		return row, err
	}
	s.publishMutation(gid)
	return row, nil
}

// RunInventoryCSV is independent of ZIP backup/import workers. Stored scope and
// kind are revalidated before an atomic pending->running claim. Duplicate or
// terminal deliveries do nothing. Data is read during preparation, not from a
// transactionally consistent point-in-time snapshot.
func (s *ExportService) RunInventoryCSV(ctx context.Context, id, gid uuid.UUID, hbURL string) {
	row, err := s.repos.Exports.GetInventoryCSV(ctx, gid, id)
	if err != nil {
		log.Err(err).Msg("CSV job: cannot load job")
		return
	}
	gid = row.GroupID
	claimed, err := s.repos.Exports.ClaimInventoryCSV(ctx, gid, id)
	if err != nil {
		log.Err(err).Msg("CSV job: cannot claim job")
		return
	}
	if !claimed {
		return
	}
	s.publishMutation(gid)
	key, size, err := s.buildInventoryCSV(ctx, id, gid, hbURL)
	if err == nil {
		err = s.repos.Exports.SetCompleted(ctx, gid, id, key, size)
	}
	if err != nil {
		s.failInventoryCSV(gid, id, key, err)
		return
	}
	// Only a successfully persisted completion emits the ready notification.
	s.publishMutation(gid)
}

// Stage percentages represent completed phases, not guessed row counts:
// 10 = preparing inventory, 60 = serializer returned, 80 = local file ready.
func (s *ExportService) csvProgress(ctx context.Context, gid, id uuid.UUID, pct int) error {
	if err := s.repos.Exports.SetProgress(ctx, gid, id, pct); err != nil {
		return err
	}
	s.publishMutation(gid)
	return nil
}

func (s *ExportService) buildInventoryCSV(ctx context.Context, id, gid uuid.UUID, hbURL string) (string, int64, error) {
	if err := s.csvProgress(ctx, gid, id, 10); err != nil {
		return "", 0, err
	}
	rows, err := s.entities.ExportCSV(ctx, gid, hbURL)
	if err != nil {
		return "", 0, fmt.Errorf("prepare inventory: %w", err)
	}
	if err := s.csvProgress(ctx, gid, id, 60); err != nil {
		return "", 0, err
	}
	tmp, err := os.CreateTemp("", "homebox-inventory-*.csv")
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	defer func() { _ = tmp.Close() }()
	writer := csv.NewWriter(tmp)
	if err := writer.WriteAll(rows); err != nil {
		return "", 0, fmt.Errorf("encode CSV: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", 0, fmt.Errorf("close CSV: %w", err)
	}
	file, err := os.Open(tmp.Name())
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = file.Close() }()
	stat, err := file.Stat()
	if err != nil {
		return "", 0, err
	}
	if err := s.csvProgress(ctx, gid, id, 80); err != nil {
		return "", 0, err
	}
	key := fmt.Sprintf("%s/exports/%s.csv", gid, id)
	bucket, err := blob.OpenBucket(ctx, s.repos.Attachments.GetConnString())
	if err != nil {
		return "", 0, fmt.Errorf("open CSV bucket: %w", err)
	}
	defer func() { _ = bucket.Close() }()
	bw, err := bucket.NewWriter(ctx, s.repos.Attachments.GetFullPath(key), &blob.WriterOptions{ContentType: "text/csv; charset=utf-8"})
	if err != nil {
		return key, 0, fmt.Errorf("open CSV upload: %w", err)
	}
	if _, err := io.Copy(bw, file); err != nil {
		_ = bw.Close()
		return key, 0, fmt.Errorf("upload CSV: %w", err)
	}
	if err := bw.Close(); err != nil {
		return key, 0, fmt.Errorf("close CSV upload: %w", err)
	}
	return key, stat.Size(), nil
}

// Use a fresh bounded context so request cancellation/shutdown cannot prevent
// failure persistence or compensating deletion. Keep a cleanup pointer if the
// bucket is unavailable; seven-day cleanup can then retry without orphaning it.
func (s *ExportService) failInventoryCSV(gid, id uuid.UUID, key string, cause error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if key != "" {
		err := s.deleteUpload(ctx, key)
		if err != nil && gcerrors.Code(err) != gcerrors.NotFound {
			cause = fmt.Errorf("%w (artifact cleanup: %v)", cause, err)
			if err := s.repos.Exports.RetainFailedCSVArtifact(ctx, gid, id, key); err != nil {
				log.Err(err).Msg("CSV job: cannot retain artifact cleanup pointer")
			}
		}
	}
	if err := s.repos.Exports.SetFailed(ctx, gid, id, cause.Error()); err != nil {
		log.Err(err).Str("cause", cause.Error()).Msg("CSV job: cannot persist failure")
		return
	}
	s.publishMutation(gid)
}
