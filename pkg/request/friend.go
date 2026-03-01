package request

type AddFriendRequest struct {
    UserId string `json:"userId" validate:"required"`
}