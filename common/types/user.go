package types

type KeyValuePair struct {
	Model
	Key   string `json:"key"`
	Value string `json:"value"`
	Extra string `json:"extra"`
}

type Settings struct {
	Model
	Dark     bool   `json:"dark"`
	ImageURL string `json:"imageurl"`
}

// User
type User struct {
	Model
	UserName   string   `form:"email" json:"email" binding:"required"`
	Password   string   `form:"password" json:"password"`
	FullName   string   `form:"fullname" json:"fullname" binding:"required"`
	Settings   Settings `json:"settings"`
	SettingsID uint     `json:"settingsid"`
}

type UserData struct {
	Model
	UserName   string   `form:"email" json:"email" binding:"required"`
	FullName   string   `form:"fullname" json:"fullname" binding:"required"`
	Settings   Settings `json:"settings"`
	SettingsID uint     `json:"settingsid"`
}

type UserPasswordUpdate struct {
	Model
	Password string `form:"password" json:"password" binding:"required"`
}

type UserCredentials struct {
	Model
	UserName string `form:"email" json:"email" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}
