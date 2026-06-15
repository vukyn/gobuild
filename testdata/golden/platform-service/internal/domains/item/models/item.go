package models

import (
	"errors"
	"time"
)

type CreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (r CreateRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

type UpdateRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *int32  `json:"status"`
}

func (r UpdateRequest) Validate() error {
	if r.Name != nil && *r.Name == "" {
		return errors.New("name cannot be empty")
	}
	return nil
}

type ListRequest struct {
	Search   string `query:"search"`
	Status   int32  `query:"status"`
	Page     int    `query:"page"`
	PageSize int    `query:"page_size"`
}

type ItemResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      int32     `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ListResponse struct {
	Items []ItemResponse `json:"items"`
	Total int64          `json:"total"`
}
