package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	entmixin "github.com/Servora-Kit/servora/pkg/ent/mixin"
	"github.com/google/uuid"
)

type Organization struct {
	ent.Schema
}

func (Organization) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(newUUIDv7),
		field.UUID("platform_id", uuid.UUID{}),
		field.String("name").MaxLen(128),
		field.String("slug").MaxLen(128).Unique(),
		field.String("display_name").MaxLen(255).Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Organization) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entmixin.SoftDeleteMixin{},
	}
}

func (Organization) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("platform", Platform.Type).
			Ref("organizations").
			Field("platform_id").
			Unique().
			Required(),
		edge.To("members", OrganizationMember.Type),
		edge.To("projects", Project.Type),
	}
}

func (Organization) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "organizations"},
	}
}
