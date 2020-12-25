# Swagger for the Iris web framework

[Iris](https://github.com/kataras/iris) middleware to automatically generate RESTful API documentation with Swagger 2.0 as requested at [#1231](https://github.com/kataras/iris/issues/1231).

[![Travis branch](https://img.shields.io/travis/iris-contrib/swagger/v12.svg)](https://travis-ci.org/iris-contrib/swagger)
[![Go Report Card](https://goreportcard.com/badge/github.com/iris-contrib/swagger)](https://goreportcard.com/report/github.com/iris-contrib/swagger)
[![GoDoc](https://godoc.org/github.com/iris-contrib/swagger?status.svg)](https://pkg.go.dev/github.com/iris-contrib/swagger)

## Usage

### Start using it

1. Add comments to your API source code, [See Declarative Comments Format](https://swaggo.github.io/swaggo.io/declarative_comments_format/).
2. Download [Swag](https://github.com/swaggo/swag) for Go by using:

```sh
$ go get -u github.com/swaggo/swag/cmd/swag
```

3. Run the [Swag](https://github.com/swaggo/swag) in your Go project root folder which contains `main.go` file, [Swag](https://github.com/swaggo/swag) will parse comments and generate required files(`docs` folder and `docs/doc.go`).

```sh
$ swag init
```

4. Download [swagger for Iris](https://github.com/iris-contrib/swagger) by using:

```sh
$ go get github.com/iris-contrib/swagger/v12@master
```

And import following in your code:

```go
import "github.com/iris-contrib/swagger/v12" // swagger middleware for Iris 
import "github.com/iris-contrib/swagger/v12/swaggerFiles" // swagger embed files

```

### Example Code:

```go
package main

import (
    "github.com/kataras/iris/v12"

    "github.com/iris-contrib/swagger/v12"
    "github.com/iris-contrib/swagger/v12/swaggerFiles"

    _ "github.com/your_username/your_project/docs"
    // docs folder should be generated by Swag CLI (swag init),
    // you have to import it.
)

// @title Swagger Example API
// @version 1.0
// @description This is a sample server Petstore server.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /v2
func main() {
    app := iris.New()

    config := swagger.Config{
        // The url pointing to API definition.
        URL:          "http://localhost:8080/swagger/doc.json",
        DeepLinking:  true,
        DocExpansion: "list",
        DomID:        "#swagger-ui",
        // The UI prefix URL (see route).
        Prefix:       "/swagger",
    }
    swaggerUI := swagger.Handler(swaggerFiles.Handler, config)

    // Register on http://localhost:8080/swagger
    app.Get("/swagger", swaggerUI)
    // And the wildcard one for index.html, *.js, *.css and e.t.c.
    app.Get("/swagger/{any:path}", swaggerUI)

    app.Listen(":8080")
}
```

5. Run it, and browse to http://localhost:8080/swagger/index.html, you can see Swagger 2.0 API documentation.

![swagger_index.html](example.png)

6. If you want to disable swagger when some environment variable is set, use `DisablingHandler` instead of `Handler`.

```go
swagger.DisablingHandler(swaggerFiles.Handler, config)
```
