// Package models contains generated OpenAPI models
// This file re-exports types from the generated code for use in internal packages
package models

import (
	gen "marketplace/internal/gen"
)

// Re-export generated types
type (
	// Product types
	ProductCreateRequest = gen.ProductCreateRequest
	ProductUpdateRequest = gen.ProductUpdateRequest
	ProductResponse      = gen.ProductResponse
	ProductPageResponse  = gen.ProductPageResponse
	ProductStatus        = gen.ProductStatus

	// Order types
	OrderCreateRequest = gen.OrderCreateRequest
	OrderUpdateRequest = gen.OrderUpdateRequest
	OrderResponse      = gen.OrderResponse
	OrderItemCreate    = gen.OrderItemCreate
	OrderItemResponse  = gen.OrderItemResponse
	OrderStatus        = gen.OrderStatus

	// Promo code types
	PromoCodeCreateRequest = gen.PromoCodeCreateRequest
	PromoCodeResponse      = gen.PromoCodeResponse
	DiscountType           = gen.DiscountType

	// User types
	UserRegisterRequest = gen.UserRegisterRequest
	UserLoginRequest    = gen.UserLoginRequest
	UserResponse        = gen.UserResponse
	UserRole            = gen.UserRole

	// Auth types
	AuthResponse        = gen.AuthResponse
	RefreshTokenRequest = gen.RefreshTokenRequest

	// Error types
	ErrorResponse = gen.ErrorResponse
	ErrorCode     = gen.ErrorCode
	OperationType = gen.OperationType
)

// Product status constants
const (
	ProductStatusACTIVE   = gen.PRODUCTSTATUS_ACTIVE
	ProductStatusINACTIVE = gen.PRODUCTSTATUS_INACTIVE
	ProductStatusARCHIVED = gen.PRODUCTSTATUS_ARCHIVED
)

// Order status constants
const (
	OrderStatusCREATED         = gen.ORDERSTATUS_CREATED
	OrderStatusPAYMENT_PENDING = gen.ORDERSTATUS_PAYMENT_PENDING
	OrderStatusPAID            = gen.ORDERSTATUS_PAID
	OrderStatusSHIPPED         = gen.ORDERSTATUS_SHIPPED
	OrderStatusCOMPLETED       = gen.ORDERSTATUS_COMPLETED
	OrderStatusCANCELED        = gen.ORDERSTATUS_CANCELED
)

// User role constants
const (
	UserRoleUSER   = gen.USERROLE_USER
	UserRoleSELLER = gen.USERROLE_SELLER
	UserRoleADMIN  = gen.USERROLE_ADMIN
)

// Error code constants
const (
	ErrorCodePRODUCT_NOT_FOUND         = gen.ERRORCODE_PRODUCT_NOT_FOUND
	ErrorCodePRODUCT_INACTIVE          = gen.ERRORCODE_PRODUCT_INACTIVE
	ErrorCodeORDER_NOT_FOUND           = gen.ERRORCODE_ORDER_NOT_FOUND
	ErrorCodeORDER_LIMIT_EXCEEDED      = gen.ERRORCODE_ORDER_LIMIT_EXCEEDED
	ErrorCodeORDER_HAS_ACTIVE          = gen.ERRORCODE_ORDER_HAS_ACTIVE
	ErrorCodeINVALID_STATE_TRANSITION  = gen.ERRORCODE_INVALID_STATE_TRANSITION
	ErrorCodeINSUFFICIENT_STOCK        = gen.ERRORCODE_INSUFFICIENT_STOCK
	ErrorCodePROMO_CODE_INVALID        = gen.ERRORCODE_PROMO_CODE_INVALID
	ErrorCodePROMO_CODE_MIN_AMOUNT     = gen.ERRORCODE_PROMO_CODE_MIN_AMOUNT
	ErrorCodeORDER_OWNERSHIP_VIOLATION = gen.ERRORCODE_ORDER_OWNERSHIP_VIOLATION
	ErrorCodeVALIDATION_ERROR          = gen.ERRORCODE_VALIDATION_ERROR
	ErrorCodeTOKEN_EXPIRED             = gen.ERRORCODE_TOKEN_EXPIRED
	ErrorCodeTOKEN_INVALID             = gen.ERRORCODE_TOKEN_INVALID
	ErrorCodeREFRESH_TOKEN_INVALID     = gen.ERRORCODE_REFRESH_TOKEN_INVALID
	ErrorCodeACCESS_DENIED             = gen.ERRORCODE_ACCESS_DENIED
)
