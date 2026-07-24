package hostasset

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Handler struct {
	repository Repository
}

// NewHandler 创建并初始化 New Handler 实例。
func NewHandler(repository Repository) *Handler {
	return &Handler{repository: repository}
}

// Create 创建新的 Create。
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var request HostAsset
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if request.HostID == "" && request.NodeName == "" {
		http.Error(w, "hostId is required", http.StatusBadRequest)
		return
	}
	if request.HostName == "" && request.DisplayName == "" {
		http.Error(w, "hostName is required", http.StatusBadRequest)
		return
	}
	created, err := h.repository.Create(r.Context(), request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// List 查询并返回 List 列表。
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	assets, err := h.repository.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(assets)
}

// Get 查询并返回指定的 Get。
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	asset, err := h.repository.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(asset)
}

// Update 更新指定的 Update。
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var request HostAsset
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if request.HostID == "" && request.NodeName == "" {
		http.Error(w, "hostId is required", http.StatusBadRequest)
		return
	}
	if request.HostName == "" && request.DisplayName == "" {
		http.Error(w, "hostName is required", http.StatusBadRequest)
		return
	}
	updated, err := h.repository.Update(r.Context(), r.PathValue("id"), request)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(updated)
}

// Delete 删除指定的 Delete。
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.repository.Delete(r.Context(), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeError 写入 write Error 数据。
func writeError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNotFound) {
		http.Error(w, "host asset not found", http.StatusNotFound)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
