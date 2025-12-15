package subscription

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"subs-service/internal/platform/logger"
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
// @Failure      409 {object} apiError
// @Failure      500 {object} apiError
// @Router       /api/v1/subscriptions [post]
func (t *TransportHTTP) handleCreate(w http.ResponseWriter, r *http.Request) {
	log := logger.FromContext(r.Context()).With(
		zap.String("op", "http.subscription.create"),
	)

	log.Debug("request received")

	var req CreateRequest
	if err := decodeJSON(r, &req); err != nil {
		log.Warn("decode json failed", zap.Error(err))
		writeError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}

	log = log.With(
		zap.String("user_id", strings.TrimSpace(req.UserID)),
		zap.String("service_name", strings.TrimSpace(req.ServiceName)),
		zap.Int64("price", req.Price),
		zap.String("start_date", strings.TrimSpace(req.StartDate)),
		zap.Bool("has_end_date", req.EndDate != nil),
	)
	if req.EndDate != nil {
		log = log.With(zap.String("end_date", strings.TrimSpace(*req.EndDate)))
	}

	log.Debug("calling service create")
	resp, err := t.svc.Create(r.Context(), req)
	if err != nil {
		if isClientError(err) {
			log.Warn("service create failed", zap.Error(err))
		} else {
			log.Error("service create failed", zap.Error(err))
		}
		writeSvcError(w, r, err)
		return
	}

	log.Info("request succeeded", zap.String("subscription_id", resp.ID))
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
// @Failure      409 {object} apiError
// @Failure      500 {object} apiError
// @Router       /api/v1/subscriptions/{id} [get]
func (t *TransportHTTP) handleGet(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))

	log := logger.FromContext(r.Context()).With(
		zap.String("op", "http.subscription.get"),
		zap.String("subscription_id", id),
	)

	log.Debug("request received")

	if id == "" {
		log.Warn("validation failed", zap.String("reason", "id is required"))
		writeError(w, r, http.StatusBadRequest, "validation_error", "id is required")
		return
	}

	log.Debug("calling service get")
	resp, err := t.svc.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			log.Info("subscription not found")
		} else if isClientError(err) {
			log.Warn("service get failed", zap.Error(err))
		} else {
			log.Error("service get failed", zap.Error(err))
		}
		writeSvcError(w, r, err)
		return
	}

	log.Debug("request succeeded")
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
// @Failure      409 {object} apiError
// @Failure      500 {object} apiError
// @Router       /api/v1/subscriptions/{id} [patch]
func (t *TransportHTTP) handlePatch(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))

	log := logger.FromContext(r.Context()).With(
		zap.String("op", "http.subscription.update"),
		zap.String("subscription_id", id),
	)

	log.Debug("request received")

	if id == "" {
		log.Warn("validation failed", zap.String("reason", "id is required"))
		writeError(w, r, http.StatusBadRequest, "validation_error", "id is required")
		return
	}

	var req UpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		log.Warn("decode json failed", zap.Error(err))
		writeError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := validateUpdate(req); err != nil {
		log.Warn("validation failed", zap.Error(err))
		writeError(w, r, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	log = log.With(
		zap.Bool("patch_service_name", req.ServiceName != nil),
		zap.Bool("patch_price", req.Price != nil),
		zap.Bool("patch_user_id", req.UserID != nil),
		zap.Bool("patch_start_date", req.StartDate != nil),
		zap.Bool("patch_end_date", req.EndDate != nil),
	)

	log.Debug("calling service update")
	resp, err := t.svc.Update(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			log.Info("subscription not found")
		} else if isClientError(err) {
			log.Warn("service update failed", zap.Error(err))
		} else {
			log.Error("service update failed", zap.Error(err))
		}
		writeSvcError(w, r, err)
		return
	}

	log.Info("request succeeded", zap.String("subscription_id", resp.ID))
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
// @Failure      409 {object} apiError
// @Failure      500 {object} apiError
// @Router       /api/v1/subscriptions/{id} [delete]
func (t *TransportHTTP) handleDelete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))

	log := logger.FromContext(r.Context()).With(
		zap.String("op", "http.subscription.delete"),
		zap.String("subscription_id", id),
	)

	log.Debug("request received")

	if id == "" {
		log.Warn("validation failed", zap.String("reason", "id is required"))
		writeError(w, r, http.StatusBadRequest, "validation_error", "id is required")
		return
	}

	log.Debug("calling service delete")
	if err := t.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			log.Info("subscription not found")
		} else if isClientError(err) {
			log.Warn("service delete failed", zap.Error(err))
		} else {
			log.Error("service delete failed", zap.Error(err))
		}
		writeSvcError(w, r, err)
		return
	}

	log.Info("request succeeded")
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
// @Failure      409 {object} apiError
// @Failure      500 {object} apiError
// @Router       /api/v1/subscriptions [get]
func (t *TransportHTTP) handleList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	userID := strings.TrimSpace(q.Get("user_id"))
	serviceName := strings.TrimSpace(q.Get("service_name"))

	limit := parseIntDefault(q.Get("limit"), 20)
	offset := parseIntDefault(q.Get("offset"), 0)

	log := logger.FromContext(r.Context()).With(
		zap.String("op", "http.subscription.list"),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
	)
	if userID != "" {
		log = log.With(zap.String("user_id", userID))
	}
	if serviceName != "" {
		log = log.With(zap.String("service_name", serviceName))
	}

	log.Debug("request received")

	if limit <= 0 || limit > 200 {
		log.Warn("validation failed", zap.String("reason", "limit out of range"))
		writeError(w, r, http.StatusBadRequest, "validation_error", "limit must be between 1 and 200")
		return
	}
	if offset < 0 {
		log.Warn("validation failed", zap.String("reason", "offset must be >= 0"))
		writeError(w, r, http.StatusBadRequest, "validation_error", "offset must be >= 0")
		return
	}

	var uidPtr *string
	if userID != "" {
		uidPtr = &userID
	}
	var snPtr *string
	if serviceName != "" {
		snPtr = &serviceName
	}

	log.Debug("calling service list")
	resp, err := t.svc.List(r.Context(), uidPtr, snPtr, limit, offset)
	if err != nil {
		if isClientError(err) {
			log.Warn("service list failed", zap.Error(err))
		} else {
			log.Error("service list failed", zap.Error(err))
		}
		writeSvcError(w, r, err)
		return
	}

	log.Info("request succeeded",
		zap.Int("items", len(resp.Items)),
		zap.Int64("total", resp.Total),
	)
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
// @Failure      409 {object} apiError
// @Failure      500 {object} apiError
// @Router       /api/v1/subscriptions/total [get]
func (t *TransportHTTP) handleTotal(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	from := strings.TrimSpace(q.Get("from"))
	to := strings.TrimSpace(q.Get("to"))

	userID := strings.TrimSpace(q.Get("user_id"))
	serviceName := strings.TrimSpace(q.Get("service_name"))

	log := logger.FromContext(r.Context()).With(
		zap.String("op", "http.subscription.total"),
		zap.String("from", from),
		zap.String("to", to),
	)
	if userID != "" {
		log = log.With(zap.String("user_id", userID))
	}
	if serviceName != "" {
		log = log.With(zap.String("service_name", serviceName))
	}

	log.Debug("request received")

	if from == "" || to == "" {
		log.Warn("validation failed", zap.String("reason", "from and to are required"))
		writeError(w, r, http.StatusBadRequest, "validation_error", "from and to are required (MM-YYYY)")
		return
	}

	if _, err := ParseMonthMMYYYY(from); err != nil {
		log.Warn("validation failed", zap.String("reason", "invalid from format"), zap.Error(err))
		writeError(w, r, http.StatusBadRequest, "validation_error", "invalid from: "+err.Error())
		return
	}
	if _, err := ParseMonthMMYYYY(to); err != nil {
		log.Warn("validation failed", zap.String("reason", "invalid to format"), zap.Error(err))
		writeError(w, r, http.StatusBadRequest, "validation_error", "invalid to: "+err.Error())
		return
	}

	var uidPtr *string
	if userID != "" {
		uidPtr = &userID
	}
	var snPtr *string
	if serviceName != "" {
		snPtr = &serviceName
	}

	log.Debug("calling service total")
	resp, err := t.svc.Total(r.Context(), from, to, uidPtr, snPtr)
	if err != nil {
		if isClientError(err) {
			log.Warn("service total failed", zap.Error(err))
		} else {
			log.Error("service total failed", zap.Error(err))
		}
		writeSvcError(w, r, err)
		return
	}

	log.Info("request succeeded", zap.Int64("total", resp.Total))
	writeJSON(w, http.StatusOK, resp)
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

func writeSvcError(w http.ResponseWriter, r *http.Request, err error) {
	var ve ValidationError

	switch {
	case errors.As(err, &ve):
		writeError(w, r, http.StatusBadRequest, "validation_error", ve.Error())
	case errors.Is(err, ErrConflict):
		writeError(w, r, http.StatusConflict, "conflict", "resource conflict")
	case errors.Is(err, ErrNotFound):
		writeError(w, r, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, ErrInvalidInput):
		writeError(w, r, http.StatusBadRequest, "validation_error", err.Error())
	default:
		writeError(w, r, http.StatusInternalServerError, "internal_error", "internal server error")
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

func writeError(w http.ResponseWriter, r *http.Request, status int, code, msg string) {
	rid := middleware.GetReqID(r.Context())
	writeJSON(w, status, apiError{Code: code, Message: msg, RequestID: rid})
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

func isClientError(err error) bool {
	var ve ValidationError
	return errors.As(err, &ve) || errors.Is(err, ErrInvalidInput) || errors.Is(err, ErrNotFound) || errors.Is(err, ErrConflict)
}
