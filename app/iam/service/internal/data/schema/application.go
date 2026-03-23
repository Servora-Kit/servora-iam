package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	entmixin "github.com/Servora-Kit/servora/pkg/db/ent/mixin"
	"github.com/google/uuid"
)

type Application struct {
	ent.Schema
}

func (Application) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(newUUIDv7),
		field.String("client_id").MaxLen(128).Unique(),
		field.String("client_secret_hash").MaxLen(255),
		field.String("name").MaxLen(128),
		field.JSON("redirect_uris", []string{}),
		field.JSON("scopes", []string{}),
		field.JSON("grant_types", []string{}),
		field.String("application_type").MaxLen(32).Default("web"),
		field.String("access_token_type").MaxLen(32).Default("jwt"),
		// type 区分应用用途：web | native | m2m
		field.String("type").MaxLen(32).Default("web"),
		field.Int("id_token_lifetime").Default(3600),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Application) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entmixin.SoftDeleteMixin{},
	}
}

func (Application) Edges() []ent.Edge {
	return nil
}

func (Application) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "applications"},
	}
}
