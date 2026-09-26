package api

import (
	"net/http"

	"pos/internal/catalog"
)

func (a *API) handleListCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := a.Catalog.ListCategories(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cats)
}

func (a *API) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var in catalog.Category
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	c, err := a.Catalog.CreateCategory(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (a *API) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var in catalog.Category
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if err := a.Catalog.UpdateCategory(r.Context(), id, in); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *API) handleListModifierGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := a.Catalog.ListModifierGroups(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, groups)
}

func (a *API) handleCreateModifierGroup(w http.ResponseWriter, r *http.Request) {
	var in catalog.ModifierGroup
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	g, err := a.Catalog.CreateModifierGroup(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, g)
}

func (a *API) handleSearchProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	categoryID := queryInt64Ptr(r, "category_id")
	limit := queryInt(r, "limit", 60)
	offset := queryInt(r, "offset", 0)
	products, err := a.Catalog.SearchProducts(r.Context(), q, categoryID, true, limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, products)
}

func (a *API) handleGetProduct(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	p, err := a.Catalog.GetProduct(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (a *API) handleGetProductByBarcode(w http.ResponseWriter, r *http.Request) {
	barcode := r.PathValue("barcode")
	p, err := a.Catalog.GetProductByBarcode(r.Context(), barcode)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (a *API) handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	var in catalog.ProductInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	user := userFromContext(r.Context())
	p, err := a.Catalog.CreateProduct(r.Context(), in, user.ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (a *API) handleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var in catalog.ProductInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	p, err := a.Catalog.UpdateProduct(r.Context(), id, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (a *API) handleAdjustStock(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ProductID int64  `json:"product_id"`
		Type      string `json:"type"`
		Quantity  int64  `json:"quantity"`
		Note      string `json:"note"`
	}
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, err)
		return
	}
	user := userFromContext(r.Context())
	if err := a.Inventory.AdjustStock(r.Context(), in.ProductID, in.Type, in.Quantity, in.Note, user.ID); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *API) handleListMovements(w http.ResponseWriter, r *http.Request) {
	productID := queryInt64Ptr(r, "product_id")
	limit := queryInt(r, "limit", 100)
	movements, err := a.Inventory.ListMovements(r.Context(), productID, limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, movements)
}

func (a *API) handleLowStock(w http.ResponseWriter, r *http.Request) {
	items, err := a.Inventory.LowStock(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
