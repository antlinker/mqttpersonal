package publish

import (
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/antlinker/go-mqtt/packet"

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
	MQTT.DefaultPublishListen
	MQTT.DefaultDisConnListen
	clientID      string
	pub           *Publish
	recvPacketCnt int64
	sendPacketCnt int64
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
func (hc *HandleConnect) OnUnSubscribeStart(event *MQTT.MqttEvent, filter []string) {
	//hc.pub.lg.Debugf("OnUnSubscribeStart:%v", filter)
}
func (*HandleConnect) OnSubscribeStart(event *MQTT.MqttEvent, sub []MQTT.SubFilter) {
	//Mlog.Debugf("OnSubscribeStart:%v", sub)
}
func (hc *HandleConnect) OnSubscribeSuccess(event *MQTT.MqttEvent, sub []MQTT.SubFilter, result []MQTT.QoS) {

	atomic.AddInt64(&hc.pub.subscribeNum, 1)
	hc.pub.clients.Set(hc.clientID, event.GetClient())
}
func (hc *HandleConnect) OnRecvPacket(event *MQTT.MqttEvent, packet packet.MessagePacket, recvPacketCnt int64) {
	rc := atomic.AddInt64(&hcrecvPacketCnt, 1)
	hc.pub.lg.Debugf("OnRecvPacket:%d", rc)
}
func (hc *HandleConnect) OnSendPacket(event *MQTT.MqttEvent, packet packet.MessagePacket, sendPacketCnt int64) {
	sc := atomic.AddInt64(&hcsendPacketCnt, 1)
	hc.pub.lg.Debugf("OnSendPacket:%d", sc)
}

func (hc *HandleConnect) OnPubReady(event *MQTT.MqttPublishEvent, mp *MQTT.MqttPacket) {
	//Mlog.Debugf("OnPubReady:%v", event.GetSendCnt(PubCnt_TOTAL))
}
func (hc *HandleConnect) OnPubSuccess(event *MQTT.MqttPublishEvent, mp *MQTT.MqttPacket) {
	atomic.AddInt64(&hc.pub.publishNum, 1)
}
func (hc *HandleConnect) OnPubFinal(event *MQTT.MqttPublishEvent, mp *MQTT.MqttPacket) {

}

var hcrecvPacketCnt int64
var hcsendPacketCnt int64
var recvSubCnt int64
