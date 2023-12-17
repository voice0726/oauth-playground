package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Client struct {
	ID           uuid.UUID
	Name         string
	Secret       string
	RedirectURIs datatypes.JSONSlice[string]
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (c *Client) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID = uuid.New()
	return
}

type AuthRequest struct {
	ID           uuid.UUID
	ClientID     uuid.UUID
	ResponseType string
	RedirectURI  string
	State        string
	Scope        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (r *AuthRequest) BeforeCreate(tx *gorm.DB) (err error) {
	r.ID = uuid.New()
	return
}

type AuthCode struct {
	ID        uuid.UUID
	State     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (c *AuthCode) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID = uuid.New()
	return
}

type Token struct {
}
