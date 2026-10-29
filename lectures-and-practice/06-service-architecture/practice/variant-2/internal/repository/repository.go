package repository

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrNotFound = errors.New("not found")

type storedBooking struct {
	ID         int64
	RoomID     int64
	EmployeeID int64
	StartAt    time.Time
	EndAt      time.Time
	Status     string // "pending_confirmation" | "confirmed" | "cancelled" | "rescheduling"
	CreatedAt  time.Time
}

type Employee struct {
	ID   int64
	Role string // "regular" | "admin"
}

type bookingRepository interface {
	FindById(ctx context.Context, id int64) (*storedBooking, error)
	FindOverlapping(ctx context.Context, roomID int64, startAt, endAt time.Time) ([]*storedBooking, error)
	Create(ctx context.Context, booking *storedBooking) error
	Update(ctx context.Context, booking *storedBooking) error
}

type EmployeeRepository interface {
	GetByID(ctx context.Context, id int64) (*Employee, error)
}

type MemoryBookingRepository struct {
	mu     sync.Mutex
	data   map[int64]*storedBooking
	nextID int64
}

type MemoryEmployeeRepository struct {
	mu        sync.RWMutex
	employees map[int64]*Employee
}

func NewMemoryBookingRepository() *MemoryBookingRepository {
	return &MemoryBookingRepository{
		data: make(map[int64]*storedBooking),
	}
}

func NewMemoryEmployeeRepository() *MemoryEmployeeRepository {
	return &MemoryEmployeeRepository{
		employees: make(map[int64]*Employee),
	}
}

// Booking
func (r *MemoryBookingRepository) FindById(ctx context.Context, id int64) (*storedBooking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.data[id]
	if !ok {
		return nil, ErrNotFound
	}
	copied := *b
	return &copied, nil
}

func (r *MemoryBookingRepository) FindOverlapping(ctx context.Context, roomID int64, startAt, endAt time.Time) (*storedBooking, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, b := range r.data {
		if b.RoomID != roomID || b.Status == "cancelled" {
			continue
		}
		if startAt.Before(b.EndAt) && b.StartAt.Before(startAt) {
			copied := *b
			return &copied, nil
		}
	}
	return nil, nil
}

func (r *MemoryBookingRepository) Create(ctx context.Context, booking *storedBooking) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	booking.ID = r.nextID
	copied := *booking
	r.data[copied.ID] = &copied
	return nil
}

func (r *MemoryBookingRepository) Update(ctx context.Context, booking *storedBooking) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.data[booking.ID]; !ok {
		return ErrNotFound
	}

	copied := *booking
	r.data[copied.ID] = &copied
	return nil
}

// Employee
func (r *MemoryEmployeeRepository) GetByID(ctx context.Context, id int64) (*Employee, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.employees[id]
	if !ok {
		return &Employee{}, ErrNotFound
	}
	copied := *e
	return &copied, nil
}
