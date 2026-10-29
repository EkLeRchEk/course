package service

import (
	"context"
	"errors"
	"time"

	"github.com/course-go-autumn-2026/practice6-variant2-booking/internal/domain"
)

type BookingService struct {
	bookingRepo  BookingRepository
	employeeRepo EmployeeRepository
	availability RoomAvailability
	notifier     Notifier
}

type CreateBookingInput struct {
	RoomID     int64
	EmployeeID int64
	StartAt    time.Time
	EndAt      time.Time
}

func (s *BookingService) Create(ctx context.Context, in CreateBookingInput) (*domain.Booking, error) {
	booking, err := domain.NewBooking(in.RoomID, in.EmployeeID, in.StartAt, in.EndAt, time.Now())
	if err != nil {
		return nil, err
	}

	canOverride, err := s.canOverride(ctx, in.EmployeeID)
	if err != nil {
		return nil, err
	}

	existing, err := s.bookingRepo.GetBookings(ctx, in.RoomID, in.StartAt, in.EndAt)
	if err != nil {
		return nil, err
	}

	var conflicts []*domain.Booking
	for _, b := range existing {
		if b.IsActive() { // отменённые комнату не занимают
			conflicts = append(conflicts, b)
		}
	}

	if len(conflicts) > 0 {
		if !canOverride {
			return nil, domain.ErrRoomBooked
		}
		// Админ побеждает: отменяются все пересекающиеся брони.
		for _, c := range conflicts {
			if err := c.Cancel(); err != nil {
				return nil, err
			}
			if _, err := s.bookingRepo.Update(ctx, c.ID(), c); err != nil {
				return nil, err
			}
		}
	}

	available, err := s.availability.Check(ctx, in.RoomID, in.StartAt, in.EndAt)
	if err == nil && available {
		if err := booking.Confirm(); err != nil {
			return nil, err
		}
	}

	id, err := s.bookingRepo.Create(ctx, booking)
	if err != nil {
		return nil, err
	}
	booking.AssignID(id)

	_ = s.notifier.BookingCreated(ctx, id)

	return booking, nil
}

func (s *BookingService) Get(ctx context.Context, id int64) (*domain.Booking, error) {
	return s.bookingRepo.Get(ctx, id)
}

func (s *BookingService) Reschedule(ctx context.Context, id int64, startAt, endAt time.Time) (*domain.Booking, error) {
	if !endAt.After(startAt) {
		return nil, domain.ErrInvalidInterval
	}

	booking, err := s.bookingRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := booking.Reschedule(startAt, endAt, time.Now()); err != nil {
		return nil, err
	}
	return s.bookingRepo.Update(ctx, id, booking)
}

func (s *BookingService) canOverride(ctx context.Context, employeeID int64) (bool, error) {
	emp, err := s.employeeRepo.Get(ctx, employeeID)
	if errors.Is(err, domain.ErrEmployeeNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return emp.CanOverrideBooking(), nil
}
