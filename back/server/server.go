package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {

		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.POST("/depot/publish", func(ctx *gin.Context) {
		f, err := ctx.FormFile("source")
		fmt.Println(err)
		fmt.Println(f)
	})
	r.Run(":8060") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
