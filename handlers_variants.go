package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
)

// toAPIVariant converts a database variant to the API representation.
func toAPIVariant(v *dbVariant) Variant {
	attrs := make(map[string]string)
	if v.Attributes != "" && v.Attributes != "{}" {
		json.Unmarshal([]byte(v.Attributes), &attrs)
	}

	return Variant{
		ID:         v.ID,
		ProductID:  v.ProductID,
		SKU:        v.SKU,
		Name:       v.Name,
		Price:      float64(v.PriceCents) / 100,
		Quantity:   v.Quantity,
		InStock:    v.InStock,
		Attributes: attrs,
		SortOrder:  v.SortOrder,
		CreatedAt:  v.CreatedAt,
		UpdatedAt:  v.UpdatedAt,
	}
}

// handleListVariants handles GET /products/:id/variants
func (s *Server) handleListVariants(w http.ResponseWriter, r *http.Request) {
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

	variants, err := s.store.ListVariants(productID)
	if err != nil {
		http.Error(w, "failed to list variants", http.StatusInternalServerError)
		return
	}

	apiVariants := make([]Variant, len(variants))
	for i, v := range variants {
		apiVariants[i] = toAPIVariant(&v)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiVariants)
}

// handleCreateVariant handles POST /products/:id/variants
func (s *Server) handleCreateVariant(w http.ResponseWriter, r *http.Request) {
	pathPart := strings.TrimPrefix(r.URL.Path, "/products/")
	idStr := strings.Split(pathPart, "/")[0]
	productID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid product ID", http.StatusBadRequest)
		return
	}

	var req CreateVariantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.SKU == "" {
		http.Error(w, `{"error":"sku is required"}`, http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
		return
	}
	if req.Price < 0 {
		http.Error(w, `{"error":"price must be non-negative"}`, http.StatusBadRequest)
		return
	}

	priceCents := int(math.Round(req.Price * 100))

	attrsJSON := "{}"
	if req.Attributes != nil {
		data, err := json.Marshal(req.Attributes)
		if err != nil {
			http.Error(w, "invalid attributes", http.StatusBadRequest)
			return
		}
		attrsJSON = string(data)
	}

	id, err := s.store.CreateVariant(productID, req.SKU, req.Name, priceCents, req.Quantity, attrsJSON, req.SortOrder)
	if err != nil {
		if err.Error() == "product not found" {
			http.Error(w, "product not found", http.StatusNotFound)
			return
		}
		log.Printf("ERROR: failed to create variant: %v", err)
		http.Error(w, `{"error":"failed to create variant"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

// handleGetVariant handles GET /products/:id/variants/:variantId
func (s *Server) handleGetVariant(w http.ResponseWriter, r *http.Request) {
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
	variantID, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "invalid variant ID", http.StatusBadRequest)
		return
	}

	if _, err := s.store.GetProduct(productID); err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	variant, err := s.store.GetVariant(variantID)
	if err != nil {
		http.Error(w, "variant not found", http.StatusNotFound)
		return
	}
	if variant.ProductID != productID {
		http.Error(w, "variant not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toAPIVariant(variant))
}

// handleUpdateVariant handles PUT /products/:id/variants/:variantId
func (s *Server) handleUpdateVariant(w http.ResponseWriter, r *http.Request) {
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
	variantID, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "invalid variant ID", http.StatusBadRequest)
		return
	}

	if _, err := s.store.GetProduct(productID); err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}
	variant, err := s.store.GetVariant(variantID)
	if err != nil || variant.ProductID != productID {
		http.Error(w, "variant not found", http.StatusNotFound)
		return
	}

	var req UpdateVariantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	priceCents := int(math.Round(req.Price * 100))

	attrsJSON := "{}"
	if req.Attributes != nil {
		data, err := json.Marshal(req.Attributes)
		if err != nil {
			http.Error(w, "invalid attributes", http.StatusBadRequest)
			return
		}
		attrsJSON = string(data)
	}

	err = s.store.UpdateVariant(variantID, req.SKU, req.Name, priceCents, req.Quantity, req.InStock, attrsJSON, req.SortOrder)
	if err != nil {
		http.Error(w, "failed to update variant", http.StatusInternalServerError)
		return
	}

	variant, err = s.store.GetVariant(variantID)
	if err != nil {
		http.Error(w, "variant not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toAPIVariant(variant))
}

// handleDeleteVariant handles DELETE /products/:id/variants/:variantId
func (s *Server) handleDeleteVariant(w http.ResponseWriter, r *http.Request) {
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
	variantID, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "invalid variant ID", http.StatusBadRequest)
		return
	}

	if _, err := s.store.GetProduct(productID); err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}
	variant, err := s.store.GetVariant(variantID)
	if err != nil || variant.ProductID != productID {
		http.Error(w, "variant not found", http.StatusNotFound)
		return
	}

	err = s.store.DeleteVariant(variantID)
	if err != nil {
		http.Error(w, "variant not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handlePurchaseVariant handles POST /products/:id/variants/:variantId/purchase
func (s *Server) handlePurchaseVariant(w http.ResponseWriter, r *http.Request) {
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
	variantID, err := strconv.Atoi(parts[2])
	if err != nil {
		http.Error(w, "invalid variant ID", http.StatusBadRequest)
		return
	}

	if _, err := s.store.GetProduct(productID); err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	variant, err := s.store.GetVariant(variantID)
	if err != nil {
		http.Error(w, "variant not found", http.StatusNotFound)
		return
	}
	if variant.ProductID != productID {
		http.Error(w, "variant not found", http.StatusNotFound)
		return
	}

	if variant.Quantity <= 0 {
		http.Error(w, `{"error":"variant out of stock"}`, http.StatusConflict)
		return
	}

	err = s.store.DecrementVariantQuantity(variantID)
	if err != nil {
		http.Error(w, "purchase failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "purchased"})
}

// handleGetVariantInventory handles GET /products/:id/inventory
func (s *Server) handleGetVariantInventory(w http.ResponseWriter, r *http.Request) {
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

	inv, err := s.store.GetVariantInventory(productID)
	if err != nil {
		http.Error(w, "failed to get inventory", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(inv)
}

// handleLookupBySKU handles GET /sku/:sku
func (s *Server) handleLookupBySKU(w http.ResponseWriter, r *http.Request) {
	sku := strings.TrimPrefix(r.URL.Path, "/sku/")
	sku = strings.TrimSuffix(sku, "/")

	if sku == "" {
		http.Error(w, "sku is required", http.StatusBadRequest)
		return
	}

	variant, err := s.store.GetVariantBySKU(sku)
	if err != nil {
		http.Error(w, "variant not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toAPIVariant(variant))
}
