package main

import (
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

var funcMap = template.FuncMap{
	"div": func(a float64, b float64) float64 {
		if b == 0 {
			return 0
		}
		return a / b
	},
}

func (s *Server) loadTemplate(names ...string) (*template.Template, error) {
	paths := make([]string, len(names))
	for i, name := range names {
		paths[i] = filepath.Join("templates", name)
	}
	return template.New(names[0]).Funcs(funcMap).ParseFiles(paths...)
}

// handlePageProductList renders the product list page at /
func (s *Server) handlePageProductList(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	products, err := s.store.ListProducts("")
	if err != nil {
		http.Error(w, "failed to load products", http.StatusInternalServerError)
		return
	}

	apiProducts := make([]Product, len(products))
	for i, p := range products {
		apiProducts[i] = toAPIProduct(&p)
	}

	tmpl, err := s.loadTemplate("layout.html", "product_list.html")
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Products": apiProducts,
		"Title":    "Products",
	}

	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handlePageNewProduct renders the create product form at /new
func (s *Server) handlePageNewProduct(w http.ResponseWriter, r *http.Request) {
	tmpl, err := s.loadTemplate("layout.html", "new_product.html")
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title": "New Product",
	}

	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
	}
}

// handlePageProductDetail renders the product detail page at /products/:id
func (s *Server) handlePageProductDetail(w http.ResponseWriter, r *http.Request) {
	pathPart := strings.TrimPrefix(r.URL.Path, "/products/")
	pathPart = strings.TrimSuffix(pathPart, "/")

	if strings.Contains(pathPart, "/") {
		http.NotFound(w, r)
		return
	}

	id, err := strconv.Atoi(pathPart)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	product, err := s.store.GetProduct(id)
	if err != nil {
		http.Error(w, "product not found", http.StatusNotFound)
		return
	}

	apiProduct := toAPIProduct(product)

	variants, err := s.store.ListVariants(id)
	if err != nil {
		variants = nil
	}
	apiVariants := make([]Variant, len(variants))
	for i, v := range variants {
		apiVariants[i] = toAPIVariant(&v)
	}

	tmpl, err := s.loadTemplate("layout.html", "product_detail.html")
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Product":  apiProduct,
		"Variants": apiVariants,
		"Title":    apiProduct.Name,
	}

	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "render error: "+err.Error(), http.StatusInternalServerError)
	}
}
