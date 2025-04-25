package models

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"time"

	"example.com/myapp/utils/helper"
	"github.com/lib/pq"
	"gopkg.in/guregu/null.v4"
	"gorm.io/gorm"
)

type User struct {
	ID   int64  `json:"id" gorm:"column:id;primaryKey"`
	Type string `json:"type" gorm:"column:type"`
}

type UserBrief struct {
	User
	Name              string        `json:"name" gorm:"column:name"`
	Gender            null.String   `json:"gender,omitempty" gorm:"column:gender"`
	Avatar            null.String   `json:"avatar,omitempty" gorm:"column:avatar"`
	PasswdHash        []byte        `json:"-" gorm:"column:password_hash"`
	CreatedAt         time.Time     `json:"created_at" gorm:"column:created_at"`
	PurchasingCourses pq.Int64Array `json:"purchasing_courses" gorm:"column:purchasing_courses;type:integer[]"`
	PurchasedCourses  pq.Int64Array `json:"purchased_courses" gorm:"column:purchased_courses;type:integer[]"`
}

type UserProfile struct {
	UserBrief
	Email           null.String    `json:"email,omitempty" gorm:"column:email"`
	PhoneNumber     null.String    `json:"phone_number,omitempty" gorm:"column:phone_number"`
	WechatId        null.String    `json:"wechat_id,omitempty" gorm:"column:wechat_id"`
	Region          null.String    `json:"region,omitempty" gorm:"column:region"`
	RegionCodeArray pq.StringArray `json:"region_code_array,omitempty" gorm:"column:region_code_array;type:integer[]"`
	Introduction    null.String    `json:"introduction,omitempty" gorm:"column:introduction"`
	Profession      null.String    `json:"profession,omitempty" gorm:"column:profession"`
	UpdatedAt       time.Time      `json:"updated_at" gorm:"column:updated_at"`
}

type UserModel struct {
	DB     *gorm.DB
	helper *helper.Helper
}

var AnonymousUser = &User{}

func (u *User) IsAnonymous() bool {
	return u == AnonymousUser
}

// GetUserBriefByAccountOrID 根据账号或ID获取用户信息
func (m *UserModel) GetUserBriefByAccountOrID(account string, id int64) (*UserBrief, error) {
	query := `
		SELECT
			id, name, type, gender, avatar, password_hash, created_at,
			(
				SELECT ARRAY_AGG(course_id) FROM business.shopping_carts
				WHERE user_id = users.id
			) AS purchasing_courses,
			(
				SELECT ARRAY_AGG(course_id) FROM business.orders
				WHERE user_id = users.id
			) AS purchased_courses
		FROM appuser.users
		WHERE email = $1 OR phone_number = $1 OR id = $2`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var user UserBrief
	tx := m.DB.WithContext(ctx).Raw(query, account, id).First(&user)

	if tx.Error != nil {
		switch {
		case errors.Is(tx.Error, sql.ErrNoRows):
			return nil, helper.ErrRecordNotFound
		default:
			return nil, tx.Error
		}
	}

	return &user, nil
}

// GetUserForToken 根据令牌获取用户信息
func (m *UserModel) GetUserForToken(tokenScope, tokenPlaintext string) (*User, error) {
	// Calculate the SHA-256 hash of the plaintext token provided by the client.
	// Remember that this returns a byte *array* with length 32, not a slice.
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))

	// Set up the SQL query.
	query := `
		SELECT
			u.id as id,
			u.type as type
		FROM appuser.users u
		INNER JOIN appuser.tokens t
		ON u.id = t.user_id
		WHERE t.hash = $1
		AND t.scope = $2
		AND t.expiry > $3`

	// Create a slice containing the query arguments. Notice how we use the [:] operator
	// to get a slice containing the token hash, rather than passing in the array (which
	// is not supported by the pq driver), and that we pass the current time as the
	// value to check against the token expiry.
	args := []any{tokenHash[:], tokenScope, time.Now()}

	// Create a context with a 3-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Execute the query and scan the result into the User struct. If no matching record
	// is found, we return an ErrRecordNotFound error.
	var user User
	tx := m.DB.WithContext(ctx).Raw(query, args...).Scan(&user)
	if tx.Error != nil {
		switch {
		case errors.Is(tx.Error, sql.ErrNoRows):
			return nil, helper.ErrRecordNotFound
		default:
			return nil, tx.Error
		}
	}
	return &user, nil
}

// 通过ID查询Avatar
func (m *UserModel) GetUserAvatarById(id int64) (*string, error) {
	query := `SELECT avatar FROM appuser.users WHERE id = $1`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var avatar null.String
	tx := m.DB.WithContext(ctx).Raw(query, id).Scan(&avatar)

	if tx.Error != nil {
		return nil, tx.Error
	}

	if avatar.Valid {
		return &avatar.String, nil
	} else {
		return nil, nil
	}
}

func (m *UserModel) GetUserProfileByID(id int64) (*UserProfile, error) {
	query := `
		SELECT
			u.id as id, u.name as name, u.email as email, u.phone_number as phone_number,
			u.wechat_id as wechat_id, u.type as type, u.gender as gender,
			region_query.region_name as region, region_query.region_code_array as region_code_array,
			u.avatar as avatar, u.introduction as introduction, p.title as profession,
			u.created_at as created_at, u.updated_at as updated_at, u.password_hash as password_hash,
			(
				SELECT ARRAY_AGG(course_id) FROM business.shopping_carts
				WHERE user_id = u.id
			) AS purchasing_courses,
			(
				SELECT ARRAY_AGG(course_id) FROM business.orders
				WHERE user_id = u.id
			) AS purchased_courses
		FROM appuser.users u
		LEFT JOIN appuser.professions p ON u.profession_id = p.id
		LEFT JOIN LATERAL china_administrative.get_region_query(u.region) as region_query ON TRUE
		WHERE u.id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var user UserProfile
	tx := m.DB.WithContext(ctx).Raw(query, id).Scan(&user)

	if tx.Error != nil {
		switch {
		case errors.Is(tx.Error, sql.ErrNoRows):
			return nil, helper.ErrRecordNotFound
		default:
			return nil, tx.Error
		}
	}

	return &user, nil
}

func (m *UserModel) UpdateUserAvatar(id int64, avatar string) (string, error) {
	query := `
		WITH old_values AS (
			SELECT avatar FROM appuser.users WHERE id = $1
		)
		UPDATE appuser.users
		SET avatar = $2
		WHERE id = $1
		RETURNING (SELECT avatar FROM old_values)`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var oldAvatar sql.NullString
	tx := m.DB.WithContext(ctx).Raw(query, id, avatar).Scan(&oldAvatar)
	if tx.Error != nil {
		return "", tx.Error
	} else {
		return oldAvatar.String, nil
	}
}
