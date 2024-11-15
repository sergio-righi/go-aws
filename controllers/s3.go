package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"go-aws/interfaces"
	"go-aws/utils"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Controller handles S3 operations.
type S3Properties struct {
	s3Client   *s3.Client
	bucketName string
}

// NewS3Controller initializes a new S3Controller with S3 client and bucket name.
func S3Controller(config *utils.Config) (*S3Properties, error) {

	s3Client := s3.New(s3.Options{
		Region:       config.S3Region,
		BaseEndpoint: &config.S3Endpoint,
		UsePathStyle: true,
		Credentials: aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider(config.S3AccessKey, config.S3SecretKey, ""),
		),
	})

	return &S3Properties{
		s3Client:   s3Client,
		bucketName: config.S3BucketName,
	}, nil
}

// InitiateMultipartUpload starts a multipart upload and returns an upload ID.
func (sc *S3Properties) InitiateMultipartUpload(w http.ResponseWriter, r *http.Request) {
	var requestBody struct {
		FileName string `json:"fileName"`
	}
	json.NewDecoder(r.Body).Decode(&requestBody)

	params := &s3.CreateMultipartUploadInput{
		Bucket: &sc.bucketName,
		Key:    &requestBody.FileName,
	}
	resp, err := sc.s3Client.CreateMultipartUpload(r.Context(), params)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(interfaces.ApiResponse{
		Status: 200,
		Payload: map[string]interface{}{
			"id":  *resp.UploadId,
			"key": *resp.Key,
		},
	})
}

func (sc *S3Properties) GeneratePresignedUrl(w http.ResponseWriter, r *http.Request) {
	var requestBody struct {
		FileKey string `json:"fileKey"`
		FileId  string `json:"fileId"`
		Parts   int    `json:"parts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Slice to hold each presigned URL and part number
	urls := make([]interfaces.SignedUrl, requestBody.Parts)

	for i := 1; i <= requestBody.Parts; i++ {
		partNumber := int32(i)
		// Prepare parameters for presigned URL
		params := &s3.UploadPartInput{
			Bucket:     &sc.bucketName,
			Key:        &requestBody.FileKey,
			UploadId:   &requestBody.FileId,
			PartNumber: &partNumber,
		}

		// Generate presigned URL for each part
		presignClient := s3.NewPresignClient(sc.s3Client)
		presignedURL, err := presignClient.PresignUploadPart(context.TODO(), params, func(opts *s3.PresignOptions) {
			opts.Expires = 15 * time.Minute
		})
		if err != nil {
			http.Error(w, "Failed to generate presigned URL: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Append the URL to the response slice
		urls[i-1] = interfaces.SignedUrl{
			SignedUrl:  presignedURL.URL,
			PartNumber: partNumber,
		}
	}

	json.NewEncoder(w).Encode(interfaces.ApiResponse{
		Status:  200,
		Payload: urls,
	})
}

// CompleteMultipartUpload completes a multipart upload by assembling uploaded parts.
func (sc *S3Properties) CompleteMultipartUpload(w http.ResponseWriter, r *http.Request) {
	var requestBody struct {
		FileKey string                `json:"fileKey"`
		FileId  string                `json:"fileId"`
		Parts   []types.CompletedPart `json:"parts"`
	}
	json.NewDecoder(r.Body).Decode(&requestBody)

	// Sort parts by PartNumber
	sort.Slice(requestBody.Parts, func(i, j int) bool {
		return *requestBody.Parts[i].PartNumber < *requestBody.Parts[j].PartNumber
	})

	// Convert UploadedPart to s3.CompletedPart
	var completedParts []types.CompletedPart
	for _, part := range requestBody.Parts {
		completedParts = append(completedParts, types.CompletedPart{
			ETag:       part.ETag,
			PartNumber: part.PartNumber,
		})
	}

	// Complete multipart upload
	params := &s3.CompleteMultipartUploadInput{
		Bucket:   &sc.bucketName,
		Key:      &requestBody.FileKey,
		UploadId: &requestBody.FileId,
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	}
	_, err := sc.s3Client.CompleteMultipartUpload(r.Context(), params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get object size after upload completion
	headParams := &s3.HeadObjectInput{
		Bucket: &sc.bucketName,
		Key:    &requestBody.FileKey,
	}
	headResp, err := sc.s3Client.HeadObject(r.Context(), headParams)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send a response with the file key and size
	json.NewEncoder(w).Encode(interfaces.ApiResponse{
		Status: 200,
		Payload: map[string]interface{}{
			"key":  requestBody.FileKey,
			"size": headResp.ContentLength,
		},
	})
}

// List lists objects in the S3 bucket based on prefix and delimiter.
func (sc *S3Properties) List(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")
	delimiter := r.URL.Query().Get("delimiter")

	params := &s3.ListObjectsV2Input{
		Bucket:    &sc.bucketName,
		Prefix:    aws.String(prefix),
		Delimiter: aws.String(delimiter),
	}
	resp, err := sc.s3Client.ListObjectsV2(r.Context(), params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	objects := []types.Object{}
	for _, item := range resp.Contents {
		if (delimiter == "/" && strings.HasSuffix(*item.Key, "/")) || (delimiter != "/" && !strings.HasSuffix(*item.Key, "/")) {
			objects = append(objects, types.Object{
				Key:  item.Key,
				Size: item.Size,
			})
		}
	}
	json.NewEncoder(w).Encode(interfaces.ApiResponse{
		Status:  200,
		Payload: objects,
	})
}

// Remove deletes an object from the S3 bucket.
func (sc *S3Properties) Remove(w http.ResponseWriter, r *http.Request) {
	fileKey := r.URL.Query().Get("fileKey")

	params := &s3.DeleteObjectInput{
		Bucket: &sc.bucketName,
		Key:    aws.String(fileKey),
	}
	_, err := sc.s3Client.DeleteObject(r.Context(), params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(interfaces.ApiResponse{
		Status:  200,
		Payload: true,
	})
}

// Rename renames an object by copying it to a new key and deleting the old key.
func (sc *S3Properties) Rename(w http.ResponseWriter, r *http.Request) {
	oldFileKey := r.URL.Query().Get("oldFileKey")
	newFileKey := r.URL.Query().Get("newFileKey")

	copyParams := &s3.CopyObjectInput{
		Bucket:     &sc.bucketName,
		CopySource: aws.String(fmt.Sprintf("%s/%s", sc.bucketName, oldFileKey)),
		Key:        aws.String(newFileKey),
	}
	_, err := sc.s3Client.CopyObject(r.Context(), copyParams)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	deleteParams := &s3.DeleteObjectInput{
		Bucket: &sc.bucketName,
		Key:    aws.String(oldFileKey),
	}
	_, err = sc.s3Client.DeleteObject(r.Context(), deleteParams)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(interfaces.ApiResponse{
		Status:  200,
		Payload: true,
	})
}

// Share generates a presigned URL for an S3 object.
func (sc *S3Properties) Share(w http.ResponseWriter, r *http.Request) {
	// Extract query parameters
	fileKey := r.URL.Query().Get("fileKey")
	expiresInStr := r.URL.Query().Get("expiresIn")

	// Validate the fileKey
	if fileKey == "" {
		http.Error(w, "fileKey is required", http.StatusBadRequest)
		return
	}

	// Parse and validate expiration time
	expiresIn, err := strconv.Atoi(expiresInStr)
	if err != nil || expiresIn <= 0 {
		http.Error(w, "Invalid or missing expiresIn parameter", http.StatusBadRequest)
		return
	}

	// Prepare S3 get object input parameters
	params := &s3.GetObjectInput{
		Bucket:                     &sc.bucketName,
		Key:                        aws.String(fileKey),
		ResponseContentDisposition: aws.String("attachment"),
	}

	// Initialize the presign client
	presignClient := s3.NewPresignClient(sc.s3Client)

	// Generate the presigned URL
	presignedURL, err := presignClient.PresignGetObject(r.Context(), params, func(opt *s3.PresignOptions) {
		opt.Expires = time.Duration(expiresIn) * time.Second
	})
	if err != nil {
		http.Error(w, "Failed to generate presigned URL: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Send response with the presigned URL
	json.NewEncoder(w).Encode(interfaces.ApiResponse{
		Status:  200,
		Payload: presignedURL.URL,
	})
}
