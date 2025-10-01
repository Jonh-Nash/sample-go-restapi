package domain

import "errors"

// ErrValidation は API cause にそのまま使う文字列を持つドメインエラー
type ErrValidation struct {
	Cause string
}

func (e *ErrValidation) Error() string { return e.Cause }

// 共通エラー
var (
	ErrAuthFailed = errors.New("auth failed") // usecase で 401 に対応
)
