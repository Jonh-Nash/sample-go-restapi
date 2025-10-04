package domain

import (
	"regexp"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	UserID       string
	PasswordHash string
	Nickname     string
	Comment      string
	Deleted      bool
}

var (
	reUserID = regexp.MustCompile(`^[A-Za-z0-9]{6,20}$`)
	rePassOK = regexp.MustCompile(`^[\x21-\x7E]{8,20}$`) // 空白/制御を除く ASCII
)

func NewUserForSignup(userID, rawPassword string) (*User, error) {
	// 必須チェック
	if userID == "" || rawPassword == "" {
		return nil, &ErrValidation{Reason: ValidationReasonCredentialRequired}
	}
	// 長さチェック
	if l := len(userID); l < 6 || l > 20 {
		return nil, &ErrValidation{Reason: ValidationReasonInputLength}
	}
	if l := len(rawPassword); l < 8 || l > 20 {
		return nil, &ErrValidation{Reason: ValidationReasonInputLength}
	}
	// パターンチェック
	if !reUserID.MatchString(userID) || !rePassOK.MatchString(rawPassword) {
		return nil, &ErrValidation{Reason: ValidationReasonInvalidPattern}
	}
	return &User{UserID: userID}, nil
}

func (u *User) HashPassword(raw string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

func (u *User) VerifyPassword(raw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(raw)) == nil
}

// nickname: 0..30（制御コード禁止）。空文字→未設定（表示は user_id）
// comment : 0..100（制御コード禁止）。空文字→クリア（未設定）
func (u *User) ApplyProfileUpdate(nickname *string, comment *string) error {
	if nickname == nil && comment == nil {
		return &ErrValidation{Reason: ValidationReasonProfileRequired}
	}
	if nickname != nil {
		if !withinLen(*nickname, 0, 30) || hasControl(*nickname) {
			return &ErrValidation{Reason: ValidationReasonProfileConstraint}
		}
		// 空文字 = 未設定（保存は空文字のまま）
		u.Nickname = *nickname
	}
	if comment != nil {
		if !withinLen(*comment, 0, 100) || hasControl(*comment) {
			return &ErrValidation{Reason: ValidationReasonProfileConstraint}
		}
		// 空文字 = クリア
		u.Comment = *comment
	}
	return nil
}

func withinLen(s string, min, max int) bool {
	l := utf8.RuneCountInString(s)
	return l >= min && l <= max
}

func hasControl(s string) bool {
	for _, r := range s {
		// ASCII 制御（0x00-0x1F, 0x7F）を禁止
		if r < 0x20 || r == 0x7F {
			return true
		}
	}
	return false
}
