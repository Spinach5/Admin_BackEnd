package dto

type LoginRequest struct {
	Account  string `json:"account" binding:"required" validate:"required"`
	Password string `json:"password" binding:"required" validate:"required,min=1"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required" validate:"required"`
	NewPassword string `json:"new_password" binding:"required" validate:"required,min=8"`
}

type CreateUserRequest struct {
	Account  string `json:"account" binding:"required" validate:"required,min=2"`
	Password string `json:"password" binding:"required" validate:"required,min=8"`
	IsSuper  int    `json:"is_super" binding:"omitempty" validate:"oneof=0 1"`
}

type UpdateUserRequest struct {
	Account  string `json:"account" binding:"required" validate:"required,min=2"`
	Password string `json:"password" binding:"omitempty" validate:"omitempty,min=8"`
	IsSuper  int    `json:"is_super" binding:"required" validate:"oneof=0 1"`
	IsActive int    `json:"is_active" binding:"required" validate:"oneof=0 1"`
}

type CreateShopRequest struct {
	Name        string  `json:"name" binding:"required" validate:"required"`
	CanteenName string  `json:"canteen_name" binding:"required" validate:"required"`
	Rating      float64 `json:"rating" binding:"required" validate:"gte=0,lte=5"`
	Comment     string  `json:"comment" binding:"required"`
	Min         float64 `json:"min" binding:"required"`
	Max         float64 `json:"max" binding:"required"`
}

type UpdateShopRequest struct {
	ID          int     `json:"id" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	CanteenName string  `json:"canteen_name" binding:"required"`
	Rating      float64 `json:"rating" binding:"required"`
	Comment     string  `json:"comment" binding:"required"`
	Min         float64 `json:"min" binding:"required"`
	Max         float64 `json:"max" binding:"required"`
}

type CreateFoodRequest struct {
	Name        string  `json:"name" binding:"required"`
	ShopName    string  `json:"shop_name" binding:"required"`
	CanteenName string  `json:"canteen_name" binding:"required"`
	Price       float64 `json:"price" binding:"required"`
	Taste       string  `json:"taste" binding:"required"`
	Category    string  `json:"category" binding:"required"`
}

type UpdateFoodRequest struct {
	ID          int     `json:"id" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	ShopName    string  `json:"shop_name" binding:"required"`
	CanteenName string  `json:"canteen_name" binding:"required"`
	Price       float64 `json:"price" binding:"required"`
	Taste       string  `json:"taste" binding:"required"`
	Category    string  `json:"category" binding:"required"`
}

type CreateAffairRequest struct {
	Name     string `json:"name" binding:"required"`
	Category string `json:"category" binding:"required"`
	Link     string `json:"link" binding:"required"`
	Details  string `json:"details" binding:"required"`
	Channel  string `json:"channel" binding:"required"`
}

type UpdateAffairRequest struct {
	ID       int    `json:"id" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Category string `json:"category" binding:"required"`
	Link     string `json:"link" binding:"required"`
	Details  string `json:"details" binding:"required"`
	Channel  string `json:"channel" binding:"required"`
}

type CreateAffairCategoryRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateAffairCategoryRequest struct {
	ID   int    `json:"id" binding:"required"`
	Name string `json:"name" binding:"required"`
}

type ClasstableRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Year     string `json:"year" binding:"required"`
	Semester string `json:"semester" binding:"required"`
}
