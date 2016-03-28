package publish

import (
	"encoding/json"
	"sync/atomic"
	"time"

	MQTT "github.com/antlinker/go-mqtt/client"

	"github.com/antlinker/mqttpersonal/config"

	"gopkg.in/mgo.v2/bson"
)

func NewHandleConnect(clientID string, pub *Publish) *HandleConnect {
	return &HandleConnect{
		clientID: clientID,
		pub:      pub,
	}
}

type HandleConnect struct {
	MQTT.DefaultConnListen
	MQTT.DefaultSubscribeListen
	clientID string
	pub      *Publish
}

func (hc *HandleConnect) OnConnStart(event *MQTT.MqttConnEvent) {

}
func (hc *HandleConnect) OnConnSuccess(event *MQTT.MqttConnEvent) {

}
func (hc *HandleConnect) OnConnFailure(event *MQTT.MqttConnEvent, returncode int, err error) {
	hc.pub.lg.Debugf("OnConnFailure(%d):%v", returncode, err)
	if !hc.pub.cfg.AutoReconnect {
		hc.pub.clients.Remove(hc.clientID)
	}
}
func (hc *HandleConnect) OnRecvPublish(event *MQTT.MqttRecvPubEvent, topic string, message []byte, qos MQTT.QoS) {
	atomic.AddInt64(&hc.pub.receiveNum, 1)
	if hc.pub.cfg.IsStore {
		var packetInfo config.PacketInfo
		json.Unmarshal(message, &packetInfo)
		err := hc.pub.database.C(config.C_Packet).Update(bson.M{"packetid": packetInfo.ID}, bson.M{"receivetime": time.Now()})
		if err != nil {
			hc.pub.lg.Errorf("Handle subscribe store error:%s", err.Error())
		}
	}
}

func (hc *HandleConnect) OnSubscribeSuccess(event *MQTT.MqttEvent, sub []MQTT.SubFilter, result []MQTT.QoS) {
	hc.DefaultSubscribeListen.OnSubscribeSuccess(event, sub, result)

}
