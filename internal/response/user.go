package response

import "pr-reviewer-service/internal/dto"

type UserResponse struct {
	User dto.UserDTO `json:"user"`
}

type AllUsersResponse struct {
	Users []dto.UserDTO `json:"users"`
	Count int           `json:"count"`
}
