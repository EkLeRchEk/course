package main

import "time"

// storedBooking is the type referenced by the untouched original main.go.
// It is kept separately so the original sample can still compile.
type storedBooking struct {
	ID         int64
	RoomID     int64
	EmployeeID int64
	StartAt    time.Time
	EndAt      time.Time
	Status     string
	CreatedAt  time.Time
}
