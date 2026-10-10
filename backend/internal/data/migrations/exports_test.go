package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	_ "github.com/sysadminsmedia/homebox/backend/pkgs/cgofreesqlite"
)

// Test the deployed SQL, not Ent's fresh-schema generator. PostgreSQL runs in
// an isolated schema when HBOX_TEST_POSTGRES_DSN points at a test database.
func TestInventoryCSVUpgrade(t *testing.T) {
	for _, dialect := range []string{"sqlite3", "postgres"} {
		t.Run(dialect, func(t *testing.T) {
			driver, dsn := "sqlite3", "file:"+uuid.NewString()+"?mode=memory&cache=shared&_fk=1&_time_format=sqlite"
			if dialect == "postgres" {
				driver, dsn = "pgx", os.Getenv("HBOX_TEST_POSTGRES_DSN")
				if dsn == "" {
					t.Skip("HBOX_TEST_POSTGRES_DSN not set")
				}
			}
			db, err := sql.Open(driver, dsn)
			require.NoError(t, err)
			defer db.Close()
			db.SetMaxOpenConns(1)
			ctx := context.Background()
			if dialect == "postgres" {
				schema := "csv_upgrade_" + strings.ReplaceAll(uuid.NewString(), "-", "")
				_, err = db.Exec("CREATE SCHEMA " + schema)
				require.NoError(t, err)
				defer db.Exec("DROP SCHEMA " + schema + " CASCADE")
				_, err = db.Exec("SET search_path TO " + schema)
				require.NoError(t, err)
			}
			// The exports migrations only depend on groups' primary key. Keep this
			// fixture minimal so this tests exactly the old->new export schema.
			_, err = db.Exec("CREATE TABLE groups (id uuid PRIMARY KEY)")
			require.NoError(t, err)
			files, err := Migrations(dialect)
			require.NoError(t, err)
			oldName := "20260512130000_add_exports.sql"
			if dialect == "postgres" {
				oldName = "20260512130001_add_exports.sql"
			}
			old, err := files.ReadFile(dialect + "/" + oldName)
			require.NoError(t, err)
			_, err = db.Exec(string(old))
			require.NoError(t, err)
			gid := uuid.NewString()
			_, err = db.Exec(fmt.Sprintf("INSERT INTO groups (id) VALUES ('%s')", gid))
			require.NoError(t, err)
			ids := []string{}
			for _, kind := range []string{"export", "import"} {
				for _, status := range []string{"pending", "running", "completed", "failed"} {
					id := uuid.NewString()
					ids = append(ids, id)
					_, err = db.Exec(fmt.Sprintf(`INSERT INTO exports (id, created_at, updated_at, kind, status, progress, artifact_path, size_bytes, error, group_id)
      VALUES ('%s', '2026-05-12 13:00:00+00:00', '2026-05-12 14:00:00+00:00', '%s', '%s', 37, 'fixture.zip', 123, 'fixture', '%s')`, id, kind, status, gid))
					require.NoError(t, err)
				}
			}
			snapshot := func() []string {
				rows, err := db.Query("SELECT id, created_at, updated_at, kind, status, progress, artifact_path, size_bytes, error, group_id FROM exports ORDER BY id")
				require.NoError(t, err)
				defer rows.Close()
				out := []string{}
				for rows.Next() {
					var vals [10]interface{}
					dest := make([]interface{}, 10)
					for i := range vals {
						dest[i] = &vals[i]
					}
					require.NoError(t, rows.Scan(dest...))
					out = append(out, fmt.Sprint(vals))
				}
				require.NoError(t, rows.Err())
				return out
			}
			before := snapshot()
			// Goose runs the actual embedded forward migration transactionally.
			migration, err := files.ReadFile(dialect + "/20261010000000_add_inventory_csv_jobs.sql")
			require.NoError(t, err)
			provider, err := goose.NewProvider(goose.Dialect(dialect), db, fstest.MapFS{
				"20261010000000_add_inventory_csv_jobs.sql": &fstest.MapFile{Data: migration},
			})
			require.NoError(t, err)
			_, err = provider.Up(ctx)
			require.NoError(t, err)
			require.Equal(t, before, snapshot())
			_, err = db.Exec(fmt.Sprintf(`INSERT INTO exports (id, created_at, updated_at, kind, group_id)
    VALUES ('%s', '2026-05-12 13:00:00+00:00', '2026-05-12 13:00:00+00:00', 'inventory_csv', '%s')`, uuid.NewString(), gid))
			require.NoError(t, err)
			var count int
			indexQuery := "SELECT count(*) FROM sqlite_master WHERE type='index' AND name IN ('export_group_id', 'export_group_id_status')"
			if dialect == "postgres" {
				indexQuery = "SELECT count(*) FROM pg_indexes WHERE schemaname=current_schema() AND indexname IN ('export_group_id', 'export_group_id_status')"
			}
			require.NoError(t, db.QueryRow(indexQuery).Scan(&count))
			require.Equal(t, 2, count)
			// CHECK and foreign key/cascade behavior must survive SQLite's table rebuild.
			for _, column := range []string{"kind", "status"} {
				_, err = db.Exec(fmt.Sprintf("UPDATE exports SET %s='invalid' WHERE id='%s'", column, ids[0]))
				require.Error(t, err)
			}
			_, err = db.Exec(fmt.Sprintf("UPDATE exports SET group_id='%s' WHERE id='%s'", uuid.NewString(), ids[0]))
			require.Error(t, err)
			_, err = db.Exec(fmt.Sprintf("DELETE FROM groups WHERE id='%s'", gid))
			require.NoError(t, err)
			require.NoError(t, db.QueryRow("SELECT count(*) FROM exports").Scan(&count))
			require.Zero(t, count)
		})
	}
}
