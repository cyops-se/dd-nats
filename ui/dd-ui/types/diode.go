package types

import (
	"net"

	"gorm.io/gorm"
)

type DiodeProxy struct {
	gorm.Model
	Name        string      `json:"name"`
	Description string      `json:"description"`
	EndpointIP  string      `json:"ip"`
	EndpointMAC string      `json:"mac"`
	MetaPort    int         `json:"metaport"`
	DataPort    int         `json:"dataport"`
	FilePort    int         `json:"fileport"`
	DataChan    chan []byte `json:"-" gorm:"-"`
	MetaChan    chan []byte `json:"-" gorm:"-"`
	FileChan    chan []byte `json:"-" gorm:"-"`
	DataCon     net.Conn    `json:"-" gorm:"-"`
	MetaCon     net.Conn    `json:"-" gorm:"-"`
	FileCon     net.Conn    `json:"-" gorm:"-"`
}
