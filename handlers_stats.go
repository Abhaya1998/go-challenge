package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

const appVersion = "1.0.0"

// handleHealthCheck handles GET /health
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	dbStatus := "ok"
	if err := s.store.db.Ping(); err != nil {
		dbStatus = "error: " + err.Error()
	}

	uptime := time.Since(s.startTime).Round(time.Second)

	status := HealthStatus{
		Status:   "ok",
		Database: dbStatus,
		Uptime:   uptime.String(),
		Version:  appVersion,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleGetStats handles GET /stats (JSON)
func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	total, inStock, outOfStock, err := s.store.GetProductCount()
	if err != nil {
		http.Error(w, "failed to get product counts", http.StatusInternalServerError)
		return
	}

	avgPrice, err := s.store.GetAverageProductPrice()
	if err != nil {
		http.Error(w, "failed to get average price", http.StatusInternalServerError)
		return
	}

	totalInventory, err := s.store.GetTotalInventory()
	if err != nil {
		http.Error(w, "failed to get inventory", http.StatusInternalServerError)
		return
	}

	totalReviews, err := s.store.GetTotalReviewCount()
	if err != nil {
		http.Error(w, "failed to get review count", http.StatusInternalServerError)
		return
	}

	categories, err := s.store.GetCategoryStats()
	if err != nil {
		http.Error(w, "failed to get category stats", http.StatusInternalServerError)
		return
	}

	stats := DashboardStats{
		TotalProducts:   total,
		TotalInStock:    inStock,
		TotalOutOfStock: outOfStock,
		AveragePrice:    avgPrice / 100,
		TotalInventory:  totalInventory,
		TotalReviews:    totalReviews,
		Categories:      categories,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleSearchProducts handles GET /search?q=...
func (s *Server) handleSearchProducts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, `{"error":"query parameter 'q' is required"}`, http.StatusBadRequest)
		return
	}

	products, err := s.store.SearchProducts(query)
	if err != nil {
		http.Error(w, "search failed", http.StatusInternalServerError)
		return
	}

	apiProducts := make([]Product, len(products))
	for i, p := range products {
		apiProducts[i] = toAPIProduct(&p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiProducts)
}

// handleListCategories handles GET /categories
func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := s.store.ListCategories()
	if err != nil {
		http.Error(w, "failed to list categories", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(categories)
}

// handleGetAuditLog handles GET /audit
func (s *Server) handleGetAuditLog(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}

	productIDStr := r.URL.Query().Get("product_id")
	if productIDStr != "" {
		productID, err := strconv.Atoi(productIDStr)
		if err != nil {
			http.Error(w, "invalid product_id", http.StatusBadRequest)
			return
		}
		entries, err := s.store.GetAuditLog(productID)
		if err != nil {
			http.Error(w, "failed to get audit log", http.StatusInternalServerError)
			return
		}
		if limit > 0 && len(entries) > limit {
			entries = entries[:limit]
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
		return
	}

	entries, err := s.store.GetRecentAuditLog(limit)
	if err != nil {
		http.Error(w, "failed to get audit log", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

// handlePageStats renders the stats dashboard page.
func (s *Server) handlePageStats(w http.ResponseWriter, r *http.Request) {
	total, inStock, outOfStock, err := s.store.GetProductCount()
	if err != nil {
		http.Error(w, "failed to get stats", http.StatusInternalServerError)
		return
	}

	avgPrice, err := s.store.GetAverageProductPrice()
	if err != nil {
		http.Error(w, "failed to get average price", http.StatusInternalServerError)
		return
	}

	totalInventory, err := s.store.GetTotalInventory()
	if err != nil {
		http.Error(w, "failed to get inventory", http.StatusInternalServerError)
		return
	}

	totalReviews, err := s.store.GetTotalReviewCount()
	if err != nil {
		http.Error(w, "failed to get review count", http.StatusInternalServerError)
		return
	}

	categories, err := s.store.GetCategoryStats()
	if err != nil {
		http.Error(w, "failed to get categories", http.StatusInternalServerError)
		return
	}

	tmpl, err := s.loadTemplate("layout.html", "stats.html")
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":           "Dashboard",
		"TotalProducts":   total,
		"TotalInStock":    inStock,
		"TotalOutOfStock": outOfStock,
		"AveragePrice":    avgPrice / 100,
		"TotalInventory":  totalInventory,
		"TotalReviews":    totalReviews,
		"Categories":      categories,
	}

	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
	}
}
