package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	entmixin "github.com/Servora-Kit/servora/pkg/ent/mixin"
	"github.com/google/uuid"
)

type Project struct {
	ent.Schema
}

func (Project) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(newUUIDv7),
		field.UUID("organization_id", uuid.UUID{}),
		field.String("name").MaxLen(128),
		field.String("slug").MaxLen(128),
		field.String("description").MaxLen(512).Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Project) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entmixin.SoftDeleteMixin{},
	}
}

func (Project) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).
			Ref("projects").
			Field("organization_id").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.To("members", ProjectMember.Type),
	}
}

func (Project) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "slug").Unique(),
	}
}

func (Project) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "projects"},
	}
}
