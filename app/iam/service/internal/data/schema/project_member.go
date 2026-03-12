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

type ProjectMember struct {
	ent.Schema
}

func (ProjectMember) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(newUUIDv7),
		field.UUID("project_id", uuid.UUID{}),
		field.UUID("user_id", uuid.UUID{}),
		field.String("role").MaxLen(32).Default("viewer"),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (ProjectMember) Edges() []ent.Edge {
	return nil
}

func (ProjectMember) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id", "user_id").Unique(),
	}
}

func (ProjectMember) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "project_members"},
	}
}
