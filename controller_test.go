package swagger_test

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/iris-contrib/swagger/v14"
	"github.com/iris-contrib/swagger/v14/swaggerFiles"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
)

// Store is the model of the StoreController: its request and its response.
type Store struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// StoreFilter is the body of the list request.
type StoreFilter struct {
	Name string `json:"name"`
}

// StoreRepository is the dependency of the StoreController.
type StoreRepository struct {
	mu     sync.RWMutex
	stores map[string]Store
}

func NewStoreRepository() *StoreRepository {
	return &StoreRepository{stores: make(map[string]Store)}
}

// StoreController is a controller of the controller/mvc package.
// Its five methods become the five routes that storesDoc describes.
type StoreController struct {
	Repository *StoreRepository
}

// POST /stores
func (c *StoreController) Create(ctx iris.Context, in Store) (Store, error) {
	c.Repository.mu.Lock()
	defer c.Repository.mu.Unlock()

	in.ID = fmt.Sprintf("store-%d", len(c.Repository.stores)+1)
	c.Repository.stores[in.ID] = in

	return in, nil
}

// GET /stores/{store_id}
func (c *StoreController) Get(ctx iris.Context, id string) (Store, error) {
	c.Repository.mu.RLock()
	defer c.Repository.mu.RUnlock()

	store, ok := c.Repository.stores[id]
	if !ok {
		return Store{}, errors.NotFound.Wrap(fmt.Errorf("store %s not found", id))
	}

	return store, nil
}

// PUT /stores/{store_id}
func (c *StoreController) Update(ctx iris.Context, id string, in Store, columns ...string) (*Store, bool, error) {
	c.Repository.mu.Lock()
	defer c.Repository.mu.Unlock()

	store, ok := c.Repository.stores[id]
	if !ok {
		return nil, false, errors.NotFound.Wrap(fmt.Errorf("store %s not found", id))
	}

	store.Name = in.Name
	c.Repository.stores[id] = store

	return &store, true, nil
}

// DELETE /stores/{store_id}
func (c *StoreController) Delete(ctx iris.Context, id string) (bool, error) {
	c.Repository.mu.Lock()
	defer c.Repository.mu.Unlock()

	_, ok := c.Repository.stores[id]
	delete(c.Repository.stores, id)

	return ok, nil
}

// POST /stores/list
func (c *StoreController) List(ctx iris.Context, filter StoreFilter, limit, offset int) ([]Store, int, error) {
	c.Repository.mu.RLock()
	defer c.Repository.mu.RUnlock()

	stores := make([]Store, 0, len(c.Repository.stores))
	for _, store := range c.Repository.stores {
		if strings.Contains(store.Name, filter.Name) {
			stores = append(stores, store)
		}
	}

	return stores, len(stores), nil
}

// macroType matches the type of a path parameter: the ":string" of
// "{store_id:string}". A definition has the name alone, "{store_id}".
var macroType = regexp.MustCompile(`:[^}]+}`)

// An application that mounts a controller serves the page and the definition
// next to it, and the definition describes the routes the controller has.
func TestControllerApplication(t *testing.T) {
	app := iris.New()

	if err := app.RegisterDependency(NewStoreRepository()); err != nil {
		t.Fatal(err)
	}

	app.Router("/stores", mvc.UseController[*StoreController]())

	swaggerUI := swagger.Handler(swaggerFiles.Handler,
		swagger.URL("/swagger/doc.json"),
		swagger.Prefix("/swagger"),
	)
	app.Get("/swagger", swaggerUI)
	app.Get("/swagger/{any:path}", swaggerUI)

	e := httptest.New(t, app)

	// The page and its files.
	e.GET("/swagger/index.html").Expect().Status(httptest.StatusOK).
		HasContentType("text/html", "utf-8").Body().
		Contains(`<div id="swagger-ui"></div>`).
		Contains(`url: "\/swagger\/doc.json"`)
	e.GET("/swagger/swagger-ui-bundle.js").Expect().Status(httptest.StatusOK).
		HasContentType("application/javascript").Body().IsEqual(file(t, "/swagger-ui-bundle.js"))

	// The definition.
	doc := e.GET("/swagger/doc.json").Expect().Status(httptest.StatusOK).
		HasContentType("application/json").JSON().Object()
	doc.HasValue("swagger", "2.0")

	var documented []string
	for path, operations := range doc.Value("paths").Object().Iter() {
		for method := range operations.Object().Iter() {
			documented = append(documented, strings.ToUpper(method)+" "+path)
		}
	}
	slices.Sort(documented)

	// The routes of the controller, as the router has them.
	var registered []string
	for _, route := range app.Routes().List() {
		path := route.GetPath()
		if !strings.HasPrefix(path, "/stores") {
			continue
		}

		if route.GetMethod() == iris.MethodQuery {
			// The list route answers POST and QUERY.
			// Swagger 2.0 has no word for the second one.
			continue
		}

		registered = append(registered, route.GetMethod()+" "+macroType.ReplaceAllString(path, "}"))
	}
	slices.Sort(registered)

	expected := []string{
		"DELETE /stores/{store_id}",
		"GET /stores/{store_id}",
		"POST /stores",
		"POST /stores/list",
		"PUT /stores/{store_id}",
	}

	if !slices.Equal(registered, expected) {
		t.Fatalf("routes of the controller:\n%v\nexpected:\n%v", registered, expected)
	}

	if !slices.Equal(documented, registered) {
		t.Fatalf("operations of the definition:\n%v\nroutes of the controller:\n%v", documented, registered)
	}

	// The controller answers on the paths the definition gives.
	created := e.POST("/stores").WithJSON(Store{Name: "Athens"}).
		Expect().Status(httptest.StatusCreated).JSON().Object()
	created.HasValue("name", "Athens")
	id := created.Value("id").String().NotEmpty().Raw()

	e.GET("/stores/"+id).Expect().Status(httptest.StatusOK).
		JSON().Object().HasValue("id", id).HasValue("name", "Athens")
	e.PUT("/stores/"+id).WithJSON(Store{Name: "Patras"}).
		Expect().Status(httptest.StatusOK).JSON().Object().HasValue("name", "Patras")
	e.POST("/stores/list").WithJSON(StoreFilter{Name: "Pat"}).
		Expect().Status(httptest.StatusOK).JSON().Path("$.items[0].id").IsEqual(id)
	e.DELETE("/stores/" + id).Expect().Status(httptest.StatusNoContent)
	e.GET("/stores/" + id).Expect().Status(httptest.StatusNotFound)

	// And the page is still there after it did.
	e.GET("/swagger").Expect().Status(httptest.StatusOK)
}

// An application made by iris.NewBuilder mounts the page as one more API.
// Every API of it is under the prefix of the builder, so the page is too,
// and the Prefix option says the whole path.
func TestBuilderApplication(t *testing.T) {
	swaggerUI := swagger.Handler(swaggerFiles.Handler,
		swagger.URL("/api/swagger/swagger.json"),
		swagger.Prefix("/api/swagger"),
	)

	app := iris.NewBuilder().
		Prefix("/api").
		AllowOrigin("*").
		Compression(false).
		LogRequests(false).
		NoHealth().
		Services(NewStoreRepository()).
		RouterMiddlewares().
		Middlewares().
		API("/stores", mvc.UseController[*StoreController]()).
		API("/swagger", func(r iris.Router) {
			r.Get("/", swaggerUI)
			r.Get("/{any:path}", swaggerUI)
		}).
		Build()

	e := httptest.New(t, app)

	for _, path := range []string{"/api/swagger", "/api/swagger/index.html"} {
		e.GET(path).Expect().Status(httptest.StatusOK).
			HasContentType("text/html", "utf-8").Body().
			Contains(`<div id="swagger-ui"></div>`).
			Contains(`src="/api/swagger/swagger-ui-bundle.js"`).
			Contains(`url: "\/api\/swagger\/swagger.json"`)
	}

	e.GET("/api/swagger/swagger-ui-bundle.js").Expect().Status(httptest.StatusOK).
		HasContentType("application/javascript").Body().IsEqual(file(t, "/swagger-ui-bundle.js"))
	e.GET("/api/swagger/swagger.json").Expect().Status(httptest.StatusOK).
		HasContentType("application/json").JSON().Path("$.info.title").IsEqual("Stores API")
	e.GET("/api/swagger/missing.css").Expect().Status(httptest.StatusNotFound)

	// Nothing of it is outside the prefix.
	e.GET("/swagger").Expect().Status(httptest.StatusNotFound)

	// The controller, next to it.
	e.POST("/api/stores").WithJSON(Store{Name: "Athens"}).
		Expect().Status(httptest.StatusCreated).JSON().Object().HasValue("name", "Athens")
}
