package v1

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hay-kot/httpkit/errchain"
	"github.com/stretchr/testify/require"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services/reporting/eventbus"
	"github.com/sysadminsmedia/homebox/backend/internal/data/ent"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/validate"
	_ "github.com/sysadminsmedia/homebox/backend/pkgs/cgofreesqlite"
)

func TestInventoryExportHandlers(t *testing.T) {
	ctx := context.Background()
	db, err := ent.Open("sqlite3", "file:"+uuid.NewString()+"?mode=memory&cache=shared&_fk=1&_time_format=sqlite")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, db.Schema.Create(ctx))
	repos := repo.New(db, eventbus.New(), config.Storage{}, "mem://{{ .Topic }}", config.Thumbnail{})
	group, err := repos.Groups.GroupCreate(ctx, "selected", uuid.Nil)
	require.NoError(t, err)
	other, err := repos.Groups.GroupCreate(ctx, "other", uuid.Nil)
	require.NoError(t, err)
	user := &repo.UserOut{ID: uuid.New(), DefaultGroupID: group.ID, GroupIDs: []uuid.UUID{group.ID, other.ID}}
	svc := services.New(repos, services.WithExportPlumbing(eventbus.New(), db, config.Storage{}, "mem://{{ .Topic }}-"+uuid.NewString(), "sqlite3"))
	t.Cleanup(func() { _ = svc.Exports.Shutdown(context.Background()) })
	ctrl := &V1Controller{repo: repos, svc: svc, config: &config.Config{}}
	authenticated := services.SetTenantCtx(services.SetUserCtx(ctx, user, "token"), group.ID)
	call := func(handler errchain.HandlerFunc, c context.Context, id uuid.UUID) (*httptest.ResponseRecorder, error) {
		req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(c)
		route := chi.NewRouteContext()
		route.URLParams.Add("id", id.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, route))
		rr := httptest.NewRecorder()
		err := handler(rr, req)
		return rr, err
	}
	rr, err := call(ctrl.HandleInventoryExportsCreate(), authenticated, uuid.Nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, rr.Code)
	var csv repo.ExportOut
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &csv))
	require.Equal(t, group.ID, csv.GroupID)
	require.Equal(t, "inventory_csv", csv.Kind)
	require.Equal(t, "pending", csv.Status)
	rr, err = call(ctrl.HandleInventoryExportGet(), authenticated, csv.ID)
	require.NoError(t, err)
	require.Contains(t, rr.Body.String(), csv.ID.String())
	rr, err = call(ctrl.HandleInventoryExportsList(), authenticated, uuid.Nil)
	require.NoError(t, err)
	require.Contains(t, rr.Body.String(), csv.ID.String())
	rr, err = call(ctrl.HandleExportsList(), authenticated, uuid.Nil)
	require.NoError(t, err)
	require.NotContains(t, rr.Body.String(), csv.ID.String())
	backup, err := repos.Exports.Create(ctx, group.ID)
	require.NoError(t, err)
	rr, err = call(ctrl.HandleInventoryExportGet(), authenticated, backup.ID)
	require.True(t, ent.IsNotFound(err))
	require.Empty(t, rr.Body.String())
	rr, err = call(ctrl.HandleExportGet(), authenticated, csv.ID)
	require.True(t, ent.IsNotFound(err))
	require.Empty(t, rr.Body.String())
	// Even a member of both collections must select the owning collection.
	otherContext := services.SetTenantCtx(authenticated, other.ID)
	rr, err = call(ctrl.HandleInventoryExportGet(), otherContext, csv.ID)
	require.True(t, ent.IsNotFound(err))
	require.Empty(t, rr.Body.String())
	rr, err = call(ctrl.HandleInventoryExportsList(), otherContext, uuid.Nil)
	require.NoError(t, err)
	require.NotContains(t, rr.Body.String(), csv.ID.String())
	for _, h := range []errchain.HandlerFunc{ctrl.HandleInventoryExportsCreate(), ctrl.HandleInventoryExportsList(), ctrl.HandleInventoryExportGet()} {
		for _, tc := range []struct {
			ctx    context.Context
			status int
		}{
			{ctx, http.StatusUnauthorized},
			{services.SetTenantCtx(authenticated, uuid.New()), http.StatusForbidden},
		} {
			rr, err := call(h, tc.ctx, csv.ID)
			var re *validate.RequestError
			require.True(t, errors.As(err, &re))
			require.Equal(t, tc.status, re.Status)
			require.Empty(t, rr.Body.String())
		}
	}
	rows, err := repos.Exports.ListInventoryCSV(ctx, group.ID)
	require.NoError(t, err)
	require.Len(t, rows, 1) // rejected creation attempts persisted nothing
}
