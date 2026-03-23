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

type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(newUUIDv7),
		// 账户字段（可查询/索引）
		field.String("username").MaxLen(64).Unique(),
		field.String("email").MaxLen(128).Unique(),
		field.String("password").MaxLen(255),
		field.String("phone").MaxLen(32).Optional(),
		field.Bool("phone_verified").Default(false),
		field.String("role").MaxLen(32).Default("user"),
		field.String("status").MaxLen(32).Default("active"),
		field.Bool("email_verified").Default(false),
		field.Time("email_verified_at").Optional().Nillable(),
		// 个人资料（JSON，对应 OIDC UserInfoProfile）
		field.JSON("profile", map[string]interface{}{}).Optional(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		entmixin.SoftDeleteMixin{},
	}
}

func (User) Edges() []ent.Edge {
	return nil
}

func (User) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "users"},
	}
}
