package controller_manager

import "github.com/google/uuid"

type BasicEntity struct {
	Name string
	UUID uuid.UUID
}

type EntityReceiver interface {
}
