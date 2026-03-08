package request

type GeoCoderRequest struct{
	KeyWord string `json:"keyword" validate:"required"`
}