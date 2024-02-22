package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"     // sw
	ginSwagger "github.com/swaggo/gin-swagger" // gin-swagger middleware
	"github.com/takeoff-capstone/async_functions"
	"github.com/takeoff-capstone/cloudfunctions"
	_ "github.com/takeoff-capstone/docs"
)

// @title Grocery API
// @version 1.0
// @description API for managing groceries
// @BasePath /api
func main() {

	// Create a new Gin router
	r := gin.Default()

	// Enable CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// Define the endpoint for creating a user
	// Define the endpoint for creating a user
	// @Summary Create a new grocery
	// @Description Create a new grocery with the provided data
	// @ID create-grocery
	// @Accept json
	// @Produce json
	// @Param data body cloudfunctions.GroceryData true "Grocery data"
	// @Success 200 {object} cloudfunctions.Grocery "OK"
	// @Router /api/CreateGrocery [post]
	r.POST("/api/CreateGrocery", func(c *gin.Context) {
		//Convert gin Context to http.ResponseWriter and *http.Request
		req := c.Request
		res := c.Writer

		//Call your CreateGrocery function passing http.ResponseWriter and *http.Request
		cloudfunctions.CreateGrocery(res, req)
	})
	r.PATCH("/api/UpdateGrocery", func(c *gin.Context) {
		//Convert gin Context to http.ResponseWriter and *http.Request
		req := c.Request
		res := c.Writer

		//Call your CreateGrocery function passing http.ResponseWriter and *http.Request
		cloudfunctions.UpdateGrocery(res, req)
	})
	r.DELETE("/api/DeleteGrocery", func(c *gin.Context) {
		//Convert gin Context to http.ResponseWriter and *http.Request
		req := c.Request
		res := c.Writer

		//Call your CreateGrocery function passing http.ResponseWriter and *http.Request
		cloudfunctions.DeleteGrocery(res, req)
	})
	r.GET("/api/GetGroceryByID", func(c *gin.Context) {
		//Convert gin Context to http.ResponseWriter and *http.Request
		req := c.Request
		res := c.Writer

		//Call your CreateGrocery function passing http.ResponseWriter and *http.Request
		cloudfunctions.GetGroceryByID(res, req)
	})
	r.GET("/api/ViewAllGroceries", func(c *gin.Context) {
		//Convert gin Context to http.ResponseWriter and *http.Request
		req := c.Request
		res := c.Writer

		//Call your CreateGrocery function passing http.ResponseWriter and *http.Request
		cloudfunctions.ViewAllGroceries(res, req)
	})
	r.POST("/api/BulkCreate", func(c *gin.Context) {
		//Convert gin Context to http.ResponseWriter and *http.Request
		req := c.Request
		res := c.Writer

		//Call your CreateGrocery function passing http.ResponseWriter and *http.Request
		cloudfunctions.BulkUploadGroceryItems(res, req)
	})
	r.POST("/api/downloadcsv", func(c *gin.Context) {
		//Convert gin Context to http.ResponseWriter and *http.Request
		req := c.Request
		res := c.Writer

		//Call your CreateGrocery function passing http.ResponseWriter and *http.Request
		async_functions.DownloadCSV(res, req)
	})
	//Swagger UI handler
	// url := httpSwagger.URL("/swagger/doc.json")
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	// r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	// http.Handle("/swagger/", httpSwagger.Handler(
	// 	httpSwagger.URL("/swagger/doc.json"), // URL to the generated Swagger JSON file
	// ))

	// // Initialize swag
	// , ginSwagger.URL("/swagger/doc.json"))

	http.Handle("/", r)
	r.Run(":8084")

}
