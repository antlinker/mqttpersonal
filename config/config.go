package config

import "time"

const (
	DataBase = "personal-testing"
	C_Client = "clientinfo"
	C_Packet = "packetinfo"
)

// ClientInfo 客户端信息
type ClientInfo struct {
	ClientID  string   `bson:"clientid"`
	Relations []string `bson:"relations"`
	Weight    int      `bson:"weight"`
}

// PacketInfo 数据包信息
type PacketInfo struct {
	ID          string    `bson:"packetid"`
	SendID      string    `bson:"sendid"`
	ReceiveID   string    `bson:"receiveid"`
	SendTime    time.Time `bson:"sendtime"`
	ReceiveTime time.Time `bson:"receivetime,omitempty"`
}
