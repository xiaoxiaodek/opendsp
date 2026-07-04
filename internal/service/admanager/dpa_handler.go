package admanager

import (
	"net/http"
)

type DPAHandler struct{}

func NewDPAHandler() *DPAHandler {
	return &DPAHandler{}
}

func (h *DPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.URL.Path {
	case "/api/dpa/products":
		if r.Method == "GET" {
			h.handleProducts(w, r)
		} else {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	case "/api/dpa/sync":
		if r.Method == "POST" {
			h.handleSync(w, r)
		} else {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	}
}

func (h *DPAHandler) handleProducts(w http.ResponseWriter, r *http.Request) {
	items := []map[string]interface{}{
		{"id": "p1", "advertiser_id": 1, "title": "Running Shoes", "image_url": "/img/shoes.jpg", "landing_url": "/p/shoes", "price": 99.99, "category": "Footwear", "brand": "Nike", "in_stock": true},
		{"id": "p2", "advertiser_id": 1, "title": "Wireless Earbuds", "image_url": "/img/earbuds.jpg", "landing_url": "/p/earbuds", "price": 149.00, "category": "Electronics", "brand": "Sony", "in_stock": true},
		{"id": "p3", "advertiser_id": 1, "title": "Yoga Mat", "image_url": "/img/yoga.jpg", "landing_url": "/p/yoga", "price": 29.99, "category": "Fitness", "brand": "Lululemon", "in_stock": false},
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"stats": map[string]interface{}{
			"total_products":       3,
			"active_campaigns":     1,
			"retargeted_users_24h": 1247,
		},
	})
}

func (h *DPAHandler) handleSync(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "Feed sync started"})
}