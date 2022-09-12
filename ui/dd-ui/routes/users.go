package routes

import (
	"dd-nats/common/db"
	"dd-nats/common/logger"
	"dd-nats/common/types"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func RegisterUserRoutes(api fiber.Router) {
	api.Get("/user", GetAllUsers)
	api.Get("/user/current", GetCurrentUser)
	api.Get("/user/id/:id", GetUserByID)
	api.Get("/user/field/:name/:value", GetUserByField)

	api.Post("/user", NewUser)
	api.Put("/user/:id", UpdateUser)
	api.Patch("/user/:id", UpdateUser)
}

func GetAllUsers(c *fiber.Ctx) error {
	var users []types.UserData
	result := db.DB.Model(&types.User{}).Find(&users)
	if result.Error != nil {
		logger.Log("error", "GetAllUsers failed", fmt.Sprintf("%v", result.Error))
		return c.Status(503).SendString(result.Error.Error())
	}

	c.Status(200)
	return c.JSON(users)
}

func GetCurrentUser(c *fiber.Ctx) error {
	var user types.User
	return c.Status(http.StatusOK).JSON(user)
}

func GetUserByID(c *fiber.Ctx) error {
	// var user types.UserData
	// id := c.Params("id")
	// result := db.DB.Model(&types.User{}).First(&user, "id = ?", id)
	// if result.Error != nil {
	// 	logger.Log("error", "GetUserByID failed", fmt.Sprintf("%v", result.Error))
	// 	// c.JSON(http.StatusNotFound, fiber.Map{"error": result.Error})
	// 	return
	// }

	// c.JSON(http.StatusOK, fiber.Map{"user": user})
	return nil
}

func GetUserByField(c *fiber.Ctx) error {
	// var user types.UserData
	// f := c.Params("name")
	// v := c.Params("value")
	// result := db.DB.Model(&types.User{}).First(&user, "? = ?", f, v)
	// if result.Error != nil {
	// 	logger.Log("error", "GetUserByField failed", fmt.Sprintf("%v", result.Error))
	// 	// c.JSON(http.StatusNotFound, fiber.Map{"error": result.Error})
	// 	return
	// }

	// c.JSON(http.StatusOK, fiber.Map{"user": user})
	return nil
}

func NewUser(c *fiber.Ctx) error {
	var data types.User
	if err := c.BodyParser(&data); err != nil {
		logger.Log("error", "NewUser failed (bind)", fmt.Sprintf("%v", err))
		return c.Status(503).SendString(err.Error())
	}

	user := &types.User{UserName: data.UserName, Password: data.Password, FullName: data.FullName}
	db.DB.Create(&user)

	c.Status(200)
	return c.JSON(user)
}

func UpdateUser(c *fiber.Ctx) error {
	var data types.UserData
	if err := c.BodyParser(&data); err != nil {
		logger.Log("error", "UpdateUser failed (bind)", fmt.Sprintf("%v", err))
		return c.Status(503).SendString(err.Error())
	}

	var user types.User
	if err := db.DB.First(&user, "ID = ?", data.ID).Error; err != nil {
		logger.Log("error", "UpdateUser failed (first)", fmt.Sprintf("%v", err))
		return c.Status(503).SendString(err.Error())
	}

	user.FullName = data.FullName
	user.UserName = data.UserName

	if err := db.DB.Save(&user).Error; err != nil {
		logger.Log("error", "UpdateUser failed (save)", fmt.Sprintf("%v", err))
		return c.Status(503).SendString(err.Error())
	}

	c.Status(200)
	return c.JSON(user)
}
