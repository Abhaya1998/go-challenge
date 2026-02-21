# Product Catalog — Project Summary

This document summarizes the work done on the Product Catalog codebase: defects addressed, documentation added, and how AI was used during the process.

---

## Overview

The Product Catalog is a Go web application for managing products with variants, reviews, and inventory. The work focused on **finding and fixing functional bugs**, **documenting fixes and validation**, and **outlining future improvements**. All changes aim for correctness and maintainability without cosmetic refactors.

---

## What Was Delivered

### 1. Code changes

- **24 functional bugs fixed** across the store, handlers, middleware, templates, and frontend. Areas covered:
  - **Correctness:** Price conversion (cents ↔ dollars), product quantity as sum of variant quantities, soft-delete filtering, atomic purchase (no negative stock).
  - **Security:** Parameterized SQL for category filter (no SQL injection).
  - **API/UX:** Create/update validation (e.g. category required), proper HTTP status and JSON errors, cache invalidation, route ordering so variant purchase is handled correctly.
  - **UI:** Delete using DELETE method, product/variant stock status and button state when quantity reaches zero, routing fix for variant purchase.
  - **Consistency:** Search, dashboard stats, and category stats use the same variant-sum logic as the product list; reviews and variants return 404 when the product is missing or deleted; sub-resource handlers verify product existence and ownership.

Detailed **defect, root cause, fix, code change, and validation** for each item are in **`FUNCTIONAL_BUGS.md`** (summary table at the top, full entries below).

### 2. Documentation

- **`FUNCTIONAL_BUGS.md`** — Fix log for all 24 bugs in a consistent format (defect, root cause, fix summary, code change, validation). Summary table moved to the top for quick reference.
- **`FUTURE_IMPROVEMENTS.md`** — Suggestions for next steps: UI parity with the API, **adding test cases** (as a high-priority item), containerization, logging, error handling, design patterns, folder structure, and RBAC. Each section is short and actionable.
- **`.cursor/rules/update-bugs-doc.mdc`** — Project rule so any future bug fix also updates the fix log.
- **`bugs.txt`** — Kept in sync with the fix log; original observations linked to the corresponding bug entries.

### 3. Validation

- Fixes were validated by **code review** (logic, edge cases, concurrency), **manual checks** (list/detail, purchase, delete, dashboard, search, category filter), and **consistency checks** (same variant-sum and `deleted_at` rules everywhere). No new automated tests were added in this pass; the focus was on correctness of behavior and data. **Future improvements** (see `FUTURE_IMPROVEMENTS.md`) list **adding test cases** as a high-priority next step to protect against regressions and support refactoring.

---

## Defects by category (high level)

| Category        | Examples |
|----------------|----------|
| Data / display | Wrong price (×100 and template double-div), negative price in seed, deleted products in list, wrong in-stock/quantity when products have variants. |
| Security       | SQL injection via category filter. |
| API / handlers | Create product not returning on error, audit limit ignored when `product_id` set, category stats average price in cents. |
| Concurrency    | Purchase could drive quantity negative; fixed with atomic UPDATE and `quantity > 0` guard. |
| Cache          | Stale product after update/delete/purchase; cache invalidated on those operations. |
| Routing / UI   | Variant purchase hitting product-purchase handler (route order), UI delete using GET, product/variant stock and buttons not updating when qty → 0. |
| Validation     | Category required on create; name, category, price validated on update; 404 for deleted/missing product on review and variant endpoints; variant/review ownership checks. |

Full list with one-line fixes is in **`FUNCTIONAL_BUGS.md`** (Summary table).

---

## AI usage — brief note

### Tools used

- **Cursor (AI-assisted editor)** for exploration, implementing fixes, and updating documentation. The AI was used for code search, reading handlers and store logic, and drafting edits across Go, HTML, and JavaScript.

### Where AI helped

- **Discovery:** Quickly scanning the repo for patterns (e.g. `ListProducts`, `GetProduct`, price conversion, routing order) and identifying multiple related bugs (e.g. variant-sum consistency in list, stats, search, dashboard).
- **Implementing fixes:** Applying the intended behavior (e.g. parameterized queries, atomic decrement, cache invalidation, 404 when product is deleted) and keeping changes localized and consistent with existing style.
- **Documentation:** Drafting **`FUNCTIONAL_BUGS.md`** entries (defect, root cause, fix, code change, validation) and **`FUTURE_IMPROVEMENTS.md`** sections so the write-up is structured and easy to follow.
- **Consistency:** Ensuring the same rules (e.g. variant-sum, `deleted_at` filtering) were applied in store, stats, and search, and that sub-resource handlers (variants, reviews) validate product existence and ownership.

### Where output was disagreed with or overridden

- **Scope:** AI sometimes suggested extra refactors (e.g. large structural changes). Those were declined in favor of minimal, behavior-only fixes as requested.
- **Validation:** AI occasionally proposed adding unit tests for every fix; the choice was to rely on code review and manual validation to stay within the stated scope.
- **Docs:** Some AI-generated wording was tightened or reordered (e.g. moving the Summary table to the top of the fix log, keeping the “AI usage” note brief and factual) to better suit the deliverables.
- **Edge cases:** For example, whether “product quantity = sum of variants” should also change purchase semantics (e.g. decrementing a variant when purchasing at product level). The final behavior (purchase only by variant when variants exist, with UI and route fixes) was decided to match the desired product/variant model rather than accepting the first suggested approach.

Overall, AI was used to speed up exploration and implementation while keeping final decisions on scope, design, and wording under manual control.
