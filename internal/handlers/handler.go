package handlers

import "github.com/zeldojov/gopost/internal/store"

type Handler struct {
	store *store.Store
}

func NewHandler(st *store.Store) *Handler {
	return &Handler{
		store: st,
	}
}
