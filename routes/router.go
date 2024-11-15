package routes

import (
	"go-aws/controllers"

	"github.com/gorilla/mux"
)

func InitRoutes(s3Controller *controllers.S3Properties) *mux.Router {
	router := mux.NewRouter()

	// s3 routes
	router.HandleFunc("/initiate-multipart-upload", s3Controller.InitiateMultipartUpload).Methods("POST")
	router.HandleFunc("/generate-presigned-urls", s3Controller.GeneratePresignedUrl).Methods("POST")
	router.HandleFunc("/complete-multipart-upload", s3Controller.CompleteMultipartUpload).Methods("POST")
	router.HandleFunc("/list-documents", s3Controller.List).Methods("GET")
	router.HandleFunc("/remove-document", s3Controller.Remove).Methods("DELETE")
	router.HandleFunc("/rename-document", s3Controller.Rename).Methods("PATCH")
	router.HandleFunc("/generate-share-url", s3Controller.Share).Methods("GET")

	return router
}
