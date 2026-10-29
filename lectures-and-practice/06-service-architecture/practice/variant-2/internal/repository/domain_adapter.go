package repository

import (
	"context"
	"time"

	"github.com/course-go-autumn-2026/practice6-variant2-booking/internal/domain"
)

// DomainBookingAdapter connects the pre-existing in-memory storage to the
// domain-based service without changing its original implementation.
type DomainBookingAdapter struct{ storage *MemoryBookingRepository }

func NewDomainBookingAdapter(storage *MemoryBookingRepository) *DomainBookingAdapter {
	return &DomainBookingAdapter{storage: storage}
}

func fromStored(b *storedBooking) *domain.Booking {
	value, _ := domain.NewBooking(b.RoomID, b.EmployeeID, b.StartAt, b.EndAt, b.Status == string(domain.Confirmed), b.CreatedAt)
	value = value.AssignID(b.ID)
	switch domain.Status(b.Status) {
	case domain.Cancelled:
		value.Cancel()
	case domain.Rescheduling:
		// Recover the persisted rescheduling status via the existing domain transition.
		// The status is already known to be rescheduling here; a new pending booking
		// cannot be mutated directly from another package.
		value, _ = domain.NewBooking(b.RoomID, b.EmployeeID, b.StartAt, b.EndAt, true, b.CreatedAt)
		value = value.AssignID(b.ID)
		_ = value.Reschedule(b.StartAt, b.EndAt, b.StartAt, 2*time.Hour)
	}
	return &value
}

func toStored(b *domain.Booking) *storedBooking {
	s := b.Snapshot()
	return &storedBooking{ID: s.ID, RoomID: s.RoomID, EmployeeID: s.EmployeeID,
		StartAt: s.StartAt, EndAt: s.EndAt, Status: string(s.Status), CreatedAt: s.CreatedAt}
}

func (a *DomainBookingAdapter) Create(ctx context.Context, b *domain.Booking) (int64, error) {
	record := toStored(b)
	if err := a.storage.Create(ctx, record); err != nil {
		return 0, err
	}
	return record.ID, nil
}
func (a *DomainBookingAdapter) Get(ctx context.Context, id int64) (*domain.Booking, error) {
	record, err := a.storage.FindById(ctx, id)
	if err == ErrNotFound {
		return nil, domain.ErrBookingNotFound
	}
	if err != nil {
		return nil, err
	}
	return fromStored(record), nil
}
func (a *DomainBookingAdapter) Update(ctx context.Context, id int64, b *domain.Booking) (*domain.Booking, error) {
	record := toStored(b)
	record.ID = id
	if err := a.storage.Update(ctx, record); err != nil {
		if err == ErrNotFound {
			return nil, domain.ErrBookingNotFound
		}
		return nil, err
	}
	return fromStored(record), nil
}
func (a *DomainBookingAdapter) GetBookings(ctx context.Context, roomID int64, start, end time.Time) ([]*domain.Booking, error) {
	a.storage.mu.Lock()
	defer a.storage.mu.Unlock()
	result := make([]*domain.Booking, 0)
	for _, b := range a.storage.data {
		if b.RoomID == roomID && b.Status != string(domain.Cancelled) && start.Before(b.EndAt) && b.StartAt.Before(end) {
			result = append(result, fromStored(b))
		}
	}
	return result, nil
}

// DomainEmployeeAdapter bridges the original employee storage to the service.
type DomainEmployeeAdapter struct{ storage *MemoryEmployeeRepository }

func NewDomainEmployeeAdapter(storage *MemoryEmployeeRepository) *DomainEmployeeAdapter {
	storage.mu.Lock()
	storage.employees[1] = &Employee{ID: 1, Role: "regular"}
	storage.employees[2] = &Employee{ID: 2, Role: "admin"}
	storage.mu.Unlock()
	return &DomainEmployeeAdapter{storage: storage}
}
func (a *DomainEmployeeAdapter) Get(ctx context.Context, id int64) (*domain.Employee, error) {
	e, err := a.storage.GetByID(ctx, id)
	if err == ErrNotFound {
		return nil, domain.ErrEmployeeNotFound
	}
	if err != nil {
		return nil, err
	}
	return &domain.Employee{ID: e.ID, Role: domain.Role(e.Role)}, nil
}
