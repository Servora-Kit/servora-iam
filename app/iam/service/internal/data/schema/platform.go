package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

type Platform struct {
	ent.Schema
}

func (Platform) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(newUUIDv7),
		field.String("slug").MaxLen(64).Unique(),
		field.String("name").MaxLen(128),
		field.String("type").MaxLen(32).Default("system"),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (Platform) Edges() []ent.Edge {
	return nil
}

func (Platform) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "platforms"},
	}
}
