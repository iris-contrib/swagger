package swagger

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/iris-contrib/swagger/v12/swaggerFiles"
	"github.com/kataras/iris/v12"
	"github.com/stretchr/testify/assert"

	_ "github.com/iris-contrib/swagger/v12/example/basic/docs"
)

func TestWrapHandler(t *testing.T) {
	app := iris.New()

	app.Get("/{any:path}", WrapHandler(swaggerFiles.Handler))

	w1 := performRequest("GET", "/index.html", app)
	assert.Equal(t, 200, w1.Code)
}

func TestWrapCustomHandler(t *testing.T) {
	app := iris.New()

	app.Get("/{any:path}", CustomWrapHandler(&Config{}, swaggerFiles.Handler))

	w1 := performRequest("GET", "/index.html", app)
	assert.Equal(t, 200, w1.Code)

	w2 := performRequest("GET", "/doc.json", app)
	assert.Equal(t, 200, w2.Code)

	w3 := performRequest("GET", "/favicon-16x16.png", app)
	assert.Equal(t, 200, w3.Code)

	w4 := performRequest("GET", "/notfound", app)
	assert.Equal(t, 404, w4.Code)
}

func TestDisablingWrapHandler(t *testing.T) {
	app := iris.New()
	disablingKey := "SWAGGER_DISABLE"

	app.Get("/simple/{any:path}", DisablingWrapHandler(swaggerFiles.Handler, disablingKey))

	w1 := performRequest("GET", "/simple/index.html", app)
	assert.Equal(t, 200, w1.Code)

	w2 := performRequest("GET", "/simple/doc.json", app)
	assert.Equal(t, 200, w2.Code)

	w3 := performRequest("GET", "/simple/favicon-16x16.png", app)
	assert.Equal(t, 200, w3.Code)

	w4 := performRequest("GET", "/simple/notfound", app)
	assert.Equal(t, 404, w4.Code)

	os.Setenv(disablingKey, "true")

	app.Get("/disabling/{any:path}", DisablingWrapHandler(swaggerFiles.Handler, disablingKey))

	w11 := performRequest("GET", "/disabling/index.html", app)
	assert.Equal(t, 404, w11.Code)

	w22 := performRequest("GET", "/disabling/doc.json", app)
	assert.Equal(t, 404, w22.Code)

	w33 := performRequest("GET", "/disabling/favicon-16x16.png", app)
	assert.Equal(t, 404, w33.Code)

	w44 := performRequest("GET", "/disabling/notfound", app)
	assert.Equal(t, 404, w44.Code)
}

func TestDisablingCustomWrapHandler(t *testing.T) {
	app := iris.New()
	disablingKey := "SWAGGER_DISABLE2"

	app.Get("/simple/{any:path}", DisablingCustomWrapHandler(&Config{}, swaggerFiles.Handler, disablingKey))

	w1 := performRequest("GET", "/simple/index.html", app)
	assert.Equal(t, 200, w1.Code)

	os.Setenv(disablingKey, "true")

	app.Get("/disabling/{any:path}", DisablingCustomWrapHandler(&Config{}, swaggerFiles.Handler, disablingKey))

	w11 := performRequest("GET", "/disabling/index.html", app)
	assert.Equal(t, 404, w11.Code)
}

func TestWithGzipMiddleware(t *testing.T) {
	app := iris.New()
	app.Use(iris.Gzip)

	app.Get("/{any:path}", WrapHandler(swaggerFiles.Handler))

	w1 := performRequest("GET", "/index.html", app)
	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, w1.Header()["Content-Type"][0], "text/html; charset=UTF-8")

	w2 := performRequest("GET", "/swagger-ui.css", app)
	assert.Equal(t, 200, w2.Code)
	assert.Equal(t, w2.Header()["Content-Type"][0], "text/css; charset=UTF-8")

	w3 := performRequest("GET", "/swagger-ui-bundle.js", app)
	assert.Equal(t, 200, w3.Code)
	assert.Equal(t, w3.Header()["Content-Type"][0], "application/javascript; charset=UTF-8")

	w4 := performRequest("GET", "/doc.json", app)
	assert.Equal(t, 200, w4.Code)
	assert.Equal(t, w4.Header()["Content-Type"][0], "application/json; charset=UTF-8")
}

func performRequest(method, target string, app *iris.Application) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, target, nil)
	w := httptest.NewRecorder()
	app.Build()
	app.ServeHTTP(w, r)
	return w
}
