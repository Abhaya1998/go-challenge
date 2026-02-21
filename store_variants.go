package main

import (
	"fmt"
	"time"
)

// createVariantTable creates the variants table if it doesn't exist.
func createVariantTable(s *Store) error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS variants (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			product_id INTEGER NOT NULL,
			sku TEXT NOT NULL,
			name TEXT NOT NULL,
			price_cents INTEGER DEFAULT 0,
			quantity INTEGER DEFAULT 0,
			in_stock BOOLEAN DEFAULT 1,
			attributes TEXT DEFAULT '{}',
			sort_order INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(sku),
			FOREIGN KEY (product_id) REFERENCES products(id)
		)
	`)
	return err
}

// CreateVariant inserts a new variant for a product.
func (s *Store) CreateVariant(productID int, sku, name string, priceCents, quantity int, attributes string, sortOrder int) (int, error) {
	if sku == "" {
		return 0, fmt.Errorf("sku is required")
	}
	if name == "" {
		return 0, fmt.Errorf("name is required")
	}

	// Verify product exists.
	_, err := s.GetProduct(productID)
	if err != nil {
		return 0, fmt.Errorf("product not found")
	}

	inStock := quantity > 0
	now := time.Now().UTC()

	result, err := s.db.Exec(
		`INSERT INTO variants (product_id, sku, name, price_cents, quantity, in_stock, attributes, sort_order, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		productID, sku, name, priceCents, quantity, inStock, attributes, sortOrder, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert variant: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

// ListVariants returns all variants for a product, ordered by sort_order.
func (s *Store) ListVariants(productID int) ([]dbVariant, error) {
	rows, err := s.db.Query(
		`SELECT id, product_id, sku, name, price_cents, quantity, in_stock, attributes, sort_order, created_at, updated_at
		 FROM variants WHERE product_id = ? ORDER BY sort_order ASC, id ASC`,
		productID,
	)
	if err != nil {
		return nil, fmt.Errorf("list variants: %w", err)
	}
	defer rows.Close()

	var variants []dbVariant
	for rows.Next() {
		var v dbVariant
		err := rows.Scan(&v.ID, &v.ProductID, &v.SKU, &v.Name, &v.PriceCents,
			&v.Quantity, &v.InStock, &v.Attributes, &v.SortOrder, &v.CreatedAt, &v.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan variant: %w", err)
		}
		variants = append(variants, v)
	}
	return variants, rows.Err()
}

// GetVariant returns a single variant by ID.
func (s *Store) GetVariant(variantID int) (*dbVariant, error) {
	var v dbVariant
	err := s.db.QueryRow(
		`SELECT id, product_id, sku, name, price_cents, quantity, in_stock, attributes, sort_order, created_at, updated_at
		 FROM variants WHERE id = ?`,
		variantID,
	).Scan(&v.ID, &v.ProductID, &v.SKU, &v.Name, &v.PriceCents,
		&v.Quantity, &v.InStock, &v.Attributes, &v.SortOrder, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// GetVariantBySKU looks up a variant by its SKU code.
func (s *Store) GetVariantBySKU(sku string) (*dbVariant, error) {
	var v dbVariant
	err := s.db.QueryRow(
		`SELECT id, product_id, sku, name, price_cents, quantity, in_stock, attributes, sort_order, created_at, updated_at
		 FROM variants WHERE sku = ?`,
		sku,
	).Scan(&v.ID, &v.ProductID, &v.SKU, &v.Name, &v.PriceCents,
		&v.Quantity, &v.InStock, &v.Attributes, &v.SortOrder, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// UpdateVariant updates fields for a variant.
func (s *Store) UpdateVariant(variantID int, sku, name string, priceCents, quantity int, inStock bool, attributes string, sortOrder int) error {
	now := time.Now().UTC()
	result, err := s.db.Exec(
		`UPDATE variants SET sku = ?, name = ?, price_cents = ?, quantity = ?, in_stock = ?, attributes = ?, sort_order = ?, updated_at = ?
		 WHERE id = ?`,
		sku, name, priceCents, quantity, inStock, attributes, sortOrder, now, variantID,
	)
	if err != nil {
		return fmt.Errorf("update variant: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("variant not found")
	}
	return nil
}

// DeleteVariant removes a variant by ID.
func (s *Store) DeleteVariant(variantID int) error {
	result, err := s.db.Exec(`DELETE FROM variants WHERE id = ?`, variantID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("variant not found")
	}
	return nil
}

// DeleteVariantsByProduct removes all variants for a product.
func (s *Store) DeleteVariantsByProduct(productID int) error {
	_, err := s.db.Exec(`DELETE FROM variants WHERE product_id = ?`, productID)
	return err
}

// DecrementVariantQuantity decreases a variant's quantity by 1.
// Returns error if variant not found or quantity is already 0 (atomic, prevents negative stock).
func (s *Store) DecrementVariantQuantity(variantID int) error {
	now := time.Now().UTC()
	result, err := s.db.Exec(
		`UPDATE variants SET quantity = quantity - 1, in_stock = (quantity > 1), updated_at = ?
		 WHERE id = ? AND quantity > 0`,
		now, variantID,
	)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("variant not found or out of stock")
	}
	return nil
}

// GetVariantInventory returns an inventory summary for a product's variants.
func (s *Store) GetVariantInventory(productID int) (*VariantInventory, error) {
	var inv VariantInventory
	inv.ProductID = productID

	err := s.db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(quantity), 0), SUM(CASE WHEN in_stock = 1 THEN 1 ELSE 0 END)
		 FROM variants WHERE product_id = ?`,
		productID,
	).Scan(&inv.VariantCount, &inv.TotalStock, &inv.InStockCount)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// BulkUpdateVariantPrices updates price for all variants of a product by a percentage.
func (s *Store) BulkUpdateVariantPrices(productID int, multiplier float64) (int, error) {
	now := time.Now().UTC()
	result, err := s.db.Exec(
		`UPDATE variants SET price_cents = CAST(price_cents * ? AS INTEGER), updated_at = ?
		 WHERE product_id = ?`,
		multiplier, now, productID,
	)
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(rows), nil
}
