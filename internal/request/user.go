package request

type SetUserActiveRequest struct {
	UserID   string `json:"user_id" validate:"required,min=1,max=255"`
	IsActive bool   `json:"is_active"`
}

type BatchDeactivateUsersRequest struct {
	UserIDs []string `json:"user_ids" validate:"required,min=1,dive,required,min=1,max=255"`
}
