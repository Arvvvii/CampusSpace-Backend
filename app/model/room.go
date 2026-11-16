package model

import "time"

type Room struct {
	ID        int64     `json:"id" db:"id"`
	Code      string    `json:"code" db:"code"`
	Name      string    `json:"name" db:"name"`
	Building  string    `json:"building" db:"building"`
	Capacity  int       `json:"capacity" db:"capacity"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type CreateRoomRequest struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Building string `json:"building"`
	Capacity int    `json:"capacity"`
}

type UpdateRoomRequest struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Building string `json:"building"`
	Capacity int    `json:"capacity"`
	IsActive bool   `json:"is_active"`
}
