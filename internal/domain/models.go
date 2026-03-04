package domain

import "time"

// User represents a user in the system
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

// Product represents a product in the catalog
type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Price       float64   `json:"price"`
	Stock       int32     `json:"stock"`
	Category    string    `json:"category"`
	Status      string    `json:"status"`
	SellerID    string    `json:"seller_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Order represents an order
type Order struct {
	ID             string      `json:"id"`
	UserID         string      `json:"user_id"`
	Status         string      `json:"status"`
	PromoCodeID    *string     `json:"promo_code_id,omitempty"`
	TotalAmount    float64     `json:"total_amount"`
	DiscountAmount float64     `json:"discount_amount"`
	Items          []OrderItem `json:"items"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	ID           string  `json:"id"`
	OrderID      string  `json:"order_id"`
	ProductID    string  `json:"product_id"`
	Quantity     int32   `json:"quantity"`
	PriceAtOrder float64 `json:"price_at_order"`
}

// PromoCode represents a promotional code
type PromoCode struct {
	ID             string    `json:"id"`
	Code           string    `json:"code"`
	DiscountType   string    `json:"discount_type"`
	DiscountValue  float64   `json:"discount_value"`
	MinOrderAmount float64   `json:"min_order_amount"`
	MaxUses        int32     `json:"max_uses"`
	CurrentUses    int32     `json:"current_uses"`
	ValidFrom      time.Time `json:"valid_from"`
	ValidUntil     time.Time `json:"valid_until"`
	Active         bool      `json:"active"`
	CreatedAt      time.Time `json:"created_at"`
}

// BusinessError represents a business logic error
type BusinessError struct {
	Code    string
	Message string
	Details map[string]interface{}
}

func (e *BusinessError) Error() string {
	return e.Message
}
