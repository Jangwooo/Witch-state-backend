package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/witchs-lounge_backend/ent/schema/mixin"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

// Mixin of the User.
func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.GlobalMixin{},
	}
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("platform_type").Values("steam", "stove").
			Comment("플랫폼 타입"),
		field.Text("platform_user_id").
			Comment("플랫폼에서의 유저 ID"),
		field.Text("platform_email").Optional().
			Comment("플랫폼 이메일"),
		field.Text("platform_avatar_url").Optional().
			Comment("플랫폼 프로필 이미지 URL"),
		field.Text("platform_display_name").Optional().
			Comment("플랫폼에서 표시되는 이름"),
		field.Text("language").Default("ko").
			Comment("선호 언어"),
		field.JSON("platform_data", map[string]interface{}{}).Optional().
			Comment("플랫폼별 추가 데이터"),
		field.Bool("is_verified").Default(false).
			Comment("플랫폼 인증 여부"),

		field.Text("nickname").
			Comment("게임 내 닉네임"),
		field.Text("display_name").Optional().
			Comment("게임 내 표시 이름"),
		field.Time("last_login_at").Default(time.Now).
			Comment("마지막 로그인 시간"),
		field.Int("level").Default(1).
			Comment("유저 레벨"),
		field.Int("exp").Default(0).
			Comment("경험치"),
		field.Int("coin").Default(0).
			Comment("게임 내 코인"),
		field.Int("gem").Default(0).
			Comment("프리미엄 재화"),
		field.JSON("settings", map[string]interface{}{}).Default(map[string]interface{}{}).
			Comment("게임 설정"),
		field.JSON("customize_data", map[string]interface{}{}).Default(map[string]interface{}{}).
			Comment("커스터마이징 데이터"),
		field.JSON("save_data", map[string]interface{}{}).Default(map[string]interface{}{}).
			Comment("게임 저장 데이터"),
		field.Bool("is_banned").Default(false).
			Comment("밴 여부"),
		field.Time("banned_until").Optional().Nillable().
			Comment("밴 해제 시간"),
		field.Text("ban_reason").Optional().
			Comment("밴 사유"),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("purchased_products", Product.Type).
			Through("user_purchases", UserPurchase.Type),
		edge.To("records", Record.Type),
		edge.To("user_achievements", UserAchievement.Type),
	}
}

// Indexes of the User.
func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("platform_type", "platform_user_id").Unique(),
		index.Fields("level"),
	}
}
