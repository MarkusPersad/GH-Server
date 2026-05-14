package request

type GeoCoderRequest struct{
	KeyWord string `json:"keyword" validate:"required"`
}

type RoutePlanRequest struct{
	Start string `json:"start" validate:"required"`
	End string `json:"end" validate:"required"`
	// 默认0 （0：最快路线，1：最短路线，2：避开高速，3：步行）
	Mode string `json:"mode" validate:"required"`
}