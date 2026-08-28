package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// OpenAIOAuthAccountUserGrant grants one local user access to one restricted
// OpenAI OAuth root account.
type OpenAIOAuthAccountUserGrant struct {
	ent.Schema
}

func (OpenAIOAuthAccountUserGrant) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "openai_oauth_account_user_grants"},
	}
}

func (OpenAIOAuthAccountUserGrant) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("account_id"),
		field.Int64("user_id"),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (OpenAIOAuthAccountUserGrant) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("account", Account.Type).
			Ref("openai_oauth_user_grants").
			Field("account_id").
			Unique().
			Required(),
		edge.From("user", User.Type).
			Ref("openai_oauth_account_grants").
			Field("user_id").
			Unique().
			Required(),
	}
}

func (OpenAIOAuthAccountUserGrant) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("account_id", "user_id").Unique(),
		index.Fields("user_id", "account_id"),
	}
}
