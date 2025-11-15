package request

type LoginRequest struct {
	UserID string `json:"user_id" validate:"required"`
}
