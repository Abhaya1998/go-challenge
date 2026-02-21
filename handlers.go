package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
)

// toAPIProduct converts a database product to the API representation.
func toAPIProduct(p *dbProduct) Product {
	return Product{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Price:       float64(p.PriceCents) / 100,
		Category:    p.Category,
		InStock:     p.InStock,
		Quantity:    p.Quantity,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		DeletedAt:   p.DeletedAt,
	}
}

func getIDFromPath(r *http.Request, prefix string) (int, error) {
	path := strings.TrimPrefix(r.URL.Path, prefix)
	path = strings.TrimSuffix(path, "/")
	path = strings.Split(path, "/")[0]
	return strconv.Atoi(path)
}

// handleListProducts handles GET /products
func (s *Server) handleListProducts(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")

	products, err := s.store.ListProducts(category)
	if err != nil {
		http.Error(w, "failed to list products", http.StatusInternalServerError)
		return
	}

	apiProducts := make([]Product, len(products))
	for i, p := range products {
		apiProducts[i] = toAPIProduct(&p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiProducts)
}

// handleCreateProduct handles POST /products
func (s *Server) handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	var req CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Price < 0 {
		log.Printf("ERROR: invalid price: %.2f", req.Price)
		http.Error(w, `{"error":"price must be non-negative"}`, http.StatusBadRequest)
		return
	}

	priceCents := int(math.Round(req.Price * 100))

	id, err := s.store.CreateProduct(req.Name, req.Description, priceCents, req.Category, req.InStock, req.Quantity)
	if err != nil {
		log.Printf("ERROR: failed to create product: %v", err)
		errMsg := err.Error()
		if errMsg == "name is required" || errMsg == "category is required" || errMsg == "price must be non-negative" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": errMsg})
			return
		}
		http.Error(w, "failed to create product", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

// handleGetProduct handles GET /products/:id
func (s *Server) handleGetProduct(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromPath(r, "/products/")
	if err != nil {
		http.Error(w, "invalid product ID", http.StatusBadRequest)
		return
	}

	product, err := s.store.GetProduct(id)
	if err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	apiProduct := toAPIProduct(product)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiProduct)
}

// handleUpdateProduct handles PUT /products/:id
func (s *Server) handleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromPath(r, "/products/")
	if err != nil {
		http.Error(w, "invalid product ID", http.StatusBadRequest)
		return
	}

	var update Product
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if update.Price < 0 {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error":"price must be non-negative"}`, http.StatusBadRequest)
		return
	}

	priceCents := int(math.Round(update.Price * 100))

	err = s.store.UpdateProduct(id, update.Name, update.Description, priceCents, update.Category, update.InStock, update.Quantity)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "name is required" || errMsg == "category is required" || errMsg == "price must be non-negative" || errMsg == "product not found" {
			w.Header().Set("Content-Type", "application/json")
			code := http.StatusBadRequest
			if errMsg == "product not found" {
				code = http.StatusNotFound
			}
			w.WriteHeader(code)
			json.NewEncoder(w).Encode(map[string]string{"error": errMsg})
			return
		}
		http.Error(w, "failed to update product", http.StatusInternalServerError)
		return
	}

	product, err := s.store.GetProduct(id)
	if err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toAPIProduct(product))
}

// handleDeleteProduct handles DELETE /products/:id
func (s *Server) handleDeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromPath(r, "/products/")
	if err != nil {
		http.Error(w, "invalid product ID", http.StatusBadRequest)
		return
	}

	err = s.store.DeleteProduct(id)
	if err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handlePurchaseProduct handles POST /products/:id/purchase
// Only for products without variants; products with variants must be purchased by variant.
func (s *Server) handlePurchaseProduct(w http.ResponseWriter, r *http.Request) {
	pathPart := strings.TrimPrefix(r.URL.Path, "/products/")
	idStr := strings.Split(pathPart, "/")[0]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid product ID", http.StatusBadRequest)
		return
	}

	product, err := s.store.GetProduct(id)
	if err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	if product.Quantity <= 0 {
		http.Error(w, `{"error":"out of stock"}`, http.StatusConflict)
		return
	}

	variants, err := s.store.ListVariants(id)
	if err == nil && len(variants) > 0 {
		http.Error(w, `{"error":"product has variants; purchase by variant"}`, http.StatusBadRequest)
		return
	}

	err = s.store.DecrementQuantity(id)
	if err != nil {
		http.Error(w, "purchase failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "purchased"})
}
