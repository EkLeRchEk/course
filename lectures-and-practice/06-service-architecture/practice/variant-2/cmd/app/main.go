// Command app wires together the existing packages for practice 6, variant 2.
package main

import (
	"log"
	"net/http"

	"github.com/course-go-autumn-2026/practice6-variant2-booking/internal/client"
	"github.com/course-go-autumn-2026/practice6-variant2-booking/internal/handler"
	"github.com/course-go-autumn-2026/practice6-variant2-booking/internal/repository"
	"github.com/course-go-autumn-2026/practice6-variant2-booking/internal/service"
)

func main() {
	bookingStorage := repository.NewMemoryBookingRepository()
	employeeStorage := repository.NewMemoryEmployeeRepository()
	bookingRepo := repository.NewDomainBookingAdapter(bookingStorage)
	employeeRepo := repository.NewDomainEmployeeAdapter(employeeStorage)
	httpClient := http.DefaultClient
	availability := client.Availability{URL: "http://room-availability.internal/check", HTTP: httpClient}
	notifier := client.Notifications{URL: "http://notify.internal/send", HTTP: httpClient}
	bookingService := service.NewBookingService(bookingRepo, employeeRepo, availability, notifier)
	bookingHandler := handler.NewBookingHandler(bookingService)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux, bookingHandler)
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
