// app.js — Client-side JavaScript for Product Catalog

function showToast(message, type) {
    var toast = document.getElementById('toast');
    toast.textContent = message;
    toast.className = 'toast toast-' + type;
    void toast.offsetWidth;
    toast.classList.add('toast-visible');

    setTimeout(function() {
        toast.classList.remove('toast-visible');
        setTimeout(function() {
            toast.className = 'toast hidden';
        }, 300);
    }, 3000);
}

async function deleteProduct(id) {
    if (!confirm('Are you sure you want to delete this product?')) {
        return;
    }

    try {
        var response = await fetch('/products/' + id, {
            method: 'DELETE',
        });
        if (response.ok) {
            showToast('Deleted!', 'success');
            setTimeout(function() {
                window.location.href = '/';
            }, 1000);
        } else {
            showToast('Failed to delete product', 'error');
        }
    } catch (err) {
        showToast('Network error', 'error');
    }
}

async function purchaseProduct(id) {
    try {
        var response = await fetch('/products/' + id + '/purchase', {
            method: 'POST',
        });
        if (response.ok) {
            showToast('Purchased!', 'success');
            var qtyEl = document.getElementById('product-quantity');
            if (qtyEl) {
                var current = parseInt(qtyEl.textContent, 10);
                var newQty = Math.max(0, current - 1);
                qtyEl.textContent = newQty;
                if (newQty === 0) {
                    var statusEl = document.getElementById('product-stock-status');
                    if (statusEl) {
                        statusEl.innerHTML = '<span class="badge badge-danger">Out of Stock</span>';
                    }
                    var purchaseBtn = document.getElementById('product-purchase-btn');
                    if (purchaseBtn) {
                        purchaseBtn.style.display = 'none';
                    }
                }
            }
        } else {
            var data = await response.json();
            showToast(data.error || 'Purchase failed', 'error');
        }
    } catch (err) {
        showToast('Network error', 'error');
    }
}

async function createProduct(event) {
    event.preventDefault();
    var form = event.target;
    var formData = new FormData(form);

    var data = {
        name: formData.get('name') || '',
        description: formData.get('description') || '',
        price: parseFloat(formData.get('price')) || 0,
        category: formData.get('category') || '',
        quantity: parseInt(formData.get('quantity')) || 0,
        in_stock: (parseInt(formData.get('quantity')) || 0) > 0
    };

    try {
        var response = await fetch('/products', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data),
        });
        if (response.ok) {
            showToast('Product created!', 'success');
            setTimeout(function() {
                window.location.href = '/';
            }, 1000);
        } else {
            var data = await response.json().catch(function() { return {}; });
            showToast(data.error || 'Failed to create product', 'error');
        }
    } catch (err) {
        document.getElementById('error-msg').style.display = 'block';
    }
}

async function submitReview(event, productId) {
    event.preventDefault();
    var form = event.target;
    var formData = new FormData(form);

    var data = {
        author: formData.get('author') || '',
        rating: parseInt(formData.get('rating')) || 0,
        comment: formData.get('comment') || ''
    };

    if (!data.author) {
        showToast('Author name is required', 'error');
        return;
    }
    if (data.rating < 1 || data.rating > 5) {
        showToast('Rating must be between 1 and 5', 'error');
        return;
    }

    try {
        var response = await fetch('/products/' + productId + '/reviews', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data),
        });
        if (response.ok) {
            showToast('Review submitted!', 'success');
            setTimeout(function() {
                window.location.reload();
            }, 1000);
        } else {
            var result = await response.json();
            showToast(result.error || 'Failed to submit review', 'error');
        }
    } catch (err) {
        showToast('Network error', 'error');
    }
}

async function deleteReview(productId, reviewId) {
    if (!confirm('Delete this review?')) {
        return;
    }

    try {
        var response = await fetch('/products/' + productId + '/reviews/' + reviewId, {
            method: 'DELETE',
        });
        if (response.ok) {
            showToast('Review deleted', 'success');
            setTimeout(function() {
                window.location.reload();
            }, 1000);
        } else {
            showToast('Failed to delete review', 'error');
        }
    } catch (err) {
        showToast('Network error', 'error');
    }
}

async function exportProducts(format) {
    var url = '/products/export';
    if (format === 'json') {
        url = '/products/export/json';
    }
    window.location.href = url;
}

async function searchProducts(event) {
    event.preventDefault();
    var query = document.getElementById('search-input').value;
    if (!query.trim()) {
        return;
    }

    try {
        var response = await fetch('/search?q=' + encodeURIComponent(query));
        if (response.ok) {
            var products = await response.json();
            displaySearchResults(products);
        } else {
            showToast('Search failed', 'error');
        }
    } catch (err) {
        showToast('Network error', 'error');
    }
}

function displaySearchResults(products) {
    var container = document.getElementById('search-results');
    if (!container) return;

    if (!products || products.length === 0) {
        container.innerHTML = '<p class="empty-state">No products found.</p>';
        return;
    }

    var html = '<table class="product-table"><thead><tr>';
    html += '<th>Name</th><th>Price</th><th>Category</th><th>In Stock</th>';
    html += '</tr></thead><tbody>';

    products.forEach(function(p) {
        html += '<tr>';
        html += '<td><a href="/products/' + p.id + '">' + escapeHtml(p.name) + '</a></td>';
        html += '<td class="price">$' + p.price.toFixed(2) + '</td>';
        html += '<td>' + escapeHtml(p.category) + '</td>';
        html += '<td>' + (p.in_stock ? '<span class="badge badge-success">In Stock</span>' : '<span class="badge badge-danger">Out of Stock</span>') + '</td>';
        html += '</tr>';
    });

    html += '</tbody></table>';
    container.innerHTML = html;
}

function escapeHtml(str) {
    var div = document.createElement('div');
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
}

function formatDate(dateStr) {
    if (!dateStr) return '';
    var d = new Date(dateStr);
    return d.toLocaleDateString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric'
    });
}

function formatCurrency(amount) {
    return '$' + parseFloat(amount).toFixed(2);
}

async function purchaseVariant(productId, variantId) {
    try {
        var response = await fetch('/products/' + productId + '/variants/' + variantId + '/purchase', {
            method: 'POST',
        });
        if (response.ok) {
            showToast('Variant purchased!', 'success');
            var qtyEl = document.getElementById('variant-qty-' + variantId);
            if (qtyEl) {
                var current = parseInt(qtyEl.textContent, 10);
                var newQty = Math.max(0, current - 1);
                qtyEl.textContent = newQty;
                if (newQty === 0) {
                    var variantStockEl = document.getElementById('variant-stock-' + variantId);
                    if (variantStockEl) {
                        variantStockEl.innerHTML = '<span class="badge badge-danger">Out of Stock</span>';
                    }
                    var buyWrap = document.getElementById('variant-buy-wrap-' + variantId);
                    if (buyWrap) {
                        buyWrap.style.display = 'none';
                    }
                }
            }
            var productQtyEl = document.getElementById('product-quantity');
            if (productQtyEl) {
                var productCurrent = parseInt(productQtyEl.textContent, 10);
                var productNewQty = Math.max(0, productCurrent - 1);
                productQtyEl.textContent = productNewQty;
                if (productNewQty === 0) {
                    var statusEl = document.getElementById('product-stock-status');
                    if (statusEl) {
                        statusEl.innerHTML = '<span class="badge badge-danger">Out of Stock</span>';
                    }
                }
            }
        } else {
            var data = await response.json();
            showToast(data.error || 'Purchase failed', 'error');
        }
    } catch (err) {
        showToast('Network error', 'error');
    }
}
