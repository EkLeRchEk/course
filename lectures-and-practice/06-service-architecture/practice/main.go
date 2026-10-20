// Package main - сервис объявлений, три ручки, весь код в одном файле.
// Работает по ТЗ ниже, но архитектурно неправильно устроен - это и
// есть задание: привести к структуре слоёв без изменения поведения.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// --- Хранилище в памяти вместо настоящей БД (для практики достаточно,
// протокол обращения к нему такой же неудобный, как к *sql.DB). ---

type storedListing struct {
	ID        int64
	Title     string
	Price     float64
	PhotoURL  string
	SellerID  int64
	Status    string // "pending" | "approved" | "rejected"
	CreatedAt time.Time
}

var (
	db       = map[int64]*storedListing{}
	nextID   int64
	moderationURL = "http://moderation.internal/check"
)

// --- Запросы/ответы ---

type createListingRequest struct {
	Title    string  `json:"title"`
	Price    float64 `json:"price"`
	PhotoURL string  `json:"photo_url"`
	SellerID int64   `json:"seller_id"`
}

type updatePriceRequest struct {
	Price float64 `json:"price"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /listings", createListingHandler)
	mux.HandleFunc("GET /listings/{id}", getListingHandler)
	mux.HandleFunc("PATCH /listings/{id}/price", updatePriceHandler)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

// createListingHandler - POST /listings. Делает всё сразу: валидация,
// запись в "базу", поход в модерацию, решение по ответу.
func createListingHandler(w http.ResponseWriter, r *http.Request) {
	var req createListingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Бизнес-правило №1: нет цены - нет публикации.
	if req.Price <= 0 {
		http.Error(w, "price must be positive", http.StatusBadRequest)
		return
	}

	// Бизнес-правило №2: нет фото - нет публикации.
	if strings.TrimSpace(req.PhotoURL) == "" {
		http.Error(w, "photo is required", http.StatusBadRequest)
		return
	}

	// Бизнес-правило №3: цена не может быть выше 10 000 000 - считаем
	// это подозрительным объявлением и сразу отклоняем.
	if req.Price > 10_000_000 {
		http.Error(w, "price looks suspicious", http.StatusBadRequest)
		return
	}

	nextID++
	id := nextID
	listing := &storedListing{
		ID:        id,
		Title:     req.Title,
		Price:     req.Price,
		PhotoURL:  req.PhotoURL,
		SellerID:  req.SellerID,
		Status:    "pending",
		CreatedAt: time.Now(),
	}
	db[id] = listing

	// Поход во внешний сервис модерации прямо из хендлера.
	modReq, _ := http.NewRequestWithContext(r.Context(), http.MethodPost,
		moderationURL, strings.NewReader(fmt.Sprintf(`{"listing_id":%d}`, id)))
	modReq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(modReq)
	if err != nil {
		// Модерация недоступна - но объявление уже создано.
		// Решаем тут же, по месту: отвечаем 201 pending.
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": id, "status": "pending"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		listing.Status = "approved"
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": id, "status": listing.Status})
}

// getListingHandler - GET /listings/{id}.
func getListingHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	listing, ok := db[id]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(listing)
}

// updatePriceHandler - PATCH /listings/{id}/price. Меняет цену, но
// только если объявление ещё не отклонено - бизнес-правило №4, зашитое
// прямо тут же, рядом с разбором запроса и ответом клиенту.
func updatePriceHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var req updatePriceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if req.Price <= 0 {
		http.Error(w, "price must be positive", http.StatusBadRequest)
		return
	}

	listing, ok := db[id]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Бизнес-правило №4: отклонённое объявление нельзя редактировать.
	if listing.Status == "rejected" {
		http.Error(w, "cannot update rejected listing", http.StatusConflict)
		return
	}

	listing.Price = req.Price
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(listing)
}

// findByID - вспомогательная функция, имитирует поход в БД через
// database/sql, чтобы в коде было видно характерное ветвление
// sql.ErrNoRows - даже хотя реальной БД тут нет. Не используется
// напрямую в хендлерах выше (они читают db map напрямую), но
// пригодится как пример того, что нужно вынести в репозиторий.
func findByID(ctx context.Context, conn *sql.DB, id int64) (*storedListing, error) {
	row := conn.QueryRowContext(ctx, `SELECT * FROM listings WHERE id = $1`, id)
	var l storedListing
	err := row.Scan(&l.ID, &l.Title, &l.Price, &l.PhotoURL, &l.SellerID, &l.Status, &l.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		// Ошибка "не найдено" отдаётся вызывающему коду как есть -
		// кто вызвал findByID, тот и решает, что с sql.ErrNoRows делать.
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}
