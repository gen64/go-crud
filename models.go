package main

type User struct {
	ID                 int64  `json:"user_id"`
	Flags              int64  `json:"user_flags"`
	Email              string `json:"email" f0x:"req email"`
	Password           string `json:"password" f0x:"lenmax:255"`
	EmailActivationKey string `json:"email_activation_key"`
	CreatedAt          int64  `json:"created_at"`
	CreatedByUserID    int64  `json:"created_by_user_id" f0x:"link:CreatedByUser"`

	CreatedByUser      *User  `json:"user,omit_empty"`
}

type Session struct {
	ID                 int64  `json:"session_id"`
	Flags              int64  `json:"session_flags"`
	Key                string `json:"session_key" f0x:"lenmax:50"`
	ExpiresAt          int64  `json:"expires_at" f0x:"req"`
	UserID             int64  `json:"user_id" f0x:"req link:User"`

	User               *User  `json:"user,omit_empty"`
}
