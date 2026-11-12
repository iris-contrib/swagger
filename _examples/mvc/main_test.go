package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/swaggo/swag"
)

func TestSwaggerUI(t *testing.T) {
	e := httptest.New(t, newApp())

	for _, path := range []string{"/swagger", "/swagger/index.html"} {
		e.GET(path).Expect().Status(httptest.StatusOK).
			HasContentType("text/html", "utf-8").Body().
			Contains(`<div id="swagger-ui"></div>`).
			Contains(`src="/swagger/swagger-ui-bundle.js"`).
			Contains(`url: "\/swagger\/swagger.json"`)
	}

	e.GET("/swagger/swagger-ui-bundle.js").Expect().Status(httptest.StatusOK).
		HasContentType("application/javascript").Body().NotEmpty()
	e.GET("/swagger/swagger-ui.css").Expect().Status(httptest.StatusOK).
		HasContentType("text/css", "utf-8").Body().NotEmpty()

	doc := e.GET("/swagger/swagger.json").Expect().Status(httptest.StatusOK).
		HasContentType("application/json").JSON().Object()
	doc.HasValue("swagger", "2.0")
	doc.Path("$.info.title").IsEqual("Stores API")
	doc.Value("paths").Object().Value("/stores/{store_id}").Object().
		Value("get").Object().HasValue("summary", "Get a store")

	// The same definition, under the name it has in other integrations.
	e.GET("/swagger/doc.json").Expect().Status(httptest.StatusOK).
		HasContentType("application/json").JSON().Object().IsEqual(doc.Raw())
}

// macroType matches the type of a path parameter: the ":string" of
// "{store_id:string}". A definition has the name alone, "{store_id}".
var macroType = regexp.MustCompile(`:[^}]+}`)

// Every route of the controller is in the definition, with the path and the
// HTTP method that Iris gave it, and the definition has nothing else.
func TestDefinitionMatchesRoutes(t *testing.T) {
	app := newApp()
	e := httptest.New(t, app)

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
		"GET /stores/count",
		"GET /stores/{store_id}",
		"POST /stores",
		"POST /stores/list",
		"PUT /stores/{store_id}",
	}

	if !slices.Equal(registered, expected) {
		t.Fatalf("routes of the controller:\n%v\nexpected:\n%v", registered, expected)
	}

	doc := e.GET("/swagger/swagger.json").Expect().Status(httptest.StatusOK).JSON().Object()
	doc.HasValue("basePath", "/")

	var documented []string
	for path, operations := range doc.Value("paths").Object().Iter() {
		for method := range operations.Object().Iter() {
			documented = append(documented, strings.ToUpper(method)+" "+path)
		}
	}
	slices.Sort(documented)

	if !slices.Equal(documented, registered) {
		t.Fatalf("operations of the definition:\n%v\nroutes of the controller:\n%v", documented, registered)
	}
}

// The docs directory is what "swag init" writes for the comments as they are
// now. It fails when a comment was changed and "swag init" was not run again.
func TestDocsAreCurrent(t *testing.T) {
	parser := swag.New(swag.SetDebugger(log.New(io.Discard, "", 0)))
	if err := parser.ParseAPI(".", "main.go", 100); err != nil {
		t.Fatal(err)
	}

	b, err := json.Marshal(parser.GetSwagger())
	if err != nil {
		t.Fatal(err)
	}

	var fromComments any
	if err = json.Unmarshal(b, &fromComments); err != nil {
		t.Fatal(err)
	}

	b, err = os.ReadFile("docs/swagger.json")
	if err != nil {
		t.Fatal(err)
	}

	var fromDocs any
	if err = json.Unmarshal(b, &fromDocs); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(fromComments, fromDocs) {
		t.Fatal("docs/swagger.json is not what the comments say, run: swag init")
	}
}

// The controller answers the way its comments say.
func TestStores(t *testing.T) {
	e := httptest.New(t, newApp())

	e.GET("/stores/count").Expect().Status(httptest.StatusOK).
		JSON().Object().IsEqual(StoreCount{Total: 0})

	// Create.
	e.POST("/stores").WithJSON(Store{Name: "Athens"}).
		Expect().Status(httptest.StatusCreated).
		JSON().Object().IsEqual(Store{ID: "store-1", Name: "Athens"})
	e.POST("/stores").WithJSON(Store{Name: "Patras"}).
		Expect().Status(httptest.StatusCreated).
		JSON().Object().IsEqual(Store{ID: "store-2", Name: "Patras"})
	e.POST("/stores").WithHeader("Content-Type", "application/json").WithBytes([]byte(`[`)).
		Expect().Status(httptest.StatusBadRequest)

	// Get.
	e.GET("/stores/store-1").Expect().Status(httptest.StatusOK).
		JSON().Object().IsEqual(Store{ID: "store-1", Name: "Athens"})
	e.GET("/stores/store-9").Expect().Status(httptest.StatusNotFound)

	// Update.
	e.PUT("/stores/store-2").WithJSON(Store{Name: "Thessaloniki"}).
		Expect().Status(httptest.StatusOK).
		JSON().Object().IsEqual(Store{ID: "store-2", Name: "Thessaloniki"})
	e.PUT("/stores/store-2").WithHeader("Content-Type", "application/json").WithBytes([]byte(`[`)).
		Expect().Status(httptest.StatusBadRequest)
	e.PUT("/stores/store-9").WithJSON(Store{Name: "Nowhere"}).
		Expect().Status(httptest.StatusNotFound)

	// List.
	page := e.POST("/stores/list").WithQuery("page", 1).WithQuery("size", 1).
		WithJSON(StoreFilter{}).
		Expect().Status(httptest.StatusOK).JSON().Object()
	page.HasValue("current_page", 1).HasValue("page_size", 1).
		HasValue("total_pages", 2).HasValue("total_items", 2).
		HasValue("has_next_page", true)
	page.Value("items").Array().IsEqual([]Store{{ID: "store-1", Name: "Athens"}})

	e.POST("/stores/list").WithJSON(StoreFilter{Name: "Thess"}).
		Expect().Status(httptest.StatusOK).JSON().Object().
		Value("items").Array().IsEqual([]Store{{ID: "store-2", Name: "Thessaloniki"}})

	// StorePage describes that response to swag. It has to list
	// the fields of an iris.Page, no more and no less.
	var documented []string
	for field := range reflect.TypeFor[StorePage]().Fields() {
		documented = append(documented, field.Tag.Get("json"))
	}
	slices.Sort(documented)

	var sent []string
	for name := range page.Raw() {
		sent = append(sent, name)
	}
	slices.Sort(sent)

	if !slices.Equal(documented, sent) {
		t.Fatalf("fields of StorePage:\n%v\nfields of the list response:\n%v", documented, sent)
	}

	// Delete.
	e.DELETE("/stores/store-1").Expect().Status(httptest.StatusNoContent)
	e.DELETE("/stores/store-1").Expect().Status(httptest.StatusNotModified)

	e.GET("/stores/count").Expect().Status(httptest.StatusOK).
		JSON().Object().IsEqual(StoreCount{Total: 1})
}
