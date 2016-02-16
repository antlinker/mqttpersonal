package publish

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	MQTT "github.com/antlinker/mqtt-cli"

	"github.com/antlinker/alog"
	"gopkg.in/mgo.v2"

	"github.com/antlinker/go-cmap"
	"github.com/antlinker/mqttpersonal/config"
)

// Pub 执行Publish操作
func Pub(cfg *Config) {
	pub := &Publish{
		cfg:             cfg,
		lg:              alog.NewALog(),
		clientIndexData: make(map[string]int),
		clients:         cmap.NewConcurrencyMap(),
		execComplete:    make(chan bool, 1),
		end:             make(chan bool, 1),
		startTime:       time.Now(),
	}
	pub.lg.SetLogTag("PUBLISH")
	session, err := mgo.Dial(cfg.MongoUrl)
	if err != nil {
		pub.lg.Errorf("数据库连接发生异常:%s", err.Error())
		return
	}
	pub.session = session
	pub.database = session.DB(config.DataBase)
	err = pub.Init()
	if err != nil {
		pub.lg.Error(err)
		return
	}
	pub.ExecPublish()
	<-pub.end
	pub.lg.Info("执行完成.")
	time.Sleep(time.Second * 2)
	os.Exit(0)
}

type Publish struct {
	cfg             *Config
	lg              *alog.ALog
	session         *mgo.Session
	database        *mgo.Database
	clientData      []config.ClientInfo
	clientIndexData map[string]int
	clients         cmap.ConcurrencyMap
	publishID       int64
	execNum         int64
	execComplete    chan bool
	publishTotalNum int64
	publishNum      int64
	prePublishNum   int64
	maxPublishNum   int64
	arrPublishNum   []int64
	receiveTotalNum int64
	receiveNum      int64
	preReceiveNum   int64
	maxReceiveNum   int64
	arrReceiveNum   []int64
	end             chan bool
	startTime       time.Time
}

func (p *Publish) Init() error {
	err := p.initData()
	if err != nil {
		return err
	}
	err = p.initConnection()
	if err != nil {
		return err
	}
	p.lg.Info("初始化操作完成.")
	calcTicker := time.NewTicker(time.Second * 1)
	go p.calcMaxAndMinPacketNum(calcTicker)
	go p.checkComplete()
	prTicker := time.NewTicker(time.Duration(p.cfg.Interval) * time.Second)
	go p.pubAndRecOutput(prTicker)
	return nil
}

func (p *Publish) initData() error {
	p.lg.Info("开始客户端数据初始化...")
	err := p.database.C(config.C_Client).Find(nil).All(&p.clientData)
	if err != nil {
		return fmt.Errorf("初始化客户端数据发生异常:%s", err.Error())
	}
	for i := 0; i < len(p.clientData); i++ {
		client := p.clientData[i]
		p.clientIndexData[client.ClientID] = i
	}
	p.lg.Info("客户端数据初始化完成.")
	return nil
}

func (p *Publish) initConnection() error {
	p.lg.Info("开始建立MQTT数据连接初始化...")
	var wgroup sync.WaitGroup
	cl := len(p.clientData)
	wgroup.Add(cl)
	for i, l := 0, cl; i < l; i++ {
		go func(clientInfo config.ClientInfo) {
			defer wgroup.Done()
			clientID := clientInfo.ClientID
			opts := MQTT.NewClientOptions()
			opts.AddBroker(fmt.Sprintf("%s://%s", p.cfg.Network, p.cfg.Address))
			opts.SetClientID(clientID)
			opts.SetProtocolVersion(4)
			opts.SetAutoReconnect(p.cfg.AutoReconnect)
			opts.SetStore(MQTT.NewMemoryStore())
			if name, pwd := p.cfg.UserName, p.cfg.Password; name != "" && pwd != "" {
				opts.SetUsername(name)
				opts.SetPassword(pwd)
			}
			if v := p.cfg.KeepAlive; v > 0 {
				opts.SetKeepAlive(time.Duration(v) * time.Second)
			}
			if v := p.cfg.CleanSession; v {
				opts.SetCleanSession(v)
			}
			clientHandle := NewHandleConnect(clientID, p)
			opts.SetConnectionLostHandler(func(cli *MQTT.Client, err error) {
				clientHandle.ErrorHandle(err)
			})
			cli := MQTT.NewClient(opts)
			connectToken := cli.Connect()
			if connectToken.Wait() && connectToken.Error() != nil {
				p.lg.Errorf("客户端[%s]建立连接发生异常:%s", clientID, connectToken.Error().Error())
				return
			}
			topic := "C/" + clientID
			subToken := cli.Subscribe(topic, p.cfg.Qos, func(cli *MQTT.Client, msg MQTT.Message) {
				clientHandle.Subscribe([]byte(msg.Topic()), msg.Payload())
			})
			if subToken.Wait() && subToken.Error() != nil {
				p.lg.Errorf("客户端[%s]订阅主题发生异常:%s", clientID, subToken.Error().Error())
				return
			}
			p.clients.Set(clientID, cli)
		}(p.clientData[i])
	}
	wgroup.Wait()
	p.lg.Info("MQTT数据连接初始化完成.")
	return nil
}

func (p *Publish) checkComplete() {
	<-p.execComplete
	go func() {
		ticker := time.NewTicker(time.Millisecond * 500)
		for range ticker.C {
			if p.receiveNum == p.receiveTotalNum {
				ticker.Stop()
				p.end <- true
			}
		}
	}()
}

func (p *Publish) calcMaxAndMinPacketNum(ticker *time.Ticker) {
	for range ticker.C {
		pubNum := p.publishNum
		recNum := p.receiveNum
		pNum := pubNum - p.prePublishNum
		rNum := recNum - p.preReceiveNum
		if pNum > p.maxPublishNum {
			p.maxPublishNum = pNum
		}
		p.arrPublishNum = append(p.arrPublishNum, pNum)
		if rNum > p.maxReceiveNum {
			p.maxReceiveNum = rNum
		}
		p.arrReceiveNum = append(p.arrReceiveNum, rNum)
		p.prePublishNum = pubNum
		p.preReceiveNum = recNum
	}
}

func (p *Publish) pubAndRecOutput(ticker *time.Ticker) {
	for ct := range ticker.C {
		currentSecond := float64(ct.Sub(p.startTime)) / float64(time.Second)
		var psNum int64
		arrPNum := p.arrPublishNum
		if len(arrPNum) == 0 {
			continue
		}
		for i := 0; i < len(arrPNum); i++ {
			psNum += arrPNum[i]
		}
		avgPNum := psNum / int64(len(arrPNum))
		var rsNum int64
		arrRNum := p.arrReceiveNum
		for i := 0; i < len(arrRNum); i++ {
			rsNum += arrRNum[i]
		}
		avgRNum := rsNum / int64(len(arrRNum))
		clientNum := p.clients.Len()
		output := `
总耗时                      %.2fs
执行次数                    %d
客户端数量                  %d
应发包量                    %d
实际发包量                  %d
每秒平均的发包量            %d
每秒最大的发包量            %d
应收包量                    %d
实际收包量                  %d
每秒平均的接包量            %d
每秒最大的接包量            %d`

		fmt.Printf(output,
			currentSecond, p.execNum, clientNum, p.publishTotalNum, p.publishNum, avgPNum, p.maxPublishNum, p.receiveTotalNum, p.receiveNum, avgRNum, p.maxReceiveNum)
		fmt.Printf("\n")
	}
}

func (p *Publish) ExecPublish() {
	time.AfterFunc(time.Duration(p.cfg.Interval)*time.Second, func() {
		if p.execNum == int64(p.cfg.ExecNum) {
			p.lg.Info("发布已执行完成，等待接收订阅...")
			p.disconnect()
			p.execComplete <- true
			return
		}
		atomic.AddInt64(&p.execNum, 1)
		p.publish()
		p.ExecPublish()
	})
}

func (p *Publish) disconnect() {
	if v := p.cfg.DisconnectScale; v <= 0 || v > 100 || p.cfg.AutoReconnect {
		return
	}
	clientCount := p.clients.Len()
	disCount := int(float32(clientCount) * (float32(p.cfg.DisconnectScale) / 100))
	for _, v := range p.clients.ToMap() {
		cli := v.(*MQTT.Client)
		cli.Disconnect(100)
		disCount--
		if disCount == 0 {
			break
		}
	}
}

func (p *Publish) publish() {
	for i, l := 0, len(p.clientData); i < l; i++ {
		clientInfo := p.clientData[i]
		pNum := len(clientInfo.Relations)
		p.publishTotalNum += int64(pNum)
		p.receiveTotalNum += int64(pNum)
		go p.userPublish(clientInfo)
	}
}

func (p *Publish) userPublish(clientInfo config.ClientInfo) {
	clientID := clientInfo.ClientID
	c, _ := p.clients.Get(clientID)
	cli := c.(*MQTT.Client)
	for j := 0; j < len(clientInfo.Relations); j++ {
		cTime := time.Now()
		sendPacket := config.PacketInfo{
			ID:        fmt.Sprintf("%d_%d", cTime.Unix(), atomic.AddInt64(&p.publishID, 1)),
			SendID:    clientID,
			ReceiveID: clientInfo.Relations[j],
			SendTime:  cTime,
		}
		if p.cfg.IsStore {
			err := p.database.C(config.C_Packet).Insert(sendPacket)
			if err != nil {
				p.lg.Errorf("Publish store error:%s", err.Error())
			}
		}
		bufData, err := json.Marshal(sendPacket)
		if err != nil {
			p.lg.Errorf("Publish json marshal error:%s", err.Error())
			continue
		}
		topic := "C/" + clientInfo.Relations[j]
		cli.Publish(topic, p.cfg.Qos, false, bufData)
		// if publishToken.Wait() && publishToken.Error() != nil {
		// 	p.lg.Errorf("客户端[%s]发布消息出现异常:%s", clientInfo.ClientID, publishToken.Error().Error())
		// 	continue
		// }
		atomic.AddInt64(&p.publishNum, 1)
		// 好友之间的发包间隔
		if v := p.cfg.UserInterval; v > 0 {
			time.Sleep(time.Duration(v) * time.Millisecond)
		}
	}
}
