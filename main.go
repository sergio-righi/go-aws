package main

import (
	"fmt"
	"go-aws/controllers"
	"go-aws/routes" // replace with actual package path
	"go-aws/utils"  // replace with actual package path
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// App struct holds the router and config
type App struct {
	Router *mux.Router
	Config *utils.Config
}

func main() {
	// Load configuration
	config, err := utils.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	s3Controller, _ := controllers.S3Controller(config)

	router := routes.InitRoutes(s3Controller)

	fmt.Println("Server is running on http://localhost:8080")
	http.ListenAndServe("localhost:8080", router)
}
