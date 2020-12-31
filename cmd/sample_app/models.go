package main

type User struct {
	ID                 int64  `json:"user_id"`
	Flags              int64  `json:"user_flags"`
	Email              string `json:"email" crudl:"req email"`
	Password           string `json:"password" crudl:"lenmax:255"`
	EmailActivationKey string `json:"email_activation_key"`
	CreatedAt          int64  `json:"created_at"`
	CreatedByUserID    int64  `json:"created_by_user_id" crudl:"link:CreatedByUser"`

	CreatedByUser *User `json:"user,omit_empty"`
}

type Session struct {
	ID        int64  `json:"session_id"`
	Flags     int64  `json:"session_flags"`
	Key       string `json:"session_key" crudl:"lenmax:50"`
	ExpiresAt int64  `json:"expires_at" crudl:"req"`
	UserID    int64  `json:"user_id" crudl:"req link:User"`

	User *User `json:"user,omit_empty"`
}

type Something struct {
	ID           int64  `json:"something_id"`
	Flags        int64  `json:"something_flags"`
	Email        string `json:"email" crudl:"req lenmin:10 lenmax:255 email"`
	Age          int    `json:"age" crudl:"req valmin:18 valmax:120"`
	Price        int    `json:"price" crudl:"req valmin:5 valmax:3580"`
	CurrencyRate int    `json:"currency_rate" crudl:"req valmin:10 valmax:50004"`
	PostCode     string `json:"post_code" crudl:"req lenmin:6 regexp:^[0-9]{2}\\-[0-9]{3}$"`
}
