package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Availability makes the same HTTP request as the original standalone main.
type Availability struct {
	URL  string
	HTTP *http.Client
}

func (a Availability) Check(ctx context.Context, roomID int64, start, end time.Time) (bool, error) {
	body := fmt.Sprintf(`{"room_id":%d,"start_at":%q,"end_at":%q}`, roomID, start.Format(time.RFC3339), end.Format(time.RFC3339))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.URL, strings.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := a.HTTP.Do(req)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	return response.StatusCode == http.StatusOK, nil
}

// Notifications are best effort: the service deliberately ignores failures.
type Notifications struct {
	URL  string
	HTTP *http.Client
}

func (n Notifications) BookingCreated(ctx context.Context, id int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.URL, strings.NewReader(fmt.Sprintf(`{"booking_id":%d}`, id)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := n.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return nil
}
