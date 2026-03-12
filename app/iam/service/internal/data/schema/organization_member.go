package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

type OrganizationMember struct {
	ent.Schema
}

func (OrganizationMember) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(newUUIDv7),
		field.UUID("organization_id", uuid.UUID{}),
		field.UUID("user_id", uuid.UUID{}),
		field.String("role").MaxLen(32).Default("member"),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (OrganizationMember) Edges() []ent.Edge {
	return nil
}

func (OrganizationMember) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "user_id").Unique(),
	}
}

func (OrganizationMember) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "organization_members"},
	}
}
