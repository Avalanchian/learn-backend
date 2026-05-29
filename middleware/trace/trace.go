package trace

import "github.com/google/uuid"

type trace struct {
	TraceID   uuid.UUID
	RequestID uuid.UUID
}
