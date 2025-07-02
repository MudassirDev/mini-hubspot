package handler

type CreateUserRequest struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Password  string `json:"password"`
}

type CreateUserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Plan  string `json:"plan"`
	Role  string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Plan  string `json:"plan"`
	Role  string `json:"role"`
}

type CreateContactRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Company  string `json:"company,omitempty"`
	Position string `json:"position,omitempty"`
	Notes    string `json:"notes,omitempty"`
}
