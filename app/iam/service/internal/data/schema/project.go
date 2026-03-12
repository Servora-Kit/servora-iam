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

func (Project) Edges() []ent.Edge {
	return nil
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
