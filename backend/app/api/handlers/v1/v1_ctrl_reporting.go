package v1

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/hay-kot/httpkit/errchain"
	"github.com/sysadminsmedia/homebox/backend/internal/core/services"
	"github.com/sysadminsmedia/homebox/backend/internal/data/repo"
	"github.com/sysadminsmedia/homebox/backend/internal/sys/validate"
	"github.com/sysadminsmedia/homebox/backend/internal/web/adapters"
)

// inventoryExportContext requires the authenticated selected collection, never
// a group ID from the request body. Routes also run userMW (including mwTenant).
func inventoryExportContext(r *http.Request) (services.Context, error) {
	ctx := services.NewContext(r.Context())
	user := services.UseUserCtx(r.Context())
	if user == nil {
		return ctx, validate.NewRequestError(errors.New("authentication required"), http.StatusUnauthorized)
	}
	for _, gid := range user.GroupIDs {
		if gid == ctx.GID && gid != uuid.Nil {
			return ctx, nil
		}
	}
	return ctx, validate.NewRequestError(errors.New("collection access required"), http.StatusForbidden)
}

// HandleInventoryExportsCreate godoc
//
// @Summary Create a pending inventory CSV job
// @Description Persists a job for the selected collection. CSV worker publication is added separately; this never invokes the ZIP worker.
// @Tags Reporting
// @Produce json
// @Success 202 {object} repo.ExportOut
// @Router /v1/reporting/inventory-exports [POST]
// @Security Bearer
func (ctrl *V1Controller) HandleInventoryExportsCreate() errchain.HandlerFunc {
	return adapters.Command(func(r *http.Request) (repo.ExportOut, error) {
		ctx, err := inventoryExportContext(r)
		if err != nil {
			return repo.ExportOut{}, err
		}
		// Story 2 replaces persistence-only creation with the CSV enqueue service.
		return ctrl.repo.Exports.CreateInventoryCSV(ctx, ctx.GID)
	}, http.StatusAccepted)
}

// HandleInventoryExportsList godoc
//
// @Summary List inventory CSV jobs for the selected collection
// @Tags Reporting
// @Produce json
// @Success 200 {object} Results[repo.ExportOut]
// @Router /v1/reporting/inventory-exports [GET]
// @Security Bearer
func (ctrl *V1Controller) HandleInventoryExportsList() errchain.HandlerFunc {
	return adapters.Command(func(r *http.Request) (Results[repo.ExportOut], error) {
		ctx, err := inventoryExportContext(r)
		if err != nil {
			return Results[repo.ExportOut]{}, err
		}
		rows, err := ctrl.repo.Exports.ListInventoryCSV(ctx, ctx.GID)
		return WrapResults(rows), err
	}, http.StatusOK)
}

// HandleInventoryExportGet godoc
//
// @Summary Get an inventory CSV job in the selected collection
// @Tags Reporting
// @Produce json
// @Param id path string true "Inventory CSV job ID"
// @Success 200 {object} repo.ExportOut
// @Router /v1/reporting/inventory-exports/{id} [GET]
// @Security Bearer
func (ctrl *V1Controller) HandleInventoryExportGet() errchain.HandlerFunc {
	return adapters.CommandID("id", func(r *http.Request, id uuid.UUID) (repo.ExportOut, error) {
		ctx, err := inventoryExportContext(r)
		if err != nil {
			return repo.ExportOut{}, err
		}
		return ctrl.repo.Exports.GetInventoryCSV(ctx, ctx.GID, id)
	}, http.StatusOK)
}

// HandleBillOfMaterialsExport godoc
//
//	@Summary	Export Bill of Materials
//	@Tags		Reporting
//	@Produce	json
//	@Success	200	{string}	string	"text/csv"
//	@Router		/v1/reporting/bill-of-materials [GET]
//	@Security	Bearer
func (ctrl *V1Controller) HandleBillOfMaterialsExport() errchain.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		tenant := services.UseTenantCtx(r.Context())

		if tenant == uuid.Nil {
			return validate.NewRequestError(errors.New("tenant required"), http.StatusBadRequest)
		}

		csv, err := ctrl.svc.Entities.ExportBillOfMaterialsCSV(r.Context(), tenant)
		if err != nil {
			return err
		}

		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=bill-of-materials.csv")
		_, err = w.Write(csv)
		return err
	}
}
