package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/sysadminsmedia/homebox/backend/internal/data/ent/schema/mixins"
)

// Export holds the schema definition for the Export entity. An Export row
// tracks a backup, restore, or inventory CSV job: its lifecycle status and,
// on completion, the blob storage key for the artifact.
type Export struct {
	ent.Schema
}

func (Export) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.BaseMixin{},
		GroupMixin{
			ref:   "exports",
			field: "group_id",
		},
	}
}

func (Export) Fields() []ent.Field {
	return []ent.Field{
		// kind separates ZIP backups, uploaded restore ZIPs, and inventory
		// CSV reports. They share persistence, not workers or download formats.
		field.Enum("kind").
			Values("export", "import", "inventory_csv").
			Default("export"),
		field.Enum("status").
			Values("pending", "running", "completed", "failed").
			Default("pending"),
		field.Int("progress").
			Default(0),
		// artifact_path is the blob key: backups and CSV reports live under
		// {gid}/exports/; restore uploads under {gid}/imports/.
		field.String("artifact_path").
			Optional(),
		field.Int64("size_bytes").
			Default(0),
		field.String("error").
			MaxLen(1000).
			Optional(),
	}
}

func (Export) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("group_id"),
		index.Fields("group_id", "status"),
	}
}
