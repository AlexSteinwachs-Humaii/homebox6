package repo

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestInventoryOverviewQuery(t *testing.T) {
	ctx := context.Background()
	g, err := tRepos.Groups.GroupCreate(ctx, "overview", uuid.Nil)
	require.NoError(t, err)
	locType, err := tRepos.EntityTypes.GetDefault(ctx, g.ID, true)
	require.NoError(t, err)
	itemType, err := tRepos.EntityTypes.GetDefault(ctx, g.ID, false)
	require.NoError(t, err)
	create := func(name string, parent uuid.UUID, isLocation bool) EntityOut {
		et := itemType.ID
		if isLocation {
			et = locType.ID
		}
		out, err := tRepos.Entities.Create(ctx, g.ID, EntityCreate{Name: name, ParentID: parent, EntityTypeID: et, Quantity: 2})
		require.NoError(t, err)
		return out
	}
	garage := create("Garage", uuid.Nil, true)
	box := create("Toolbox", garage.ID, false)
	drill := create("Drill A", box.ID, false)
	create("Drill B", box.ID, false)
	create("Drill outside", uuid.Nil, false)
	q := EntityQuery{Search: "Drill", ParentIDs: []uuid.UUID{garage.ID}, IncludeDescendants: true, IncludeLocation: true, Page: 1, PageSize: 1}
	out, err := tRepos.Entities.QueryByGroup(ctx, g.ID, q)
	require.NoError(t, err)
	require.Equal(t, 2, out.Total)
	require.Len(t, out.Items, 1)
	require.Equal(t, drill.ID, out.Items[0].ID)
	require.NotNil(t, out.Items[0].Location)
	require.Equal(t, garage.ID, out.Items[0].Location.ID)
	require.Equal(t, box.ID, out.Items[0].Parent.ID)
	q.Page = 2
	out, err = tRepos.Entities.QueryByGroup(ctx, g.ID, q)
	require.NoError(t, err)
	require.Equal(t, 2, out.Total)
	require.Len(t, out.Items, 1)
	require.Equal(t, "Drill B", out.Items[0].Name)
	require.Equal(t, garage.ID, out.Items[0].Location.ID)
	// The pre-existing direct-parent API stays direct unless explicitly requested.
	q.IncludeDescendants = false
	out, err = tRepos.Entities.QueryByGroup(ctx, g.ID, q)
	require.NoError(t, err)
	require.Zero(t, out.Total)
	// Foreign parent IDs cannot cause another collection's inventory to be returned.
	q.IncludeDescendants = true
	out, err = tRepos.Entities.QueryByGroup(ctx, tGroup.ID, q)
	require.NoError(t, err)
	require.Zero(t, out.Total)
	q.ParentIDs = nil
	q.Page = 1
	q.PageSize = 20
	out, err = tRepos.Entities.QueryByGroup(ctx, g.ID, q)
	require.NoError(t, err)
	require.Equal(t, 3, out.Total)
	for _, item := range out.Items {
		if item.Name == "Drill outside" {
			require.Nil(t, item.Location)
		}
	}
}
