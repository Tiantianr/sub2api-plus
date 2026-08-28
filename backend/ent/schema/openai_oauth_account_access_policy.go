package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// OpenAIOAuthAccountAccessPolicy stores local-user access policy for one
// credential-owning OpenAI OAuth account.
type OpenAIOAuthAccountAccessPolicy struct {
	ent.Schema
}

func (OpenAIOAuthAccountAccessPolicy) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "openai_oauth_account_access_policies"},
	}
}

func (OpenAIOAuthAccountAccessPolicy) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("account_id"),
		field.Enum("mode").Values("public", "restricted").Default("public"),
		field.Bool("default_for_new_users").Default(false),
		field.Int64("revision").Default(1).Positive(),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (OpenAIOAuthAccountAccessPolicy) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("account", Account.Type).
			Ref("openai_oauth_access_policy").
			Field("account_id").
			Unique().
			Required(),
	}
}
