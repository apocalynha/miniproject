package controller

import (
	"app/config"
	"app/middleware"
	"app/model"
	"app/model/web"
	"app/utils"
	"app/utils/req"
	"app/utils/res"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

func GetAllUser(c echo.Context) error {
	var users []model.User

	err := config.DB.Find(&users).Error
	if err != nil {
		return c.JSON(http.StatusInternalServerError, utils.ErrorResponse("Failed to retrieve user"))
	}

	if len(users) == 0 {
		return c.JSON(http.StatusNotFound, utils.ErrorResponse("Empty data"))
	}

	response := res.ConvertIndex(users)

	return c.JSON(http.StatusOK, utils.SuccessResponse("User data successfully retrieved", response))
}

func GetUserByID(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, utils.ErrorResponse("Invalid ID"))
	}

	var user model.User

	if err := config.DB.First(&user, id).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, utils.ErrorResponse("Failed to retrieve user"))
	}

	response := res.ConvertGeneral(&user)

	return c.JSON(http.StatusOK, utils.SuccessResponse("User data successfully retrieved", response))
}

func Register(c echo.Context) error {
	var user web.UserRequest

	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, utils.ErrorResponse("Invalid request body"))
	}

	ExistingUser := model.User{}
	if err := config.DB.Where("email = ?", user.Email).First(&ExistingUser).Error; err == nil {
		// Email already exists, return an error response
		return c.JSON(http.StatusConflict, utils.ErrorResponse("Email already registered"))
	}

	userDb := req.PassBody(user)

	// Hash the user's password before storing it
	userDb.Password = middleware.HashPassword(userDb.Password)

	if err := config.DB.Create(&userDb).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, utils.ErrorResponse("Failed to store user data"))
	}

	// Return the response without including a JWT token
	response := res.ConvertGeneral(userDb)

	return c.JSON(http.StatusCreated, utils.SuccessResponse("Success Created Data", response))
}

func Login(c echo.Context) error {
	var loginRequest web.LoginRequest

	if err := c.Bind(&loginRequest); err != nil {
		return c.JSON(http.StatusBadRequest, utils.ErrorResponse("Invalid request body"))
	}

	var user model.User
	if err := config.DB.Where("email = ?", loginRequest.Email).First(&user).Error; err != nil {
		return c.JSON(http.StatusUnauthorized, utils.ErrorResponse("Invalid email login credentials"))
	}

	if err := middleware.ComparePassword(user.Password, loginRequest.Password); err != nil {
		return c.JSON(http.StatusUnauthorized, utils.ErrorResponse("Invalid password login credentials"))
	}

	token, err := middleware.CreateToken(int(user.ID), user.Name, user.Role)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, utils.ErrorResponse("Failed to create token"))
	}

	// Buat respons dengan data yang diminta
	response := web.UserLoginResponse{
		Email: user.Email,
		Token: token,
	}

	return c.JSON(http.StatusOK, utils.SuccessResponse("Login successful", response))
}

func UpdateUser(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, utils.ErrorResponse("Invalid ID"))
	}

	var updatedUser model.User

	role := middleware.ExtractTokenUserRole(c)
	userId := middleware.ExtractTokenUserId(c)

	if role != "admin" && userId != id {
		return c.JSON(http.StatusUnauthorized, utils.ErrorResponse("Permission denied"))
	}

	if err := c.Bind(&updatedUser); err != nil {
		return c.JSON(http.StatusBadRequest, utils.ErrorResponse("Invalid request body"))
	}

	var existingUser model.User
	result := config.DB.First(&existingUser, id)
	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, utils.ErrorResponse("Failed to retrieve user"))
	}

	// Check Email
	if updatedUser.Email != existingUser.Email {
		existingUserWithEmail := model.User{}
		if err := config.DB.Where("email = ?", updatedUser.Email).First(&existingUserWithEmail).Error; err == nil {
			// Email already exists, return an error response
			return c.JSON(http.StatusConflict, utils.ErrorResponse("Email already in use"))
		}
	}

	// Check Password
	if updatedUser.Password != "" {
		// Hash new password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(updatedUser.Password), bcrypt.DefaultCost)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, utils.ErrorResponse("Failed to hash the password"))
		}

		updatedUser.Password = string(hashedPassword)
	} else {
		// If the password is not changed, use the existing password in the database
		updatedUser.Password = existingUser.Password
	}

	config.DB.Model(&existingUser).Updates(updatedUser)

	response := res.ConvertGeneral(&existingUser)

	return c.JSON(http.StatusOK, utils.SuccessResponse("User data successfully updated", response))
}

func DeleteUser(c echo.Context) error {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, utils.ErrorResponse("Invalid ID"))
	}

	var existingUser model.User
	result := config.DB.First(&existingUser, id)
	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, utils.ErrorResponse("Failed to retrieve user"))
	}

	role := middleware.ExtractTokenUserRole(c)
	if role != "admin" {
		return c.JSON(http.StatusUnauthorized, utils.ErrorResponse("Permission denied"))
	}

	config.DB.Delete(&existingUser)

	return c.JSON(http.StatusOK, utils.SuccessResponse("User data successfully deleted", nil))
}
