package swagger

import (
	"html/template"
	"os"
	"strings"

	"golang.org/x/net/webdav"

	"github.com/kataras/iris/v12"
	"github.com/swaggo/swag"
)

// Configurator represents a configuration setter.
type Configurator interface {
	Configure(*Config)
}

// ConfiguratorFunc implements the Configuration as a function type.
type ConfiguratorFunc func(*Config)

// Configure calls itself and modifies the default config.
func (fn ConfiguratorFunc) Configure(config *Config) {
	fn(config)
}

// Config stores swagger configuration variables.
type Config struct {
	// The URL pointing to API definition (normally swagger.json or swagger.yaml).
	// Default is `doc.json`.
	URL string
	// The prefix url which this swagger ui is registered on.
	// Defaults to "/swagger". It can be a "." too.
	Prefix       string
	FontCDN      string
	DeepLinking  bool
	DocExpansion string
	DomID        string
	// Enabling tag Filtering
	Filter bool
}

// Configure completes the Configurator interface.
// It allows to pass a Config as it is and override any option.
func (c Config) Configure(config *Config) {
	config.URL = c.URL
	config.Prefix = c.Prefix
	config.DeepLinking = c.DeepLinking
	config.DocExpansion = c.DocExpansion
	config.DomID = c.DomID
	config.Filter = c.Filter
}

// URL presents the URL pointing to API definition (normally swagger.json or swagger.yaml).
func URL(url string) ConfiguratorFunc {
	return func(c *Config) {
		c.URL = url
	}
}

// Prefix presents the URL prefix of this swagger UI (normally "/swagger" or ".").
func Prefix(prefix string) ConfiguratorFunc {
	return func(c *Config) {
		c.Prefix = prefix
	}
}

// Change google font cdn to any you like.
func FontCDN(cdn string) ConfiguratorFunc {
	cdn = strings.TrimSuffix(cdn, "/")
	return func(c *Config) {
		c.FontCDN = cdn
	}
}

// DocExpansion list, full, none.
func DocExpansion(docExpansion string) ConfiguratorFunc {
	return func(c *Config) {
		c.DocExpansion = docExpansion
	}
}

// DomID #swagger-ui.
func DomID(domID string) ConfiguratorFunc {
	return func(c *Config) {
		c.DomID = domID
	}
}

// DeepLinking set the swagger deeplinking configuration.
func DeepLinking(deepLinking bool) ConfiguratorFunc {
	return func(c *Config) {
		c.DeepLinking = deepLinking
	}
}

// Handler wraps the webdav http handler into an Iris Handler one.
//
// Usage:
//
//	swaggerUI := swagger.Handler(swaggerFiles.Handler,
//	 swagger.URL("http://localhost:8080/swagger/doc.json"), // The url pointing to API definition))
//	 swagger.DeepLinking(true),
//	 swagger.Prefix("/swagger"),
//	)
//	app.Get("/swagger", swaggerUI)
//	app.Get("/swagger/{any:path}", swaggerUI)
//
// OR
//
//	swaggerUI := swagger.Handler(swaggerFiles.Handler, swagger.Config{
//	 URL: ...,
//	 Prefix: ...,
//	 DeepLinking: ...,
//	 DocExpansion: ...,
//	 DomID: ...,
//	}
func Handler(h *webdav.Handler, configurators ...Configurator) iris.Handler {
	config := &Config{
		URL:          "doc.json",
		DeepLinking:  true,
		DocExpansion: "list",
		DomID:        "#swagger-ui",
		Prefix:       "/swagger",
		FontCDN:      "https://fonts.googleapis.com",
		Filter:       true,
	}

	for _, c := range configurators {
		c.Configure(config)
	}

	if prefix := config.Prefix; prefix != "" && prefix != "." {
		h.Prefix = prefix
	} else {
		config.Prefix = "." // relative files and don't touch the webdav one's (index swagger will not work without index.html).
	}

	handler := func(ctx iris.Context) {
		path := strings.TrimPrefix(ctx.Path(), config.Prefix)
		if sufIdx := strings.LastIndexByte(path, '.'); sufIdx > 0 {
			suffix := path[sufIdx:]
			switch suffix {
			case ".html":
				ctx.ContentType("text/html; charset=utf-8")
			case ".css":
				ctx.ContentType("text/css; charset=utf-8")
			case ".js":
				ctx.ContentType("application/javascript")
			case ".json":
				ctx.ContentType("application/json")
			}
		}

		switch path {
		case "", "/", "/index.html":
			ctx.ContentType("text/html; charset=utf-8")
			err := indexTmpl.Execute(ctx, config)
			if err != nil {
				ctx.Application().Logger().Errorf("swagger: %v", err)
				ctx.StopWithStatus(iris.StatusInternalServerError)
				return
			}
		case "/doc.json":
			doc, err := swag.ReadDoc()
			if err != nil {
				ctx.Application().Logger().Errorf("swagger: %v", err)
				ctx.StopWithStatus(iris.StatusInternalServerError)
				return
			}
			ctx.WriteString(doc)
		default:
			h.ServeHTTP(ctx.ResponseWriter(), ctx.Request())
		}
	}

	return handler
}

// DisablingHandler turns handler off
// if specified environment variable passed.
func DisablingHandler(h *webdav.Handler, envName string, configurators ...Configurator) iris.Handler {
	eFlag := os.Getenv(envName)
	if eFlag != "" {
		return func(ctx iris.Context) {
			// Simulate behavior when route unspecified and
			// return 404 HTTP code
			ctx.NotFound()
		}
	}

	return Handler(h, configurators...)
}

var indexTmpl = template.Must(template.New("swagger_index.html").Parse(`<!-- HTML for static distribution bundle build -->
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Swagger UI</title>
  <link href="{{.FontCDN}}/css?family=Open+Sans:400,700|Source+Code+Pro:300,600|Titillium+Web:400,600,700" rel="stylesheet">
  <link rel="stylesheet" type="text/css" href="{{.Prefix}}/swagger-ui.css" >
  <link rel="icon" type="image/png" href="{{.Prefix}}/favicon-32x32.png" sizes="32x32" />
  <link rel="icon" type="image/png" href="{{.Prefix}}/favicon-16x16.png" sizes="16x16" />
  <style>
    html
    {
        box-sizing: border-box;
        overflow: -moz-scrollbars-vertical;
        overflow-y: scroll;
    }
    *,
    *:before,
    *:after
    {
        box-sizing: inherit;
    }
    body {
      margin:0;
      background: #fafafa;
    }
  </style>
</head>
<body>
<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" style="position:absolute;width:0;height:0">
  <defs>
    <symbol viewBox="0 0 20 20" id="unlocked">
      <path d="M15.8 8H14V5.6C14 2.703 12.665 1 10 1 7.334 1 6 2.703 6 5.6V6h2v-.801C8 3.754 8.797 3 10 3c1.203 0 2 .754 2 2.199V8H4c-.553 0-1 .646-1 1.199V17c0 .549.428 1.139.951 1.307l1.197.387C5.672 18.861 6.55 19 7.1 19h5.8c.549 0 1.428-.139 1.951-.307l1.196-.387c.524-.167.953-.757.953-1.306V9.199C17 8.646 16.352 8 15.8 8z"></path>
    </symbol>
    <symbol viewBox="0 0 20 20" id="locked">
      <path d="M15.8 8H14V5.6C14 2.703 12.665 1 10 1 7.334 1 6 2.703 6 5.6V8H4c-.553 0-1 .646-1 1.199V17c0 .549.428 1.139.951 1.307l1.197.387C5.672 18.861 6.55 19 7.1 19h5.8c.549 0 1.428-.139 1.951-.307l1.196-.387c.524-.167.953-.757.953-1.306V9.199C17 8.646 16.352 8 15.8 8zM12 8H8V5.199C8 3.754 8.797 3 10 3c1.203 0 2 .754 2 2.199V8z"/>
    </symbol>
    <symbol viewBox="0 0 20 20" id="close">
      <path d="M14.348 14.849c-.469.469-1.229.469-1.697 0L10 11.819l-2.651 3.029c-.469.469-1.229.469-1.697 0-.469-.469-.469-1.229 0-1.697l2.758-3.15-2.759-3.152c-.469-.469-.469-1.228 0-1.697.469-.469 1.228-.469 1.697 0L10 8.183l2.651-3.031c.469-.469 1.228-.469 1.697 0 .469.469.469 1.229 0 1.697l-2.758 3.152 2.758 3.15c.469.469.469 1.229 0 1.698z"/>
    </symbol>
    <symbol viewBox="0 0 20 20" id="large-arrow">
      <path d="M13.25 10L6.109 2.58c-.268-.27-.268-.707 0-.979.268-.27.701-.27.969 0l7.83 7.908c.268.271.268.709 0 .979l-7.83 7.908c-.268.271-.701.27-.969 0-.268-.269-.268-.707 0-.979L13.25 10z"/>
    </symbol>
    <symbol viewBox="0 0 20 20" id="large-arrow-down">
      <path d="M17.418 6.109c.272-.268.709-.268.979 0s.271.701 0 .969l-7.908 7.83c-.27.268-.707.268-.979 0l-7.908-7.83c-.27-.268-.27-.701 0-.969.271-.268.709-.268.979 0L10 13.25l7.418-7.141z"/>
    </symbol>
    <symbol viewBox="0 0 24 24" id="jump-to">
      <path d="M19 7v4H5.83l3.58-3.59L8 6l-6 6 6 6 1.41-1.41L5.83 13H21V7z"/>
    </symbol>
    <symbol viewBox="0 0 24 24" id="expand">
      <path d="M10 18h4v-2h-4v2zM3 6v2h18V6H3zm3 7h12v-2H6v2z"/>
    </symbol>
  </defs>
</svg>
<div id="swagger-ui"></div>
<script src="{{.Prefix}}/swagger-ui-bundle.js"> </script>
<script src="{{.Prefix}}/swagger-ui-standalone-preset.js"> </script>
<script>
window.onload = function() {
  // Build a system
  const ui = SwaggerUIBundle({
    url: "{{.URL}}",
    deepLinking: {{.DeepLinking}},
    docExpansion: "{{.DocExpansion}}",
    dom_id: "{{.DomID}}",
    validatorUrl: null,
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIStandalonePreset
    ],
    plugins: [
      SwaggerUIBundle.plugins.DownloadUrl
    ],
    layout: "StandaloneLayout",
    filter: {{.Filter}}
  })
  window.ui = ui
}
</script>
</body>
</html>
`))
