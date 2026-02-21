package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// toAPIReview converts a database review to the API representation.
func toAPIReview(r *dbReview) Review {
	return Review{
		ID:        r.ID,
		ProductID: r.ProductID,
		Author:    r.Author,
		Rating:    r.Rating,
		Comment:   r.Comment,
		Approved:  r.Approved,
		CreatedAt: r.CreatedAt,
	}
}

// handleListReviews handles GET /products/:id/reviews
func (s *Server) handleListReviews(w http.ResponseWriter, r *http.Request) {
	pathPart := strings.TrimPrefix(r.URL.Path, "/products/")
	idStr := strings.Split(pathPart, "/")[0]
	productID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid product ID", http.StatusBadRequest)
		return
	}

	if _, err := s.store.GetProduct(productID); err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	reviews, err := s.store.ListReviews(productID)
	if err != nil {
		http.Error(w, "failed to list reviews", http.StatusInternalServerError)
		return
	}

	apiReviews := make([]Review, len(reviews))
	for i, r := range reviews {
		apiReviews[i] = toAPIReview(&r)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiReviews)
}

// handleCreateReview handles POST /products/:id/reviews
func (s *Server) handleCreateReview(w http.ResponseWriter, r *http.Request) {
	pathPart := strings.TrimPrefix(r.URL.Path, "/products/")
	idStr := strings.Split(pathPart, "/")[0]
	productID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid product ID", http.StatusBadRequest)
		return
	}

	var req CreateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Author == "" {
		http.Error(w, `{"error":"author is required"}`, http.StatusBadRequest)
		return
	}
	if req.Rating < 1 || req.Rating > 5 {
		http.Error(w, `{"error":"rating must be between 1 and 5"}`, http.StatusBadRequest)
		return
	}

	id, err := s.store.CreateReview(productID, req.Author, req.Rating, req.Comment)
	if err != nil {
		if err.Error() == "product not found" {
			http.Error(w, "product not found", http.StatusNotFound)
			return
		}
		log.Printf("ERROR: failed to create review: %v", err)
		http.Error(w, `{"error":"failed to create review"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

// handleDeleteReview handles DELETE /products/:id/reviews/:reviewId
func (s *Server) handleDeleteReview(w http.ResponseWriter, r *http.Request) {
	pathPart := strings.TrimPrefix(r.URL.Path, "/products/")
	parts := strings.Split(pathPart, "/")
	if len(parts) < 3 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	productID, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, "invalid product ID", http.StatusBadRequest)
		return
	}
	reviewID, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "invalid review ID", http.StatusBadRequest)
		return
	}

	if _, err := s.store.GetProduct(productID); err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}
	review, err := s.store.GetReview(reviewID)
	if err != nil || review.ProductID != productID {
		http.Error(w, "review not found", http.StatusNotFound)
		return
	}

	err = s.store.DeleteReview(reviewID)
	if err != nil {
		http.Error(w, "review not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleApproveReview handles POST /products/:id/reviews/:reviewId/approve
func (s *Server) handleApproveReview(w http.ResponseWriter, r *http.Request) {
	pathPart := strings.TrimPrefix(r.URL.Path, "/products/")
	parts := strings.Split(pathPart, "/")
	if len(parts) < 3 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	productID, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, "invalid product ID", http.StatusBadRequest)
		return
	}
	reviewID, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "invalid review ID", http.StatusBadRequest)
		return
	}

	if _, err := s.store.GetProduct(productID); err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}
	review, err := s.store.GetReview(reviewID)
	if err != nil || review.ProductID != productID {
		http.Error(w, "review not found", http.StatusNotFound)
		return
	}

	err = s.store.ApproveReview(reviewID)
	if err != nil {
		http.Error(w, "review not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "approved"})
}

// handleGetProductWithReviews handles GET /products/:id/details
func (s *Server) handleGetProductWithReviews(w http.ResponseWriter, r *http.Request) {
	pathPart := strings.TrimPrefix(r.URL.Path, "/products/")
	idStr := strings.Split(pathPart, "/")[0]
	productID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid product ID", http.StatusBadRequest)
		return
	}

	product, err := s.store.GetProduct(productID)
	if err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	reviews, err := s.store.ListReviews(productID)
	if err != nil {
		http.Error(w, "failed to load reviews", http.StatusInternalServerError)
		return
	}

	avgRating, reviewCount, err := s.store.GetAverageRating(productID)
	if err != nil {
		http.Error(w, "failed to compute rating", http.StatusInternalServerError)
		return
	}

	apiReviews := make([]Review, len(reviews))
	for i, r := range reviews {
		apiReviews[i] = toAPIReview(&r)
	}

	result := ProductWithReviews{
		Product:       toAPIProduct(product),
		Reviews:       apiReviews,
		AverageRating: avgRating,
		ReviewCount:   reviewCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
