package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"homework/internal/model"
	"homework/internal/repository"
)

const deviceCollectionPath = "/api/v1/devices"

// DeviceHandler serves requests for the devices resource.
type DeviceHandler struct {
	store repository.DeviceStore
}

type apiError struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewDeviceHandler creates a handler backed by the provided device store.
func NewDeviceHandler(store repository.DeviceStore) *DeviceHandler {
	return &DeviceHandler{store: store}
}

// Health returns the service health status.
func (h *DeviceHandler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, struct {
		Status string `json:"status"`
	}{Status: "ok"})
}

// Collection handles requests to the devices collection.
func (h *DeviceHandler) Collection(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		h.list(w, r)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	request, err := decodeDeviceRequest(r)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "validation_error", "Invalid request body")
		return
	}

	device, err := h.store.Create(model.Device{
		Name:           request.Name,
		Serial:         request.Serial,
		Manufacturer:   request.Manufacturer,
		Type:           request.Type,
		WarrantyMonths: request.WarrantyMonths,
		Comment:        request.Comment,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_server_error", "Could not create device")
		return
	}
	writeJSON(w, http.StatusCreated, device)
}

func (h *DeviceHandler) list(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	page, ok := positiveQueryParam(query, "page", 1)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_pagination", "Invalid pagination parameters")
		return
	}
	limit, ok := positiveQueryParam(query, "limit", 10)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid_pagination", "Invalid pagination parameters")
		return
	}
	if limit > 100 {
		limit = 100
	}

	devices := h.store.List()
	if filter := query.Get("type"); filter != "" {
		filtered := make([]model.Device, 0, len(devices))
		for _, device := range devices {
			if device.Type == filter {
				filtered = append(filtered, device)
			}
		}
		devices = filtered
	}

	total := uint64(len(devices))
	lastPage := total / limit
	if total%limit != 0 {
		lastPage++
	}
	if page > lastPage {
		writeJSON(w, http.StatusOK, []model.Device{})
		return
	}

	start := int((page - 1) * limit)
	end := len(devices)
	remaining := len(devices) - start
	if limit < uint64(remaining) {
		end = start + int(limit)
	}
	writeJSON(w, http.StatusOK, devices[start:end])
}

func positiveQueryParam(query url.Values, name string, defaultValue uint64) (uint64, bool) {
	values, exists := query[name]
	if !exists {
		return defaultValue, true
	}
	if len(values) == 0 || values[0] == "" {
		return 0, false
	}
	for _, char := range values[0] {
		if char < '0' || char > '9' {
			return 0, false
		}
	}

	value, err := strconv.ParseUint(values[0], 10, 64)
	return value, err == nil && value >= 1
}

// Item handles requests for a single device.
func (h *DeviceHandler) Item(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, deviceCollectionPath+"/")
	if path == "" || strings.Contains(path, "/") {
		writeError(w, http.StatusNotFound, "not_found", "Device not found")
		return
	}

	if r.Method != http.MethodGet && r.Method != http.MethodPut && r.Method != http.MethodDelete {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
		return
	}

	id, err := parseID(path)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_id", "Invalid device ID")
		return
	}

	switch r.Method {
	case http.MethodGet:
		device, ok := h.store.Get(id)
		if !ok {
			writeError(w, http.StatusNotFound, "not_found", "Device not found")
			return
		}
		writeJSON(w, http.StatusOK, device)
	case http.MethodPut:
		request, err := decodeDeviceRequest(r)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, "validation_error", "Invalid request body")
			return
		}
		device, ok := h.store.Update(id, model.Device{
			Name:           request.Name,
			Serial:         request.Serial,
			Manufacturer:   request.Manufacturer,
			Type:           request.Type,
			WarrantyMonths: request.WarrantyMonths,
			Comment:        request.Comment,
		})
		if !ok {
			writeError(w, http.StatusNotFound, "not_found", "Device not found")
			return
		}
		writeJSON(w, http.StatusOK, device)
	case http.MethodDelete:
		if !h.store.Delete(id) {
			writeError(w, http.StatusNotFound, "not_found", "Device not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// NotFound returns the API's standard response for an unknown route.
func NotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not_found", "Resource not found")
}

func decodeDeviceRequest(r *http.Request) (model.CreateDeviceRequest, error) {
	var fields map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&fields); err != nil || fields == nil {
		return model.CreateDeviceRequest{}, errInvalidBody
	}
	if err := ensureNoTrailingJSON(r); err != nil {
		return model.CreateDeviceRequest{}, errInvalidBody
	}

	name, ok := requiredString(fields, "name")
	if !ok {
		return model.CreateDeviceRequest{}, errInvalidBody
	}
	serial, ok := requiredString(fields, "serial")
	if !ok {
		return model.CreateDeviceRequest{}, errInvalidBody
	}
	manufacturer, ok := requiredString(fields, "manufacturer")
	if !ok {
		return model.CreateDeviceRequest{}, errInvalidBody
	}
	deviceType, ok := requiredString(fields, "type")
	if !ok {
		return model.CreateDeviceRequest{}, errInvalidBody
	}

	warrantyMonths, err := optionalInt(fields, "warranty_months")
	if err != nil {
		return model.CreateDeviceRequest{}, errInvalidBody
	}
	comment, err := optionalString(fields, "comment")
	if err != nil {
		return model.CreateDeviceRequest{}, errInvalidBody
	}

	return model.CreateDeviceRequest{
		Name:           name,
		Serial:         serial,
		Manufacturer:   manufacturer,
		Type:           deviceType,
		WarrantyMonths: warrantyMonths,
		Comment:        comment,
	}, nil
}

var errInvalidBody = &requestError{}

type requestError struct{}

func (*requestError) Error() string { return "invalid request body" }

func ensureNoTrailingJSON(r *http.Request) error {
	var extra any
	if err := json.NewDecoder(r.Body).Decode(&extra); err != io.EOF {
		return errInvalidBody
	}
	return nil
}

func requiredString(fields map[string]json.RawMessage, name string) (string, bool) {
	raw, ok := fields[name]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return "", false
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil || strings.TrimSpace(value) == "" {
		return "", false
	}
	return value, true
}

func optionalInt(fields map[string]json.RawMessage, name string) (*int, error) {
	raw, ok := fields[name]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}

	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, errInvalidBody
	}
	return &value, nil
}

func optionalString(fields map[string]json.RawMessage, name string) (*string, error) {
	raw, ok := fields[name]
	if !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, errInvalidBody
	}
	return &value, nil
}

func parseID(path string) (uint32, error) {
	if path == "" {
		return 0, strconv.ErrSyntax
	}
	for _, char := range path {
		if char < '0' || char > '9' {
			return 0, strconv.ErrSyntax
		}
	}

	id, err := strconv.ParseUint(path, 10, 32)
	return uint32(id), err
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, apiError{Error: errorBody{Code: code, Message: message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
