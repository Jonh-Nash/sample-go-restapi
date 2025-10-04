package rest

// 共通レスポンス
type messageOnly struct {
	Message string `json:"message"`
}

// /signup 入力
type signUpRequest struct {
	UserID   string `json:"user_id"`
	Password string `json:"password"`
}

// /signup 出力
type signUpResponse struct {
	Message string            `json:"message"`
	User    userSummaryNoComm `json:"user"`
}

type userSummaryNoComm struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
}

// GET/PATCH のユーザ表示
type userResponse struct {
	Message string     `json:"message"`
	User    userDetail `json:"user"`
}

type userDetail struct {
	UserID   string  `json:"user_id"`
	Nickname string  `json:"nickname"`
	Comment  *string `json:"comment,omitempty"`
}

// PATCH 入力
type updateUserRequest struct {
	Nickname *string `json:"nickname,omitempty"`
	Comment  *string `json:"comment,omitempty"`
	UserID   *string `json:"user_id,omitempty"`
	Password *string `json:"password,omitempty"`
}
