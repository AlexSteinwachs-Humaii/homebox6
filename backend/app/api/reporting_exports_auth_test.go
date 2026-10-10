package main

import (
	"context"
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
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/config"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/validate"
)

func TestInventoryExportRoutesRequireAuthentication(t *testing.T) {
	a := new(&config.Config{})
	a.services = &services.AllServices{User: &services.UserService{}, Entities: &services.EntityService{}}
	a.repos = &repo.AllRepos{}
	a.bus = eventbus.New()
	t.Cleanup(func() { a.notifierTestLimiter.Stop(); a.authLimiter.Stop() })
	// Render the request errors from the production middleware without needing
	// to initialize services: unauthenticated requests must never reach them.
	errorMW := func(next errchain.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := next.ServeHTTP(w, r)
			var requestErr *validate.RequestError
			if errors.As(err, &requestErr) {
				w.WriteHeader(requestErr.Status)
				return
			}
			require.NoError(t, err)
		})
	}
	router := chi.NewRouter()
	a.mountRoutes(router, errchain.New(errorMW), nil)
	for _, tc := range []struct{ method, url string }{
		{http.MethodPost, "/api/v1/reporting/inventory-exports"},
		{http.MethodGet, "/api/v1/reporting/inventory-exports"},
		{http.MethodGet, "/api/v1/reporting/inventory-exports/" + uuid.NewString()},
	} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.url, nil)
		req.Header.Set("X-Tenant", uuid.NewString())
		router.ServeHTTP(rr, req)
		require.Equal(t, http.StatusUnauthorized, rr.Code, tc.url)
		require.Empty(t, rr.Body.String())
	}
}

func TestInventoryExportTenantSelection(t *testing.T) {
	a := &app{}
	defaultGID, selectedGID, forbiddenGID := uuid.New(), uuid.New(), uuid.New()
	user := &repo.UserOut{ID: uuid.New(), DefaultGroupID: defaultGID, GroupIDs: []uuid.UUID{defaultGID, selectedGID}}
	for _, tc := range []struct {
		header, query string
		want          uuid.UUID
		status        int
	}{
		{"", "", defaultGID, 0},
		{selectedGID.String(), "", selectedGID, 0},
		{"", selectedGID.String(), selectedGID, 0},
		{forbiddenGID.String(), "", uuid.Nil, http.StatusForbidden},
		{"", forbiddenGID.String(), uuid.Nil, http.StatusForbidden},
		{forbiddenGID.String(), selectedGID.String(), uuid.Nil, http.StatusForbidden},
		{"invalid", "", uuid.Nil, http.StatusBadRequest},
	} {
		reached := false
		next := errchain.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			reached = true
			require.Equal(t, tc.want, services.NewContext(r.Context()).GID)
			return nil
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/reporting/inventory-exports?tenant="+tc.query, nil)
		req.Header.Set("X-Tenant", tc.header)
		req = req.WithContext(services.SetUserCtx(context.Background(), user, "test-token"))
		err := a.mwTenant(next).ServeHTTP(httptest.NewRecorder(), req)
		if tc.status == 0 {
			require.NoError(t, err)
			require.True(t, reached)
		} else {
			var requestErr *validate.RequestError
			require.True(t, errors.As(err, &requestErr))
			require.Equal(t, tc.status, requestErr.Status)
			require.False(t, reached)
		}
	}
}
