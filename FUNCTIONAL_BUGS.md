# Functional Bugs – Fix Log

Each entry: **Defect** | **Root cause** | **Fix summary** | **Code change** | **Validation**

---

## Summary

| # | Area | One-line fix |
|---|------|--------------|
| 1 | Price | toAPIProduct: `/ 100` not `* 100` |
| 2 | Cache | Invalidate cache on update/delete/decrement |
| 3 | List | `WHERE deleted_at IS NULL`; param category |
| 4 | Security | Parameterized category in ListProducts |
| 5 | Create | Return + 500 on CreateProduct error |
| 6 | Purchase | Atomic decrement with `quantity > 0` |
| 7 | Detail | NotFound for bad path |
| 8 | UI | DELETE method for delete |
| 9 | Audit | Apply limit when product_id set |
| 10 | Stats | Category avg: `/ 100.0` in SQL |
| 11 | List price | Template: no div by 100 |
| 12 | Qty | Product qty = sum(variants) when has variants |
| 13 | Variant purchase | No main Purchase if variants; update product qty in UI |
| 14 | Routing | Check /variants before /purchase |
| 15 | Negative price | Seed fix + abs() on startup |
| 16 | UI (product) | Stock status + hide Purchase when qty 0 (no variants) |
| 17 | UI (variant row) | Stock badge + hide Buy when variant qty 0 |
| 18 | Logic | Product in_stock only if sum(variant qty) ≥ 1 (#12) |
| 19 | Create product | Category required; 400 + JSON error; form required + toast |
| 20 | Dashboard | In-stock count + total inventory use variant-sum logic (GetProductCount, GetTotalInventory) |
| 21 | Search | SearchProducts use variant-sum for quantity/in_stock |
| 22 | Update product | UpdateProduct validate name, category, price ≥ 0; handler 400/404 |
| 23 | Reviews/variants | 404 when product deleted or missing (list, create, inventory) |
| 24 | Sub-resources | Variant/review handlers verify product exists and resource belongs to product |

---

## 1. Product/detail price 100× too large

| | |
|---|---|
| **Defect** | Product list and detail show price as e.g. 249900 instead of 24.99. |
| **Root cause** | `toAPIProduct` used `* 100` instead of `/ 100` when converting cents to dollars. |
| **Fix** | Convert cents to dollars with divide by 100. |
| **Code change** | `handlers.go`: `Price: float64(p.PriceCents) / 100` (was `* 100`). |
| **Validation** | GET /products and product detail page show correct dollar prices (e.g. $24.99). |

---

## 2. Stale product after update/delete; stock wrong after purchase

| | |
|---|---|
| **Defect** | After update/delete, product API and UI show old data. After purchase, detail page can still show “in stock”. |
| **Root cause** | `GetProduct` caches for 3s; cache never cleared on `UpdateProduct`, `DeleteProduct`, or `DecrementQuantity`. |
| **Fix** | Invalidate product cache on update, delete, and after decrement. |
| **Code change** | `store.go`: Added `invalidateProductCache(id)`; call it from `UpdateProduct`, `DeleteProduct`, `DecrementQuantity`. |
| **Validation** | Update/delete product then GET /products/:id shows new data; purchase then refresh shows correct stock. |

---

## 3. Deleted products still in list/export

| | |
|---|---|
| **Defect** | Soft-deleted products still appear in product list and export. |
| **Root cause** | `ListProducts("")` had no `WHERE deleted_at IS NULL`. |
| **Fix** | Always filter by `deleted_at IS NULL`; use parameterized category. |
| **Code change** | `store.go`: ListProducts uses `WHERE deleted_at IS NULL`, optional `AND category = ?` with args, `ORDER BY p.id`. Export/CSV and export/JSON use ListProducts so they also exclude deleted. |
| **Validation** | Delete a product; list and export (CSV/JSON) no longer include it. |

---

## 4. SQL injection via category filter

| | |
|---|---|
| **Defect** | Category from query string was concatenated into SQL. |
| **Root cause** | `fmt.Sprintf(..., category)` built the WHERE clause. |
| **Fix** | Use parameterized query: `AND category = ?` with `args`. |
| **Code change** | `store.go`: ListProducts uses `?` placeholder and `args = append(args, category)`. |
| **Validation** | `GET /products?category=electronics'` (or malicious payload) no longer alters query. |

---

## 5. Create product returns 201 and id:0 on failure

| | |
|---|---|
| **Defect** | On CreateProduct error, client receives 201 Created and `{"id":0}`. |
| **Root cause** | Handler logged error but did not `return`; always wrote 201 and body. |
| **Fix** | On error, send 500 and return without writing success body. |
| **Code change** | `handlers.go`: After `CreateProduct` error, `http.Error(..., 500)` then `return`. |
| **Validation** | Force create failure (e.g. duplicate name); response is 500, no 201. |

---

## 6. Product/variant quantity can go negative; race on purchase

| | |
|---|---|
| **Defect** | Under concurrent purchases or when already 0, quantity becomes negative. |
| **Root cause** | TOCTOU: check then decrement; store did SELECT then UPDATE with no `quantity > 0` guard. |
| **Fix** | Atomic UPDATE with `WHERE quantity > 0`; return error if RowsAffected == 0. |
| **Code change** | `store.go`: `DecrementQuantity` uses `UPDATE ... quantity = quantity - 1 ... WHERE id = ? AND deleted_at IS NULL AND quantity > 0`; check RowsAffected. `store_variants.go`: Same pattern for `DecrementVariantQuantity`. |
| **Validation** | Purchase until 0 then purchase again → 409/error; DB quantity never negative. |

---

## 7. Product detail URL with extra path returns empty 200

| | |
|---|---|
| **Defect** | e.g. `/products/1/foo` returns empty body, no status. |
| **Root cause** | Handler `return`ed without writing response when path contained `/`. |
| **Fix** | Respond with 404. |
| **Code change** | `handlers_pages.go`: `http.NotFound(w, r)` before `return` when path contains `/`. |
| **Validation** | GET /products/1/foo returns 404. |

---

## 8. UI Delete does not delete product

| | |
|---|---|
| **Defect** | Delete button shows “Deleted!” but product remains. |
| **Root cause** | `deleteProduct()` used `method: 'GET'`; backend expects `DELETE`. |
| **Fix** | Use DELETE method. |
| **Code change** | `static/app.js`: `method: 'DELETE'` in fetch for delete. |
| **Validation** | Click Delete; product disappears from list and GET /products/:id returns 404. |

---

## 9. Audit API ignores limit when product_id set

| | |
|---|---|
| **Defect** | `GET /audit?product_id=1&limit=10` returns all entries for product. |
| **Root cause** | `limit` only applied in `GetRecentAuditLog`; `GetAuditLog(productID)` returned full list. |
| **Fix** | Apply limit to product-specific audit response. |
| **Code change** | `handlers_stats.go`: After `GetAuditLog(productID)`, if `limit > 0` slice `entries = entries[:limit]`. |
| **Validation** | GET /audit?product_id=1&limit=2 returns at most 2 entries. |

---

## 10. Category average price 100× too large on dashboard

| | |
|---|---|
| **Defect** | Stats/dashboard category average price shows e.g. 2499 instead of 24.99. |
| **Root cause** | `GetCategoryStats` used `AVG(price_cents)` and exposed as dollars without conversion. |
| **Fix** | Convert to dollars in SQL. |
| **Code change** | `store_stats.go`: `AVG(price_cents) / 100.0` in GetCategoryStats SELECT. |
| **Validation** | GET /products/stats and /stats page show category average in dollars. |

---

## 11. Product list page price wrong (double division)

| | |
|---|---|
| **Defect** | Main product list shows e.g. $0.25 instead of $24.99. |
| **Root cause** | Backend already sends Price in dollars; template did `(div .Price 100)` again. |
| **Fix** | Display .Price as-is (already dollars). |
| **Code change** | `templates/product_list.html`: `${{printf "%.2f" .Price}}` (removed `(div .Price 100)`). |
| **Validation** | Product list shows correct dollar prices. |

---

## 12. Main product quantity not sum of variants

| | |
|---|---|
| **Defect** | Product quantity/stock should reflect sum of variant quantities when product has variants. |
| **Root cause** | List/Get used product row quantity only. |
| **Fix** | For products with variants, use sum(variant quantities) and derived in_stock. |
| **Code change** | `store.go`: ListProducts and GetProduct use LEFT JOIN to variant sum; `quantity = COALESCE(v.tot, p.quantity)`, `in_stock = (quantity > 0)`. Product is in stock only if this quantity ≥ 1. |
| **Validation** | Product with variants shows quantity = sum of variant quantities; in_stock only when sum ≥ 1. |

---

## 13. Purchase by variant: UI and rules

| | |
|---|---|
| **Defect** | When product has variants, main Purchase button should be hidden; buying a variant should update main product quantity/stock on page. |
| **Root cause** | Template showed main Purchase for all in-stock products; variant purchase only updated variant row in DOM. |
| **Fix** | Show main Purchase only when product has no variants; on variant purchase success, decrement main product quantity in DOM and set “Out of Stock” when 0. |
| **Code change** | `templates/product_detail.html`: Purchase button `{{if and .Product.InStock (not .Variants)}}`; variant row ids `variant-stock-{{.ID}}`, `variant-buy-wrap-{{.ID}}`. `static/app.js`: `purchaseVariant` updates `#product-quantity` and `#product-stock-status` when variant purchased. `handlers.go`: handlePurchaseProduct returns 400 “purchase by variant” when product has variants. |
| **Validation** | Product with variants: no main Purchase button; buy variant → product quantity and stock status update; product in stock only if sum(variant qty) ≥ 1. |

---

## 14. Variant purchase returns “product has variants, purchase by variant”

| | |
|---|---|
| **Defect** | POST /products/1/variants/5/purchase incorrectly handled as product purchase and rejected. |
| **Root cause** | Route checked “path ends with /purchase” before “path contains /variants”, so variant purchase hit product purchase handler. |
| **Fix** | Check /variants (and /reviews) before the generic /purchase check. |
| **Code change** | `server.go`: Move “path contains /reviews” and “path contains /variants” above “path ends with /purchase”. |
| **Validation** | Click Buy on a variant → success; no “purchase by variant” error. |

---

## 15. Negative product price (e.g. Notebook Pack)

| | |
|---|---|
| **Defect** | One product (Notebook Pack) showed negative price. |
| **Root cause** | Seed had `priceCents: -500`; existing DBs kept negative value. |
| **Fix** | Fix seed to positive; on startup correct any negative prices to absolute value. |
| **Code change** | `store.go`: Seed “Notebook Pack” `-500` → `499`. After seedData, `UPDATE products SET price_cents = abs(price_cents) WHERE price_cents < 0`; same for variants. |
| **Validation** | New DB: Notebook Pack shows $4.99. Existing DB: negative prices become positive (e.g. -500 → 500 → $5.00). |

---

## 16. Main product (no variants): stock status and Purchase button when qty reaches 0

| | |
|---|---|
| **Defect** | After buying the last item on a product page (no variants), quantity shows 0 but status stays “In Stock” and Purchase button remains. |
| **Root cause** | `purchaseProduct()` only updated the quantity span in the DOM. |
| **Fix** | When newQty === 0, set stock status to “Out of Stock” and hide the Purchase button. |
| **Code change** | `templates/product_detail.html`: `id="product-stock-status"`, `id="product-purchase-btn"`. `static/app.js`: In `purchaseProduct()` when newQty === 0, update `#product-stock-status` innerHTML and `#product-purchase-btn` display none. |
| **Validation** | Product without variants: buy until 0 → status “Out of Stock”, Purchase button hidden. |

---

## 17. Variant row: stock badge and Buy button when variant qty reaches 0

| | |
|---|---|
| **Defect** | After buying the last unit of a variant, that row still showed “In Stock” and the Buy button. |
| **Root cause** | `purchaseVariant()` only updated the variant quantity span. |
| **Fix** | When variant newQty === 0, set that row’s stock to “Out of Stock” and hide its Buy button. |
| **Code change** | `templates/product_detail.html`: Wrapper `id="variant-stock-{{.ID}}"` around stock badge, `id="variant-buy-wrap-{{.ID}}"` around Buy button. `static/app.js`: In `purchaseVariant()` when variant newQty === 0, update `#variant-stock-{id}` and hide `#variant-buy-wrap-{id}`. |
| **Validation** | Buy last unit of a variant → that row shows “Out of Stock” and Buy button hidden. |

---

## 18. Product in_stock only when sum of variant quantities ≥ 1

| | |
|---|---|
| **Defect** | (Requirement) Product with variants should show “in stock” only if total variant quantity ≥ 1. |
| **Root cause** | N/A – implemented as part of #12. |
| **Fix** | Backend already uses `in_stock = (quantity > 0)` where quantity = sum(variant quantities) for products with variants. |
| **Code change** | `store.go`: ListProducts/GetProduct use `(COALESCE(v.tot, p.quantity) > 0) AS in_stock`. |
| **Validation** | Product with variants: in_stock when sum ≥ 1; out of stock when sum = 0. |

---

## 19. Create product allows empty category

| | |
|---|---|
| **Defect** | Product can be created with an empty category (API and form allowed it). |
| **Root cause** | No validation for category in `CreateProduct` or handler; form had no `required` on category. |
| **Fix** | Require non-empty category in store and return 400 with JSON error; mark category required in form and show validation error in UI. |
| **Code change** | `store.go`: In `CreateProduct`, if `strings.TrimSpace(category) == ""` return `fmt.Errorf("category is required")`. `handlers.go`: On CreateProduct error, if error is "name is required", "category is required", or "price must be non-negative" return 400 with `{"error":"..."}`. `templates/new_product.html`: Category label "Category *", input `required`. `static/app.js`: On create failure, parse JSON and showToast(data.error). |
| **Validation** | POST /products with empty category → 400 and `{"error":"category is required"}`. New product form: submit without category → browser validation or API error toast. |

---

## 20. Dashboard total in-stock count wrong

| | |
|---|---|
| **Defect** | Dashboard “Total in stock” (and out of stock) product count does not match list/detail logic when products have variants. |
| **Root cause** | `GetProductCount()` used raw `products.in_stock`; for products with variants, effective in-stock is (sum of variant quantities > 0). |
| **Fix** | Use same logic as ListProducts: LEFT JOIN variant sum, count in-stock as `COALESCE(v.tot, p.quantity) > 0`. Also fix `GetTotalInventory()` to sum effective quantity (variant sum when present). |
| **Code change** | `store_stats.go`: `GetProductCount` query uses LEFT JOIN to variant sum, `SUM(CASE WHEN COALESCE(v.tot, p.quantity) > 0 THEN 1 ELSE 0 END)` and `<= 0` for out of stock. `GetTotalInventory` uses same join and `SUM(COALESCE(v.tot, p.quantity))`. |
| **Validation** | Dashboard /stats and GET /products/stats: total in stock matches number of products that show “In Stock” on list; total inventory matches sum of displayed quantities. |

---

## 21. Search results wrong quantity/in_stock for products with variants

| | |
|---|---|
| **Defect** | GET /search?q=... returned products with raw product quantity/in_stock; products with variants showed wrong values. |
| **Root cause** | `SearchProducts` queried `products` only, not variant-sum logic. |
| **Fix** | Use same LEFT JOIN variant-sum as ListProducts for quantity and in_stock. |
| **Code change** | `store_stats.go`: `SearchProducts` query uses LEFT JOIN to variant sum, `(COALESCE(v.tot, p.quantity) > 0) AS in_stock`, `COALESCE(v.tot, p.quantity) AS quantity`. |
| **Validation** | Search for a product that has variants; results show same quantity/in_stock as list. |

---

## 22. Update product allowed empty name, empty category, negative price

| | |
|---|---|
| **Defect** | PUT /products/:id could set empty name, empty category, or negative price. |
| **Root cause** | `UpdateProduct` had no validation; handler did not check or return 400 for validation errors. |
| **Fix** | Validate in store (name required, category required, price ≥ 0); handler returns 400 with JSON error or 404 for not found. |
| **Code change** | `store.go`: In `UpdateProduct`, return error for empty name, `strings.TrimSpace(category) == ""`, or `priceCents < 0`. `handlers.go`: Before/after UpdateProduct, validate price < 0 → 400; on store error, if validation/not-found return 400 or 404 with `{"error":"..."}`. |
| **Validation** | PUT with empty name or category or negative price → 400; PUT for deleted product → 404. |

---

## 23. Reviews/variants returned 200 for deleted or missing product

| | |
|---|---|
| **Defect** | GET /products/:id/reviews, GET /products/:id/variants, GET /products/:id/inventory, and create review/variant could succeed for deleted or non-existent product (list returned data, create returned 500). |
| **Root cause** | Handlers did not verify product exists (GetProduct) before listing or creating. |
| **Fix** | Call GetProduct(productID) first; if error return 404. For create, return 404 when store returns “product not found”. |
| **Code change** | `handlers_reviews.go`: handleListReviews, handleCreateReview (on “product not found” → 404). `handlers_variants.go`: handleListVariants, handleGetVariantInventory, handleCreateVariant (on “product not found” → 404). |
| **Validation** | GET/POST for deleted or invalid product id → 404. |

---

## 24. Variant/review sub-routes didn’t validate product id or ownership

| | |
|---|---|
| **Defect** | GET/PUT/DELETE /products/:id/variants/:vid and DELETE/approve /products/:id/reviews/:rid did not ensure product exists or that variant/review belongs to that product (e.g. /products/1/variants/99 could return variant 99 even if it belongs to product 2). |
| **Root cause** | Handlers used only variant/review id; did not check product id in path or resource ownership. |
| **Fix** | Parse productID from path; GetProduct(productID) → 404 if missing/deleted; GetVariant/GetReview then require variant.ProductID == productID or review.ProductID == productID, else 404. |
| **Code change** | `handlers_variants.go`: handleGetVariant, handleUpdateVariant, handleDeleteVariant, handlePurchaseVariant — parse productID, GetProduct(productID), then variant.ProductID == productID. `handlers_reviews.go`: handleDeleteReview, handleApproveReview — parse productID, GetProduct(productID), GetReview and review.ProductID == productID. |
| **Validation** | /products/1/variants/99 with variant 99 belonging to product 2 → 404; same for wrong product in review routes. |
