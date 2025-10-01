package usecase

import (
	"errors"

	"accountapi/internal/domain"
)

type Usecase struct {
	Repo domain.UserRepository
}

var (
	ErrAuthFailed = errors.New("auth failed") // 401
	ErrNoPerm     = errors.New("no perm")     // 403
)

// SignUp: 既存チェック、ハッシュ化、作成
func (u *Usecase) SignUp(userID, rawPassword string) (*domain.User, error) {
	user, err := domain.NewUserForSignup(userID, rawPassword)
	if err != nil {
		return nil, err
	}
	if err := user.HashPassword(rawPassword); err != nil {
		return nil, err
	}
	rec := &domain.UserRecord{
		UserID:       user.UserID,
		PasswordHash: user.PasswordHash,
		Nickname:     "", // 未設定
		Comment:      "",
	}
	if err := u.Repo.Create(rec); err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			return nil, &domain.ErrValidation{Cause: "Already same user_id is used"}
		}
		return nil, err
	}
	return user, nil
}

// GetUser: Basic 認証（userID/pw）を検証して本人の情報を返す
func (u *Usecase) GetUser(pathUserID, authUserID, authPassword string) (*domain.User, error) {
	// 認証ユーザーの存在確認とパスワード検証
	authRec, err := u.Repo.FindByID(authUserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, ErrAuthFailed
		}
		return nil, err
	}
	authUser := toDomain(authRec)
	if !authUser.VerifyPassword(authPassword) {
		return nil, ErrAuthFailed
	}

	// 自身の場合はそのまま返す
	if pathUserID == authUserID {
		return authUser, nil
	}

	// 別ユーザーの取得
	targetRec, err := u.Repo.FindByID(pathUserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return toDomain(targetRec), nil
}

// UpdateUser: 本人認証し、プロフィールのみ更新
func (u *Usecase) UpdateUser(pathUserID, authUserID, authPassword string, nickname *string, comment *string, forbidChangingIDOrPass bool) (*domain.User, error) {
	if pathUserID != authUserID {
		return nil, ErrNoPerm // 403
	}
	rec, err := u.Repo.FindByID(authUserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	d := toDomain(rec)
	if !d.VerifyPassword(authPassword) {
		return nil, ErrAuthFailed
	}
	// user_id/password がボディに含まれていたら即 400
	if forbidChangingIDOrPass {
		return nil, &domain.ErrValidation{Cause: "Not updatable user_id and password"}
	}
	if err := d.ApplyProfileUpdate(nickname, comment); err != nil {
		return nil, err
	}
	if err := u.Repo.UpdateProfile(d.UserID, d.Nickname, d.Comment); err != nil {
		return nil, err
	}
	return d, nil
}

// CloseUser: 本人認証し、物理削除（未存在も 401）
func (u *Usecase) CloseUser(authUserID, authPassword string) error {
	rec, err := u.Repo.FindByID(authUserID)
	if err != nil {
		// /close は未存在も 401
		return ErrAuthFailed
	}
	d := toDomain(rec)
	if !d.VerifyPassword(authPassword) {
		return ErrAuthFailed
	}
	if err := u.Repo.Delete(d.UserID); err != nil {
		return err
	}
	return nil
}

func toDomain(rec *domain.UserRecord) *domain.User {
	return &domain.User{
		UserID:       rec.UserID,
		PasswordHash: rec.PasswordHash,
		Nickname:     rec.Nickname,
		Comment:      rec.Comment,
		Deleted:      rec.Deleted,
	}
}
