package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

// Store is the model of the API: the body of a create and of an update
// request, and the body of every response that is a store.
type Store struct {
	ID   string `json:"id" example:"store-1"`
	Name string `json:"name" example:"Athens"`
}

// StoreFilter is the body of the list request.
type StoreFilter struct {
	// Name keeps the stores that have it in their name.
	Name string `json:"name" example:"Ath"`
}

// StorePage is the body of the list response, written down for swag.
//
// The response itself is an iris.Page[Store] and it has these fields.
// swag reads the types of this directory, not the ones of the framework,
// so a comment cannot name iris.Page.
type StorePage struct {
	CurrentPage int         `json:"current_page" example:"1"`
	PageSize    int         `json:"page_size" example:"10"`
	TotalPages  int         `json:"total_pages" example:"1"`
	TotalItems  int64       `json:"total_items" example:"1"`
	HasNextPage bool        `json:"has_next_page" example:"false"`
	Filter      StoreFilter `json:"filter"`
	Items       []Store     `json:"items"`
}

// StoreCount is the body of the count response.
type StoreCount struct {
	Total int `json:"total" example:"1"`
}

// StoreController is mounted with mvc.UseController, which makes a route out
// of each of its five methods: Create, Get, Update, Delete and List.
//
// swag does not see those routes. It reads comments, it does not run the
// application. So every method has the comments of its own operation, and
// the @Router line among them gives the full path and the HTTP method of
// the route that Iris makes for that method.
type StoreController struct {
	// Repository is set by Iris, see app.RegisterDependency in main.go.
	Repository *StoreRepository
}

// Create adds a store.
//
// @Summary Create a store
// @Tags stores
// @Accept json
// @Produce json
// @Param store body Store true "The new store. The server sets its id."
// @Success 201 {object} Store
// @Failure 400 "The body is not a JSON object"
// @Router /stores [post]
func (c *StoreController) Create(ctx iris.Context, in Store) (Store, error) {
	return c.Repository.Create(in), nil
}

// Get returns a store.
//
// The path parameter is named after the model: store_id for a Store.
//
// @Summary Get a store
// @Tags stores
// @Produce json
// @Param store_id path string true "The id of the store"
// @Success 200 {object} Store
// @Failure 404 "There is no store with this id"
// @Router /stores/{store_id} [get]
func (c *StoreController) Get(ctx iris.Context, id string) (Store, error) {
	store, ok := c.Repository.Get(id)
	if !ok {
		return Store{}, errors.NotFound.Wrap(fmt.Errorf("store %s not found", id))
	}

	return store, nil
}

// Update changes the name of a store.
//
// @Summary Update a store
// @Tags stores
// @Accept json
// @Produce json
// @Param store_id path string true "The id of the store"
// @Param store body Store true "The new values of the store"
// @Success 200 {object} Store
// @Failure 400 "The body is not a JSON object"
// @Failure 404 "There is no store with this id"
// @Router /stores/{store_id} [put]
func (c *StoreController) Update(ctx iris.Context, id string, in Store, columns ...string) (*Store, bool, error) {
	store, ok := c.Repository.Rename(id, in.Name)
	if !ok {
		return nil, false, errors.NotFound.Wrap(fmt.Errorf("store %s not found", id))
	}

	return &store, true, nil
}

// Delete removes a store.
//
// @Summary Delete a store
// @Tags stores
// @Param store_id path string true "The id of the store"
// @Success 204 "The store is removed"
// @Success 304 "There was no store with this id"
// @Router /stores/{store_id} [delete]
func (c *StoreController) Delete(ctx iris.Context, id string) (bool, error) {
	return c.Repository.Delete(id), nil
}

// List returns a page of stores.
//
// The page number and its size are read from the URL query,
// the filter from the body.
//
// @Summary List the stores
// @Tags stores
// @Accept json
// @Produce json
// @Param page query int false "The number of the page, the first one is 1" default(1)
// @Param size query int false "The number of stores in a page"
// @Param filter body StoreFilter false "Which stores to list"
// @Success 200 {object} StorePage
// @Router /stores/list [post]
func (c *StoreController) List(ctx iris.Context, filter StoreFilter, limit, offset int) ([]Store, int, error) {
	stores := c.Repository.Find(filter.Name)
	total := len(stores)

	if offset > total {
		offset = total
	}

	stores = stores[offset:]
	if limit > 0 && limit < len(stores) {
		stores = stores[:limit]
	}

	return stores, total, nil
}

// Routing adds routes next to the five ones that Iris makes.
// Iris calls it once, when the controller is mounted.
func (c *StoreController) Routing(r iris.Router) {
	r.Get("/count", c.count)
}

// count is the handler of a route that is registered by hand.
// The comments work the same way: they are on the method that the route runs.
//
// @Summary Count the stores
// @Tags stores
// @Produce json
// @Success 200 {object} StoreCount
// @Router /stores/count [get]
func (c *StoreController) count(ctx iris.Context) {
	ctx.JSON(StoreCount{Total: c.Repository.Count()})
}

// StoreRepository keeps the stores in memory.
type StoreRepository struct {
	mu     sync.RWMutex
	stores map[string]Store
	lastID int
}

func NewStoreRepository() *StoreRepository {
	return &StoreRepository{stores: make(map[string]Store)}
}

func (r *StoreRepository) Create(store Store) Store {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastID++
	store.ID = fmt.Sprintf("store-%d", r.lastID)
	r.stores[store.ID] = store

	return store
}

func (r *StoreRepository) Get(id string) (Store, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	store, ok := r.stores[id]
	return store, ok
}

func (r *StoreRepository) Rename(id, name string) (Store, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	store, ok := r.stores[id]
	if !ok {
		return Store{}, false
	}

	store.Name = name
	r.stores[id] = store

	return store, true
}

func (r *StoreRepository) Delete(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.stores[id]
	delete(r.stores, id)

	return ok
}

// Find returns the stores that have "name" in their name, sorted by id.
func (r *StoreRepository) Find(name string) []Store {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stores := make([]Store, 0, len(r.stores))
	for _, store := range r.stores {
		if strings.Contains(store.Name, name) {
			stores = append(stores, store)
		}
	}

	sort.Slice(stores, func(i, j int) bool {
		return stores[i].ID < stores[j].ID
	})

	return stores
}

func (r *StoreRepository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.stores)
}
