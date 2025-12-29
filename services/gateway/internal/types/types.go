package types

type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=6,max=100"`
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type CreateProblemRequest struct {
	Title       string `json:"title" validate:"required"`
	Description string `json:"description"`
}

type JSONResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type CreateTestCaseRequest struct {
	InputData  string `json:"input_data" validate:"required"`
	OutputData string `json:"output_data" validate:"required"`
}
