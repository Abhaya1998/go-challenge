package main

import (
	"net/http"
	"strings"
	"time"
)

type Server struct {
	store     *Store
	router    http.Handler
	startTime time.Time
}

func NewServer(store *Store) *Server {
	s := &Server{
		store:     store,
		startTime: time.Now(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	mux := http.NewServeMux()

	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Page routes (HTML)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			s.handlePageProductList(w, r)
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/new", s.handlePageNewProduct)
	mux.HandleFunc("/stats", s.handlePageStats)

	// Health check
	mux.HandleFunc("/health", s.handleHealthCheck)

	// Search
	mux.HandleFunc("/search", s.handleSearchProducts)

	// Categories
	mux.HandleFunc("/categories", s.handleListCategories)

	// Audit log
	mux.HandleFunc("/audit", s.handleGetAuditLog)

	// SKU lookup
	mux.HandleFunc("/sku/", s.handleLookupBySKU)

	// API routes for /products
	mux.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			s.handleListProducts(w, r)
		case http.MethodPost:
			s.handleCreateProduct(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Export/Import routes
	mux.HandleFunc("/products/export", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			s.handleExportCSV(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("/products/export/json", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			s.handleExportJSON(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("/products/import", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			s.handleImportCSV(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	// Stats API
	mux.HandleFunc("/products/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			s.handleGetStats(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/products/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/products/")

		// Handle /products/:id/reviews and /products/:id/reviews/:reviewId (before /purchase so path is not matched by suffix)
		if strings.Contains(path, "/reviews") {
			s.routeReviews(w, r, path)
			return
		}

		// Handle /products/:id/variants and /products/:id/variants/:variantId/purchase (before /purchase)
		if strings.Contains(path, "/variants") {
			s.routeVariants(w, r, path)
			return
		}

		// Handle /products/:id/purchase (product-level only; variant purchase is above)
		if strings.HasSuffix(path, "/purchase") {
			if r.Method == http.MethodPost {
				s.handlePurchaseProduct(w, r)
				return
			}
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Handle /products/:id/inventory
		if strings.HasSuffix(path, "/inventory") {
			if r.Method == http.MethodGet {
				s.handleGetVariantInventory(w, r)
				return
			}
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Handle /products/:id/details
		if strings.HasSuffix(path, "/details") {
			if r.Method == http.MethodGet {
				s.handleGetProductWithReviews(w, r)
				return
			}
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check Accept header to decide: HTML page or JSON API
		accept := r.Header.Get("Accept")
		wantsHTML := strings.Contains(accept, "text/html")

		switch r.Method {
		case http.MethodGet:
			if wantsHTML {
				s.handlePageProductDetail(w, r)
			} else {
				s.handleGetProduct(w, r)
			}
		case http.MethodPut:
			s.handleUpdateProduct(w, r)
		case http.MethodDelete:
			s.handleDeleteProduct(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Apply middleware
	rl := newRateLimiter(100, time.Minute)
	s.router = chain(mux, recoveryMiddleware, loggingMiddleware, corsMiddleware, rl.middleware)
}

// routeReviews dispatches review sub-routes.
func (s *Server) routeReviews(w http.ResponseWriter, r *http.Request, path string) {
	// path is like "1/reviews" or "1/reviews/5" or "1/reviews/5/approve"
	if strings.HasSuffix(path, "/approve") {
		if r.Method == http.MethodPost {
			s.handleApproveReview(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(path, "/")
	// parts[0] = id, parts[1] = "reviews", parts[2] = reviewId (optional)

	if len(parts) == 2 {
		// /products/:id/reviews
		switch r.Method {
		case http.MethodGet:
			s.handleListReviews(w, r)
		case http.MethodPost:
			s.handleCreateReview(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	if len(parts) >= 3 {
		// /products/:id/reviews/:reviewId
		switch r.Method {
		case http.MethodDelete:
			s.handleDeleteReview(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	http.NotFound(w, r)
}

// routeVariants dispatches variant sub-routes.
func (s *Server) routeVariants(w http.ResponseWriter, r *http.Request, path string) {
	// path is like "1/variants" or "1/variants/5" or "1/variants/5/purchase"
	if strings.HasSuffix(path, "/purchase") {
		if r.Method == http.MethodPost {
			s.handlePurchaseVariant(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(path, "/")
	// parts[0] = id, parts[1] = "variants", parts[2] = variantId (optional)

	if len(parts) == 2 {
		// /products/:id/variants
		switch r.Method {
		case http.MethodGet:
			s.handleListVariants(w, r)
		case http.MethodPost:
			s.handleCreateVariant(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	if len(parts) >= 3 {
		// /products/:id/variants/:variantId
		switch r.Method {
		case http.MethodGet:
			s.handleGetVariant(w, r)
		case http.MethodPut:
			s.handleUpdateVariant(w, r)
		case http.MethodDelete:
			s.handleDeleteVariant(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	http.NotFound(w, r)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
