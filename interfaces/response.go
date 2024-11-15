package interfaces

type ApiResponse struct {
	Status  int         `json:"status"`
	Payload interface{} `json:"payload"`
}
