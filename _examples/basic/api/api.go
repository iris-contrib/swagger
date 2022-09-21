package api

import (
	"github.com/kataras/iris/v12"
)

// @Summary Add a new pet to the store
// @Description get string by ID
// @Accept  json
// @Produce  json
// @Param   some_id     path    int     true        "Some ID"
// @Success 200 {string} string	"ok"
// @Failure 400 {object} web.APIError "We need ID!!"
// @Failure 404 {object} web.APIError "Can not find ID"
// @Router /testapi/get-string-by-int/{some_id} [get]
func GetStringByInt(ctx iris.Context) {
	// err := web.APIError{}
	// fmt.Println(err)

	id := ctx.Params().GetIntDefault("some_id", 0)
	ctx.JSON(Pet3{
		ID: id,
	})
}

// @Description get struct array by ID
// @Accept  json
// @Produce  json
// @Param   some_id     path    string     true        "Some ID"
// @Param   offset     query    int     true        "Offset"
// @Param   limit      query    int     true        "Offset"
// @Success 200 {string} string	"ok"
// @Failure 400 {object} web.APIError "We need ID!!"
// @Failure 404 {object} web.APIError "Can not find ID"
// @Router /testapi/get-struct-array-by-string/{some_id} [get]
func GetStructArrayByString(ctx iris.Context) {
	id := ctx.Params().GetIntDefault("some_id", 0)
	ctx.Writef("OK: GetStructArrayByString:  %d", id)
}

type Pet3 struct {
	ID int `json:"id"`
}
