// A controller of the Iris controller/mvc package, documented with swag comments.
//
// The docs package is generated from the comments of this directory:
//
//	go install github.com/swaggo/swag/cmd/swag@latest
//	swag init
//
// Run:
//
//	go run .
//
// And open http://localhost:8080/swagger in a browser.
package main

import (
	"github.com/iris-contrib/swagger/v14"
	"github.com/iris-contrib/swagger/v14/swaggerFiles"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/controller/mvc"

	// The package that "swag init" writes.
	// Its init function registers the API definition, so it has to be imported.
	_ "github.com/iris-contrib/swagger/v14/_examples/mvc/docs"
)

// The general information of the API is written once, above the main function.
// The comments of each operation are on the methods of the controller,
// see store_controller.go.

// @title Stores API
// @version 1.0
// @description A controller of the Iris controller/mvc package, documented with swag comments.

// @license.name MIT
// @license.url https://github.com/iris-contrib/swagger/blob/master/LICENSE

// @BasePath /
func main() {
	app := newApp()
	app.Listen(":8080")
}

func newApp() *iris.Application {
	app := iris.New()

	// The controller has a Repository field of this type, Iris sets it.
	if err := app.RegisterDependency(NewStoreRepository()); err != nil {
		panic(err)
	}

	// The five methods of the controller become five routes:
	//
	//	POST   /stores             Create
	//	GET    /stores/{store_id}  Get
	//	PUT    /stores/{store_id}  Update
	//	DELETE /stores/{store_id}  Delete
	//	POST   /stores/list        List
	//
	// Its Routing method adds one more, GET /stores/count.
	app.Router("/stores", mvc.UseController[*StoreController]())

	// The Swagger UI page, its files and the API definition.
	swaggerUI := swagger.Handler(swaggerFiles.Handler,
		swagger.URL("/swagger/swagger.json"), // The url pointing to API definition.
		swagger.DeepLinking(true),
		swagger.Prefix("/swagger"),
	)
	// Register on http://localhost:8080/swagger
	app.Get("/swagger", swaggerUI)
	// And http://localhost:8080/swagger/index.html, *.js, *.css and e.t.c.
	app.Get("/swagger/{any:path}", swaggerUI)

	return app
}
