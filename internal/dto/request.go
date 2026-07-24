package dto

type LoginRequest struct {
	Account  string `json:"account" binding:"required" validate:"required"`
	Password string `json:"password" binding:"required" validate:"required,min=1"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required" validate:"required"`
	NewPassword string `json:"new_password" binding:"required" validate:"required,min=8"`
}

type CreateAdminRequest struct {
	Account  string `json:"account" binding:"required" validate:"required,min=2"`
	Password string `json:"password" binding:"required" validate:"required,min=8"`
	SchoolID string `json:"schoolId"`
	IsSuper  int    `json:"is_super" binding:"omitempty" validate:"oneof=0 1"`
}

type UpdateAdminRequest struct {
	Account  string `json:"account" binding:"required" validate:"required,min=2"`
	Password string `json:"password" binding:"omitempty" validate:"omitempty,min=8"`
	IsSuper  *int   `json:"is_super" binding:"required"`
	IsActive *int   `json:"is_active" binding:"required"`
}

type UpdateAdminInfoRequest struct {
	Account  string `json:"account" binding:"required" validate:"required,min=2"`
	IsSuper  *int   `json:"is_super" binding:"required"`
	IsActive *int   `json:"is_active" binding:"required"`
}

type CreateShopRequest struct {
	Name        string  `json:"name" binding:"required" validate:"required"`
	CanteenName string  `json:"canteen_name" binding:"required" validate:"required"`
	SchoolID    string  `json:"school_id"`
	Rating      float64 `json:"rating" binding:"required" validate:"gte=0,lte=5"`
	Comment     string  `json:"comment" binding:"required"`
	Min         float64 `json:"min" binding:"required"`
	Max         float64 `json:"max" binding:"required"`
}

type UpdateShopRequest struct {
	ID          int     `json:"id" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	CanteenName string  `json:"canteen_name" binding:"required"`
	SchoolID    string  `json:"school_id" binding:"required"`
	Rating      float64 `json:"rating" binding:"required"`
	Comment     string  `json:"comment" binding:"required"`
	Min         float64 `json:"min" binding:"required"`
	Max         float64 `json:"max" binding:"required"`
}

type CreateFoodRequest struct {
	Name        string  `json:"name" binding:"required"`
	ShopName    string  `json:"shop_name" binding:"required"`
	CanteenName string  `json:"canteen_name" binding:"required"`
	SchoolID    string  `json:"school_id"`
	Price       float64 `json:"price" binding:"required"`
	Taste       string  `json:"taste" binding:"required"`
	Category    string  `json:"category" binding:"required"`
}

type UpdateFoodRequest struct {
	ID          int     `json:"id" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	ShopName    string  `json:"shop_name" binding:"required"`
	CanteenName string  `json:"canteen_name" binding:"required"`
	SchoolID    string  `json:"school_id" binding:"required"`
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
	SchoolID string `json:"school_id"`
}

type UpdateAffairRequest struct {
	ID       int    `json:"id" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Category string `json:"category" binding:"required"`
	Link     string `json:"link" binding:"required"`
	Details  string `json:"details" binding:"required"`
	Channel  string `json:"channel" binding:"required"`
	SchoolID string `json:"school_id" binding:"required"`
}

type CreateAffairCategoryRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateAffairCategoryRequest struct {
	ID   int    `json:"id" binding:"required"`
	Name string `json:"name" binding:"required"`
}

type CreateUserRequest struct {
	StuID    string `json:"stuId" binding:"required"`
	NickName string `json:"nickName" binding:"required"`
	SchoolID string `json:"schoolId" binding:"required"`
}

type UpdateUserRequest struct {
	StuID    string `json:"stuId" binding:"required"`
	NickName string `json:"nickName" binding:"required"`
	SchoolID string `json:"schoolId" binding:"required"`
}

// V1 普通用户请求
type V1BaseRequest struct {
	ID       int    `json:"id" binding:"required"`
	StuID    string `json:"stuId" binding:"required"`
	SchoolID string `json:"schoolId" binding:"required"`
}

type V1AddBookRequest struct {
	Title    string  `json:"title" binding:"required"`
	Category *string `json:"category"`
	ImageURL *string `json:"image_url"`
	Price    *string `json:"price"`
	ISBN     *string `json:"isbn"`
	Contact  *string `json:"contact"`
}

type V1DeleteBookRequest struct {
	BookID int `json:"book_id" binding:"required"`
}

type StudentLoginRequest struct {
	StuID    string `json:"stuId" binding:"required"`
	Password string `json:"password" binding:"required"`
	SchoolID string `json:"schoolId" binding:"required"`
}

type StudentRegisterRequest struct {
	StuID    string `json:"stuId" binding:"required"`
	Password string `json:"password" binding:"required"`
	NickName string `json:"nickName" binding:"required"`
	SchoolID string `json:"schoolId" binding:"required"`
}

type CreateClubRequest struct {
	Name         string  `json:"name" binding:"required"`
	Introduction *string `json:"introduction"`
	Activities   *string `json:"activities"`
	Category     *string `json:"category"`
	ImageURL     *string `json:"image_url"`
	SchoolID     string  `json:"schoolId"`
	Nature       int     `json:"nature"`
	Contact      *string `json:"contact"`
	PrincipalID  *int    `json:"principal_id"`
}

type UpdateClubRequest struct {
	ID           int     `json:"id"`
	Name         string  `json:"name" binding:"required"`
	Introduction *string `json:"introduction"`
	Activities   *string `json:"activities"`
	Category     *string `json:"category"`
	ImageURL     *string `json:"image_url"`
	SchoolID     string  `json:"schoolId"`
	Nature       int     `json:"nature"`
	Contact      *string `json:"contact"`
	PrincipalID  *int    `json:"principal_id"`
}

// ============ 教材管理 ============

type CreateMaterialRequest struct {
	ISBN      string  `json:"isbn" binding:"required"`
	Title     string  `json:"title" binding:"required"`
	Author    string  `json:"author"`
	Publisher string  `json:"publisher"`
	Price     float64 `json:"price"`
	ExtraInfo string  `json:"extra_info"`
	// 关联到教材包（可选）
	Semester     string   `json:"semester"`
	AcademicYear string   `json:"academic_year"`
	ClassNames   []string `json:"class_names"`
}

type UpdateMaterialRequest struct {
	ISBN      string  `json:"isbn" binding:"required"`
	Title     string  `json:"title" binding:"required"`
	Author    string  `json:"author"`
	Publisher string  `json:"publisher"`
	Price     float64 `json:"price"`
	ExtraInfo string  `json:"extra_info"`
}
