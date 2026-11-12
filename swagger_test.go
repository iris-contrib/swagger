package swagger_test

import (
	"compress/gzip"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/iris-contrib/swagger/v14"
	"github.com/iris-contrib/swagger/v14/swaggerFiles"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/compress"
	"github.com/swaggo/swag"
)

// storesDoc is the API definition of the StoreController in
// controller_test.go, the way "swag init" writes it into the docs package.
const storesDoc = `{
    "swagger": "2.0",
    "info": {
        "title": "Stores API",
        "version": "1.0"
    },
    "basePath": "/",
    "paths": {
        "/stores": {
            "post": {
                "summary": "Create a store",
                "responses": {"201": {"description": "Created"}}
            }
        },
        "/stores/list": {
            "post": {
                "summary": "List the stores",
                "responses": {"200": {"description": "OK"}}
            }
        },
        "/stores/{store_id}": {
            "get": {
                "summary": "Get a store",
                "parameters": [{"type": "string", "name": "store_id", "in": "path", "required": true}],
                "responses": {"200": {"description": "OK"}}
            },
            "put": {
                "summary": "Update a store",
                "parameters": [{"type": "string", "name": "store_id", "in": "path", "required": true}],
                "responses": {"200": {"description": "OK"}}
            },
            "delete": {
                "summary": "Delete a store",
                "parameters": [{"type": "string", "name": "store_id", "in": "path", "required": true}],
                "responses": {"204": {"description": "No Content"}}
            }
        }
    }
}`

// internalDoc is a second definition, registered under its own instance name.
const internalDoc = `{
    "swagger": "2.0",
    "info": {
        "title": "Internal API",
        "version": "1.0"
    },
    "paths": {}
}`

func TestMain(m *testing.M) {
	// The init function of a generated docs package does this.
	swag.Register(swag.Name, &swag.Spec{
		InfoInstanceName: swag.Name,
		SwaggerTemplate:  storesDoc,
	})
	swag.Register("internal", &swag.Spec{
		InfoInstanceName: "internal",
		SwaggerTemplate:  internalDoc,
	})

	os.Exit(m.Run())
}

// file returns an embedded file of the Swagger UI.
func file(t *testing.T, name string) string {
	t.Helper()

	b, err := swaggerFiles.ReadFile(name)
	if err != nil {
		t.Fatalf("embedded file %s: %v", name, err)
	}

	return string(b)
}

func TestHandler(t *testing.T) {
	app := iris.New()

	swaggerUI := swagger.Handler(swaggerFiles.Handler,
		swagger.URL("/swagger/swagger.json"),
		swagger.DeepLinking(true),
		swagger.Prefix("/swagger"),
	)
	app.Get("/swagger", swaggerUI)
	app.Get("/swagger/{any:path}", swaggerUI)

	e := httptest.New(t, app)

	// The page, under both of its addresses.
	for _, path := range []string{"/swagger", "/swagger/index.html"} {
		page := e.GET(path).Expect().Status(httptest.StatusOK).
			HasContentType("text/html", "utf-8").Body()
		page.Contains(`<div id="swagger-ui"></div>`)
		page.Contains(`href="/swagger/swagger-ui.css"`)
		page.Contains(`src="/swagger/swagger-ui-bundle.js"`)
		page.Contains(`src="/swagger/swagger-ui-standalone-preset.js"`)
		page.Contains(`url: "\/swagger\/swagger.json"`)
		page.Contains(`deepLinking:  true`)
		page.Contains(`docExpansion: "list"`)
		page.Contains(`dom_id: "#swagger-ui"`)
		page.Contains(`filter:  true`)
		// No theme was asked for.
		page.NotContains("theme-").NotContains("unknow")
	}

	// The files of the page.
	e.GET("/swagger/swagger-ui.css").Expect().Status(httptest.StatusOK).
		HasContentType("text/css", "utf-8").Body().IsEqual(file(t, "/swagger-ui.css"))
	e.GET("/swagger/swagger-ui-bundle.js").Expect().Status(httptest.StatusOK).
		HasContentType("application/javascript").Body().IsEqual(file(t, "/swagger-ui-bundle.js"))
	e.GET("/swagger/swagger-ui-standalone-preset.js").Expect().Status(httptest.StatusOK).
		HasContentType("application/javascript").Body().IsEqual(file(t, "/swagger-ui-standalone-preset.js"))
	e.GET("/swagger/favicon-32x32.png").Expect().Status(httptest.StatusOK).
		HasContentType("image/png").Body().IsEqual(file(t, "/favicon-32x32.png"))
	e.GET("/swagger/oauth2-redirect.html").Expect().Status(httptest.StatusOK).
		HasContentType("text/html", "utf-8").Body().IsEqual(file(t, "/oauth2-redirect.html"))

	// A file is not sent again to a browser that has it.
	etag := e.GET("/swagger/swagger-ui.css").Expect().Status(httptest.StatusOK).
		Header("ETag").NotEmpty().Raw()
	e.GET("/swagger/swagger-ui.css").WithHeader("If-None-Match", etag).
		Expect().Status(httptest.StatusNotModified).Body().IsEmpty()

	e.GET("/swagger/missing.css").Expect().Status(httptest.StatusNotFound)

	// The API definition.
	doc := e.GET("/swagger/swagger.json").Expect().Status(httptest.StatusOK).
		HasContentType("application/json").JSON().Object()
	doc.HasValue("swagger", "2.0")
	doc.Path("$.info.title").IsEqual("Stores API")
	doc.Value("paths").Object().ContainsKey("/stores/{store_id}")
}

// Every embedded file is reachable, the themes and the source maps included.
func TestHandlerServesEveryFile(t *testing.T) {
	app := iris.New()

	swaggerUI := swagger.Handler(swaggerFiles.Handler)
	app.Get("/swagger", swaggerUI)
	app.Get("/swagger/{any:path}", swaggerUI)

	e := httptest.New(t, app)

	names, err := swaggerFiles.WalkDirs("", false)
	if err != nil {
		t.Fatal(err)
	}

	if len(names) < 19 {
		t.Fatalf("expected at least 19 embedded files, got %d: %v", len(names), names)
	}

	for _, name := range names {
		if name == "index.html" {
			// The page is rendered by the handler, with the options in it.
			continue
		}

		body := e.GET("/swagger/" + name).Expect().Status(httptest.StatusOK).Body().Raw()
		if body != file(t, "/"+name) {
			t.Errorf("%s: served %d bytes, which is not the embedded file", name, len(body))
		}
	}
}

// With no option at all, the page finds the definition under the prefix,
// whichever of its two addresses it was opened from.
func TestHandlerDefaults(t *testing.T) {
	app := iris.New()

	swaggerUI := swagger.Handler(swaggerFiles.Handler)
	app.Get("/swagger", swaggerUI)
	app.Get("/swagger/{any:path}", swaggerUI)

	e := httptest.New(t, app)

	for _, path := range []string{"/swagger", "/swagger/index.html"} {
		e.GET(path).Expect().Status(httptest.StatusOK).Body().
			Contains(`url: "\/swagger\/swagger.json"`).
			Contains(`href="/swagger/swagger-ui.css"`)
	}

	e.GET("/swagger/swagger.json").Expect().Status(httptest.StatusOK).
		JSON().Object().HasValue("swagger", "2.0")
}

// "/swagger/doc.json" is where the definition was before instance names,
// and where the Swagger integrations of other frameworks keep it.
func TestHandlerDocJSON(t *testing.T) {
	app := iris.New()

	swaggerUI := swagger.Handler(swaggerFiles.Handler, swagger.URL("/swagger/doc.json"))
	app.Get("/swagger", swaggerUI)
	app.Get("/swagger/{any:path}", swaggerUI)

	e := httptest.New(t, app)

	e.GET("/swagger").Expect().Status(httptest.StatusOK).Body().
		Contains(`url: "\/swagger\/doc.json"`)

	byName := e.GET("/swagger/swagger.json").Expect().Status(httptest.StatusOK).Body().Raw()
	e.GET("/swagger/doc.json").Expect().Status(httptest.StatusOK).
		HasContentType("application/json").Body().IsEqual(byName)
}

func TestHandlerInstanceName(t *testing.T) {
	app := iris.New()

	swaggerUI := swagger.Handler(swaggerFiles.Handler, swagger.URL("/swagger/internal.json"))
	app.Get("/swagger", swaggerUI)
	app.Get("/swagger/{any:path}", swaggerUI)

	e := httptest.New(t, app)

	e.GET("/swagger").Expect().Status(httptest.StatusOK).Body().
		Contains(`url: "\/swagger\/internal.json"`)

	e.GET("/swagger/internal.json").Expect().Status(httptest.StatusOK).
		HasContentType("application/json").JSON().Path("$.info.title").IsEqual("Internal API")
	// The default one is still there.
	e.GET("/swagger/swagger.json").Expect().Status(httptest.StatusOK).
		JSON().Path("$.info.title").IsEqual("Stores API")

	// A name that no docs package has registered.
	e.GET("/swagger/unknown.json").Expect().Status(httptest.StatusInternalServerError)
}

// Two pages in one application, each under its own prefix.
func TestHandlerPrefix(t *testing.T) {
	app := iris.New()

	public := swagger.Handler(swaggerFiles.Handler, swagger.Prefix("/swagger"))
	app.Get("/swagger", public)
	app.Get("/swagger/{any:path}", public)

	internal := swagger.Handler(swaggerFiles.Handler,
		swagger.Prefix("/internal/docs"),
		swagger.URL("/internal/docs/internal.json"),
	)
	app.Get("/internal/docs", internal)
	app.Get("/internal/docs/{any:path}", internal)

	e := httptest.New(t, app)

	css := file(t, "/swagger-ui.css")

	e.GET("/swagger").Expect().Status(httptest.StatusOK).Body().
		Contains(`href="/swagger/swagger-ui.css"`).
		Contains(`url: "\/swagger\/swagger.json"`)
	e.GET("/swagger/swagger-ui.css").Expect().Status(httptest.StatusOK).Body().IsEqual(css)

	e.GET("/internal/docs").Expect().Status(httptest.StatusOK).Body().
		Contains(`href="/internal/docs/swagger-ui.css"`).
		Contains(`src="/internal/docs/swagger-ui-bundle.js"`).
		Contains(`url: "\/internal\/docs\/internal.json"`)
	e.GET("/internal/docs/swagger-ui.css").Expect().Status(httptest.StatusOK).Body().IsEqual(css)
	e.GET("/internal/docs/internal.json").Expect().Status(httptest.StatusOK).
		JSON().Path("$.info.title").IsEqual("Internal API")

	// The handler it was given is left as it was.
	if got := swaggerFiles.Handler.Prefix; got != "" {
		t.Fatalf("swaggerFiles.Handler.Prefix was changed to %q", got)
	}
}

// The "." prefix makes every link of the page relative,
// for a page that is served from the root of a host or of a subdomain.
func TestHandlerRelativePrefix(t *testing.T) {
	app := iris.New()

	swaggerUI := swagger.Handler(swaggerFiles.Handler, swagger.Prefix("."))
	app.Get("/", swaggerUI)
	app.Get("/{any:path}", swaggerUI)

	e := httptest.New(t, app)

	for _, path := range []string{"/", "/index.html"} {
		e.GET(path).Expect().Status(httptest.StatusOK).
			HasContentType("text/html", "utf-8").Body().
			Contains(`href="./swagger-ui.css"`).
			Contains(`src="./swagger-ui-bundle.js"`).
			Contains(`url: ".\/swagger.json"`)
	}

	e.GET("/swagger-ui.css").Expect().Status(httptest.StatusOK).
		Body().IsEqual(file(t, "/swagger-ui.css"))
	e.GET("/swagger.json").Expect().Status(httptest.StatusOK).
		JSON().Object().HasValue("swagger", "2.0")
}

func TestHandlerOptions(t *testing.T) {
	app := iris.New()

	swaggerUI := swagger.Handler(swaggerFiles.Handler,
		swagger.URL("/swagger/swagger.json"),
		swagger.Prefix("/swagger"),
		swagger.DeepLinking(false),
		swagger.DocExpansion("none"),
		// The id of the element that the page has for the UI.
		swagger.DomID("#swagger-ui"),
		swagger.FontCDN("https://fonts.example.com/"),
		swagger.SetTheme(swagger.Monokai),
	)
	app.Get("/swagger", swaggerUI)
	app.Get("/swagger/{any:path}", swaggerUI)

	e := httptest.New(t, app)

	e.GET("/swagger").Expect().Status(httptest.StatusOK).Body().
		Contains(`deepLinking:  false`).
		Contains(`docExpansion: "none"`).
		Contains(`dom_id: "#swagger-ui"`).
		Contains(`href="https://fonts.example.com/css?family=`).
		Contains(`href="/swagger/theme-monokai.css"`)

	e.GET("/swagger/theme-monokai.css").Expect().Status(httptest.StatusOK).
		HasContentType("text/css", "utf-8").Body().IsEqual(file(t, "/theme-monokai.css"))
}

// Every theme has a file behind it.
func TestThemes(t *testing.T) {
	themes := []swagger.Theme{
		swagger.FeelingBlue, swagger.Material, swagger.Muted, swagger.Outline,
		swagger.Flattop, swagger.Monokai, swagger.Newspaper,
	}

	for _, theme := range themes {
		if !theme.IsValid() {
			t.Errorf("%s: expected a valid theme", theme)
		}

		if _, err := swaggerFiles.ReadFile("/" + theme.String() + ".css"); err != nil {
			t.Errorf("%s: %v", theme, err)
		}
	}

	if swagger.Unknow.IsValid() {
		t.Error("Unknow: expected an invalid theme")
	}
}

// A Config value is an option too.
func TestHandlerConfig(t *testing.T) {
	app := iris.New()

	swaggerUI := swagger.Handler(swaggerFiles.Handler, swagger.Config{
		URL:         "/swagger/doc.json",
		Prefix:      "/swagger",
		DeepLinking: true,
		Theme:       swagger.Material,
		FontCDN:     "https://fonts.example.com/",
	})
	app.Get("/swagger", swaggerUI)
	app.Get("/swagger/{any:path}", swaggerUI)

	e := httptest.New(t, app)

	e.GET("/swagger").Expect().Status(httptest.StatusOK).Body().
		Contains(`url: "\/swagger\/doc.json"`).
		Contains(`deepLinking:  true`).
		// Filter is a boolean, so a literal without it turns it off.
		Contains(`filter:  false`).
		// The fields that were left empty have their defaults.
		Contains(`docExpansion: "list"`).
		Contains(`dom_id: "#swagger-ui"`).
		Contains(`href="https://fonts.example.com/css?family=`).
		Contains(`href="/swagger/theme-material.css"`)

	// A literal that says nothing about the font or the theme keeps both.
	app = iris.New()

	swaggerUI = swagger.Handler(swaggerFiles.Handler,
		swagger.SetTheme(swagger.Muted),
		swagger.Config{Prefix: "/swagger"},
	)
	app.Get("/swagger", swaggerUI)

	e = httptest.New(t, app)

	e.GET("/swagger").Expect().Status(httptest.StatusOK).Body().
		Contains(`url: "\/swagger\/swagger.json"`).
		Contains(`href="https://fonts.googleapis.com/css?family=`).
		Contains(`href="/swagger/theme-muted.css"`)
}

// The files are in a webdav file system that all the pages of the process
// share. A route which lets more than GET through must not make it writable.
func TestHandlerIsReadOnly(t *testing.T) {
	app := iris.New()

	swaggerUI := swagger.Handler(swaggerFiles.Handler)
	app.Any("/swagger", swaggerUI)
	app.Any("/swagger/{any:path}", swaggerUI)

	e := httptest.New(t, app)

	css := file(t, "/swagger-ui.css")

	e.PUT("/swagger/swagger-ui.css").WithText("body{display:none}").
		Expect().Status(httptest.StatusMethodNotAllowed).Header("Allow").IsEqual("GET, HEAD")
	e.PUT("/swagger/new.js").WithText("alert(1)").
		Expect().Status(httptest.StatusMethodNotAllowed)
	e.DELETE("/swagger/swagger-ui.css").
		Expect().Status(httptest.StatusMethodNotAllowed)
	e.POST("/swagger/swagger-ui.css").
		Expect().Status(httptest.StatusMethodNotAllowed)
	e.POST("/swagger/swagger.json").
		Expect().Status(httptest.StatusMethodNotAllowed)

	if got := file(t, "/swagger-ui.css"); got != css {
		t.Fatal("the embedded swagger-ui.css was changed by a request")
	}

	if _, err := swaggerFiles.ReadFile("/new.js"); err == nil {
		t.Fatal("a request added a file next to the embedded ones")
	}

	e.GET("/swagger/swagger-ui.css").Expect().Status(httptest.StatusOK).Body().IsEqual(css)
	e.GET("/swagger/new.js").Expect().Status(httptest.StatusNotFound)

	// HEAD is answered like GET, without the body.
	e.HEAD("/swagger").Expect().Status(httptest.StatusOK).
		HasContentType("text/html", "utf-8")
	e.HEAD("/swagger/swagger-ui.css").Expect().Status(httptest.StatusOK).
		HasContentType("text/css", "utf-8").Body().IsEmpty()
}

// The page, its files and the definition go through the compression
// middleware like any other response.
func TestHandlerCompression(t *testing.T) {
	app := iris.New()
	app.Use(compress.Handler)

	swaggerUI := swagger.Handler(swaggerFiles.Handler)
	app.Get("/swagger", swaggerUI)
	app.Get("/swagger/{any:path}", swaggerUI)

	e := httptest.New(t, app)

	gunzip := func(path string) string {
		t.Helper()

		resp := e.GET(path).WithHeader("Accept-Encoding", "gzip").
			Expect().Status(httptest.StatusOK).HasContentEncoding("gzip")

		r, err := gzip.NewReader(strings.NewReader(resp.Body().Raw()))
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}

		b, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}

		return string(b)
	}

	if got := gunzip("/swagger/swagger-ui-bundle.js"); got != file(t, "/swagger-ui-bundle.js") {
		t.Errorf("swagger-ui-bundle.js: %d bytes after gunzip, which is not the embedded file", len(got))
	}

	if got := gunzip("/swagger"); !strings.Contains(got, `<div id="swagger-ui"></div>`) {
		t.Errorf("the page is not the Swagger UI one after gunzip: %d bytes", len(got))
	}

	if got := gunzip("/swagger/swagger.json"); !strings.Contains(got, `"/stores/{store_id}"`) {
		t.Errorf("the definition is not the registered one after gunzip: %s", got)
	}
}

func TestDisablingHandler(t *testing.T) {
	const envName = "SWAGGER_TEST_DISABLED"

	newApp := func() *iris.Application {
		app := iris.New()

		swaggerUI := swagger.DisablingHandler(swaggerFiles.Handler, envName,
			swagger.URL("/swagger/swagger.json"),
			swagger.Prefix("/swagger"),
		)
		app.Get("/swagger", swaggerUI)
		app.Get("/swagger/{any:path}", swaggerUI)

		return app
	}

	// The variable is not set: the page is there.
	t.Setenv(envName, "")
	e := httptest.New(t, newApp())

	e.GET("/swagger").Expect().Status(httptest.StatusOK).
		Body().Contains(`<div id="swagger-ui"></div>`)
	e.GET("/swagger/swagger-ui.css").Expect().Status(httptest.StatusOK)
	e.GET("/swagger/swagger.json").Expect().Status(httptest.StatusOK)

	// The variable is set, to any value: nothing is.
	t.Setenv(envName, "1")
	e = httptest.New(t, newApp())

	e.GET("/swagger").Expect().Status(httptest.StatusNotFound).Body().IsEmpty()
	e.GET("/swagger/index.html").Expect().Status(httptest.StatusNotFound)
	e.GET("/swagger/swagger-ui.css").Expect().Status(httptest.StatusNotFound)
	e.GET("/swagger/swagger.json").Expect().Status(httptest.StatusNotFound)
}
