package subscription

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type TransportHTTP struct {
	svc Service
}

func NewTransportHTTP(svc Service) *TransportHTTP {
	return &TransportHTTP{svc: svc}
}

func (t *TransportHTTP) Routes() http.Handler {
	r := chi.NewRouter()

	r.Post("/", t.handleCreate)
	r.Get("/", t.handleList)
	r.Get("/total", t.handleTotal)
	r.Get("/{id}", t.handleGet)
	r.Patch("/{id}", t.handlePatch)
	r.Delete("/{id}", t.handleDelete)

	return r
}

// handleCreate godoc
// @Summary      Create subscription
// @Description  Creates a new subscription record
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        request body CreateRequest true "Create subscription request"
// @Success      201 {object} SubscriptionResponse
// @Failure      400 {object} apiError
// @Failure      500 {object} apiError
// @Router       /api/v1/subscriptions [post]
func (t *TransportHTTP) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	resp, err := t.svc.Create(r.Context(), req)
	if err != nil {
		writeSvcError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// handleGet godoc
// @Summary      Get subscription
// @Description  Returns subscription by id
// @Tags         subscriptions
// @Produce      json
// @Param        id path string true "Subscription ID (UUID)"
// @Success      200 {object} SubscriptionResponse
// @Failure      400 {object} apiError
// @Failure      404 {object} apiError
// @Failure      500 {object} apiError
// @Router       /api/v1/subscriptions/{id} [get]
func (t *TransportHTTP) handleGet(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if strings.TrimSpace(id) == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "id is required")
		return
	}

	resp, err := t.svc.Get(r.Context(), id)
	if err != nil {
		writeSvcError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// handlePatch godoc
// @Summary      Update subscription
// @Description  Partially updates subscription by id
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        id path string true "Subscription ID (UUID)"
// @Param        request body UpdateRequest true "Update subscription request"
// @Success      200 {object} SubscriptionResponse
// @Failure      400 {object} apiError
// @Failure      404 {object} apiError
// @Failure      500 {object} apiError
// @Router       /api/v1/subscriptions/{id} [patch]
func (t *TransportHTTP) handlePatch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if strings.TrimSpace(id) == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "id is required")
		return
	}

	var req UpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := validateUpdate(req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	resp, err := t.svc.Update(r.Context(), id, req)
	if err != nil {
		writeSvcError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleDelete godoc
// @Summary      Delete subscription
// @Description  Deletes subscription by id
// @Tags         subscriptions
// @Param        id path string true "Subscription ID (UUID)"
// @Success      204 "No Content"
// @Failure      400 {object} apiError
// @Failure      404 {object} apiError
// @Failure      500 {object} apiError
// @Router       /api/v1/subscriptions/{id} [delete]
func (t *TransportHTTP) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if strings.TrimSpace(id) == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "id is required")
		return
	}

	if err := t.svc.Delete(r.Context(), id); err != nil {
		writeSvcError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleList godoc
// @Summary      List subscriptions
// @Description  Lists subscriptions with optional filters and pagination
// @Tags         subscriptions
// @Produce      json
// @Param        user_id query string false "Filter by user id (UUID)"
// @Param        service_name query string false "Filter by service name"
// @Param        limit query int false "Limit (1..200)" default(20)
// @Param        offset query int false "Offset (>=0)" default(0)
// @Success      200 {object} ListResponse
// @Failure      400 {object} apiError
// @Failure      500 {object} apiError
// @Router       /api/v1/subscriptions [get]
func (t *TransportHTTP) handleList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var userID *string
	if v := strings.TrimSpace(q.Get("user_id")); v != "" {
		userID = &v
	}

	var serviceName *string
	if v := strings.TrimSpace(q.Get("service_name")); v != "" {
		serviceName = &v
	}

	limit := parseIntDefault(q.Get("limit"), 20)
	offset := parseIntDefault(q.Get("offset"), 0)
	if limit <= 0 || limit > 200 {
		writeError(w, http.StatusBadRequest, "validation_error", "limit must be between 1 and 200")
		return
	}
	if offset < 0 {
		writeError(w, http.StatusBadRequest, "validation_error", "offset must be >= 0")
		return
	}

	resp, err := t.svc.List(r.Context(), userID, serviceName, limit, offset)
	if err != nil {
		writeSvcError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleTotal godoc
// @Summary      Calculate total cost
// @Description  Calculates total monthly cost for subscriptions within period (inclusive) with optional filters
// @Tags         subscriptions
// @Produce      json
// @Param        from query string true "Start month (MM-YYYY)" example(07-2025)
// @Param        to query string true "End month (MM-YYYY)" example(12-2025)
// @Param        user_id query string false "Filter by user id (UUID)"
// @Param        service_name query string false "Filter by service name"
// @Success      200 {object} TotalResponse
// @Failure      400 {object} apiError
// @Failure      500 {object} apiError
// @Router       /api/v1/subscriptions/total [get]
func (t *TransportHTTP) handleTotal(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	from := strings.TrimSpace(q.Get("from"))
	to := strings.TrimSpace(q.Get("to"))
	if from == "" || to == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "from and to are required (MM-YYYY)")
		return
	}
	if _, err := ParseMonthMMYYYY(from); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "invalid from: "+err.Error())
		return
	}
	if _, err := ParseMonthMMYYYY(to); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "invalid to: "+err.Error())
		return
	}

	var userID *string
	if v := strings.TrimSpace(q.Get("user_id")); v != "" {
		userID = &v
	}

	var serviceName *string
	if v := strings.TrimSpace(q.Get("service_name")); v != "" {
		serviceName = &v
	}

	resp, err := t.svc.Total(r.Context(), from, to, userID, serviceName)
	if err != nil {
		writeSvcError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func validateCreate(req CreateRequest) error {
	if strings.TrimSpace(req.ServiceName) == "" {
		return errors.New("service_name is required")
	}
	if req.Price < 0 {
		return errors.New("price must be >= 0")
	}
	if strings.TrimSpace(req.UserID) == "" {
		return errors.New("user_id is required")
	}
	if _, err := ParseMonthMMYYYY(req.StartDate); err != nil {
		return errors.New("start_date: " + err.Error())
	}
	if req.EndDate != nil {
		em, err := ParseMonthMMYYYY(*req.EndDate)
		if err != nil {
			return errors.New("end_date: " + err.Error())
		}
		sm, _ := ParseMonthMMYYYY(req.StartDate)
		if em.Compare(sm) < 0 {
			return errors.New("end_date must be >= start_date")
		}
	}
	return nil
}

func validateUpdate(req UpdateRequest) error {
	if req.ServiceName != nil && strings.TrimSpace(*req.ServiceName) == "" {
		return errors.New("service_name must not be empty")
	}
	if req.Price != nil && *req.Price < 0 {
		return errors.New("price must be >= 0")
	}
	if req.UserID != nil && strings.TrimSpace(*req.UserID) == "" {
		return errors.New("user_id must not be empty")
	}
	if req.StartDate != nil {
		if _, err := ParseMonthMMYYYY(*req.StartDate); err != nil {
			return errors.New("start_date: " + err.Error())
		}
	}
	if req.EndDate != nil {
		if _, err := ParseMonthMMYYYY(*req.EndDate); err != nil {
			return errors.New("end_date: " + err.Error())
		}
	}
	if req.StartDate != nil && req.EndDate != nil {
		sm, _ := ParseMonthMMYYYY(*req.StartDate)
		em, _ := ParseMonthMMYYYY(*req.EndDate)
		if em.Compare(sm) < 0 {
			return errors.New("end_date must be >= start_date")
		}
	}
	return nil
}

func writeSvcError(w http.ResponseWriter, err error) {
	var ve ValidationError

	switch {
	case errors.As(err, &ve):
		writeError(w, http.StatusBadRequest, "validation_error", ve.Error())
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, apiError{Code: code, Message: msg})
}

func parseIntDefault(s string, def int) int {
	if strings.TrimSpace(s) == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}
