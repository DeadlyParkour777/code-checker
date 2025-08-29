package types

import "time"

type User struct {
	ID        string
	Username  string
	Password  string
	Role      string
	CreatedAt time.Time
}

type UserLoginPayload struct {
	Username string
	Password string
}

type UserRegisterPayload struct {
	Username string
	Password string
}
