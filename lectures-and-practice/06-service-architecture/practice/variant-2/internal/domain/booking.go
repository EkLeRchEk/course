package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalidInterval = errors.New("end_at must be after start_at")
	ErrBookingNotFound = errors.New("booking not found")
	ErrRoomBooked      = errors.New("room is already booked for this time")
	ErrCancelled       = errors.New("cannot reschedule cancelled booking")
)

type Status string

const (
	Pending      Status = "pending_comfigurate"
	Confirmed    Status = "confirmed"
	Cancelled    Status = "cancelled"
	Rescheduling Status = "rescheduling"
)

type Role string

const (
	Regular Role = "regular"
	Admin   Role = "admin"
)

type Employee struct {
	ID   int64
	Role Role
}

func (e Employee) CanOverrideBooking() bool {
	return e.Role == Admin
}

type Booking struct {
	id, roomID, employeeID   int64
	startAt, endAt, createAt time.Time
	Status                   Status
}

// func NewBooking(
// 	roomID, employeeID int64,
// 	startAt, endAt time.Time,
// 	available bool,
// 	now time.Time,
// ) (Booking, error) {
// 	if !endAt.After(startAt) {
// 		return Booking{}, ErrInvalidInterval
// 	}

// 	status :=
// }
