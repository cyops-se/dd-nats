package routes

import (
	"dd-nats/common/db"
	"dd-nats/common/logger"
	"dd-nats/common/types"
	"fmt"
	"net/http"

	"github.com/form3tech-oss/jwt-go"
	"github.com/gofiber/fiber/v2"
)

func RegisterAuthRoutes(api fiber.Router) {
	api.Get("/auth/verify", verifyToken)
	api.Post("/auth/login", login)
	api.Post("/auth/logout", logout)
	api.Post("/auth/register", NewUser)
}

func verifyToken(c *fiber.Ctx) error {
	// c.JSON(http.StatusOK, fiber.Map{"status": "ok"})
	c.Status(http.StatusOK).JSON(fiber.Map{"status": "ok"})
	return nil
}

func login(c *fiber.Ctx) error {
	var data types.User
	if err := c.BodyParser(&data); err != nil {
		logger.Log("error", "login failed (bind)", fmt.Sprintf("%v", err))
		return c.Status(503).SendString(err.Error())
	}

	// log.Println("username:", data.UserName, "password:", data.Password)

	var user types.User
	result := db.DB.Model(&types.User{}).Where("user_name = ? AND password = ?", data.UserName, data.Password).Preload("Settings").First(&user)
	if result.Error != nil {
		logger.Log("error", "login failed", fmt.Sprintf("%v", result.Error))
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	// Create token
	token := jwt.New(jwt.SigningMethodHS256)

	// Set claims
	claims := token.Claims.(jwt.MapClaims)
	claims["name"] = user.FullName
	claims["email"] = user.UserName
	claims["id"] = user.ID
	claims["exp"] = 0 // time.Now().Add(time.Hour * 72).Unix()

	// Generate encoded token and send it as response.
	t, err := token.SignedString([]byte("897puihjöknawerthgfp7<yvalknp98h"))
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.JSON(fiber.Map{"token": t, "user": user})
}

func logout(c *fiber.Ctx) error {
	return nil
}
