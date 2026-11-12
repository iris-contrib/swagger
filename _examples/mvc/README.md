# MVC

A controller of the Iris `controller/mvc` package with its routes in the Swagger UI. The comments that Swag reads are on the methods of the controller, in [store_controller.go](store_controller.go). The general information of the API is above `main`, in [main.go](main.go).

```sh
$ go install github.com/swaggo/swag/cmd/swag@latest
$ swag init
$ go run .
```

Navigate to <http://localhost:8080/swagger>.

`swag init` writes the `docs` directory. Run it again after a comment changes. `go test` fails as long as `docs` is behind the comments.
