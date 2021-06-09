package main

type User struct {
	ID                 int64  `json:"user_id"`
	Flags              int64  `json:"user_flags"`
	Email              string `json:"email" crud:"req email"`
	Password           string `json:"password" crud:"lenmax:255"`
	EmailActivationKey string `json:"email_activation_key"`
	CreatedAt          int64  `json:"created_at"`
	CreatedByUserID    int64  `json:"created_by_user_id"`
}

type Session struct {
	ID        int64  `json:"session_id"`
	Flags     int64  `json:"session_flags"`
	Key       string `json:"session_key" crud:"lenmax:50"`
	ExpiresAt int64  `json:"expires_at" crud:"req"`
	UserID    int64  `json:"user_id" crud:"req"`
}

type Something struct {
	ID           int64  `json:"something_id"`
	Flags        int64  `json:"something_flags"`
	Email        string `json:"email" crud:"req lenmin:10 lenmax:255 email"`
	Age          int    `json:"age" crud:"req valmin:18 valmax:120"`
	Price        int    `json:"price" crud:"req valmin:5 valmax:3580"`
	CurrencyRate int    `json:"currency_rate" crud:"req valmin:10 valmax:50004"`
	PostCode     string `json:"post_code" crud:"req lenmin:6 regexp:^[0-9]{2}\\-[0-9]{3}$"`
}
