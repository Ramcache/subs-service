package subscription

type CreateRequest struct {
	ServiceName string  `json:"service_name"`
	Price       int64   `json:"price"`
	UserID      string  `json:"user_id"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date,omitempty"`
}

type UpdateRequest struct {
	ServiceName *string `json:"service_name,omitempty"`
	Price       *int64  `json:"price,omitempty"`
	UserID      *string `json:"user_id,omitempty"`
	StartDate   *string `json:"start_date,omitempty"`
	EndDate     *string `json:"end_date,omitempty"`
}

type SubscriptionResponse struct {
	ID          string  `json:"id"`
	ServiceName string  `json:"service_name"`
	Price       int64   `json:"price"`
	UserID      string  `json:"user_id"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date,omitempty"`
}

type ListResponse struct {
	Items  []SubscriptionResponse `json:"items"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
	Total  int64                  `json:"total"`
}

type TotalResponse struct {
	From        string  `json:"from"`
	To          string  `json:"to"`
	UserID      *string `json:"user_id,omitempty"`
	ServiceName *string `json:"service_name,omitempty"`
	Total       int64   `json:"total"`
}

type apiError struct {
	Code    string `json:"code" example:"validation_error"`
	Message string `json:"message" example:"service_name is required"`
}
