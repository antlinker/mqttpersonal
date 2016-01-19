package generate

import (
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"

	"gopkg.in/alog.v1"
	"gopkg.in/mgo.v2"

	"github.com/antlinker/mqttpersonal/config"
)

// Gen 生成客户端数据
func Gen(cfg Config) {
	gen := &Generate{
		cfg:              cfg,
		lg:               alog.NewALog(),
		clientData:       make([]*config.ClientInfo, cfg.ClientNum),
		weightClientData: make(map[int][]string),
	}
	gen.lg.SetLogTag("GENERATE")
	session, err := mgo.Dial(cfg.MongoURL)
	if err != nil {
		gen.lg.Error("连接数据库出现异常：", err)
		return
	}
	gen.session = session
	gen.database = session.DB(config.DataBase)
	gen.InitClientData()
	err = gen.Store()
	if err != nil {
		gen.lg.Error("存储客户端发生错误:", err)
		return
	}
}

// Generate 生成客户端关系
type Generate struct {
	cfg              Config
	lg               *alog.ALog
	session          *mgo.Session
	database         *mgo.Database
	gID              uint64
	clientData       []*config.ClientInfo
	weightClientData map[int][]string
}

func (g *Generate) InitClientData() {
	g.lg.Info("开始初始化客户端数据...")
	for i := 0; i < g.cfg.ClientNum; i++ {
		weight := g.getWeight()
		clientID := fmt.Sprintf("%d_%d", time.Now().Unix(), atomic.AddUint64(&g.gID, 1))
		g.clientData[i] = &config.ClientInfo{
			ClientID: clientID,
			Weight:   weight,
		}
		var ids []string
		vids, ok := g.weightClientData[weight]
		if ok {
			ids = vids
		}
		g.weightClientData[weight] = append(ids, clientID)
	}
	g.lg.Info("客户端数据初始化完成")
}

func (g *Generate) Store() error {
	g.lg.Info("开始存储客户端数据...")
	var maxRelationsNum, minRelationsNum, sumRelationsNum, totalNum int
	for i := 0; i < g.cfg.ClientNum; i++ {
		clientInfo := g.clientData[i]
		for j := 0; j <= g.cfg.Level; j++ {
			w := clientInfo.Weight - j
			clientIDs := g.weightClientData[w]
			if j == 0 {
				for k := 0; k < len(clientIDs); k++ {
					if clientIDs[k] == clientInfo.ClientID {
						clientIDs = append(clientIDs[:k], clientIDs[k+1:]...)
						break
					}
				}
			}
			clientInfo.Relations = append(clientInfo.Relations, clientIDs...)
		}
		l := len(clientInfo.Relations)
		if l == 0 {
			continue
		}
		if l > g.cfg.RelationsLimit {
			clientInfo.Relations = clientInfo.Relations[:g.cfg.RelationsLimit]
		}
		err := g.database.C(config.C_Client).Insert(clientInfo)
		if err != nil {
			return err
		}
		totalNum++
		sumRelationsNum += l
		if minRelationsNum == 0 {
			minRelationsNum = l
		}
		if maxRelationsNum < l {
			maxRelationsNum = l
		}
		if minRelationsNum > l {
			minRelationsNum = l
		}
	}
	g.lg.Infof("客户端数据生成完成，客户端总数:%d,最多好友数量:%d,最少好友数量:%d,平均好友数量:%d", totalNum, maxRelationsNum, minRelationsNum, sumRelationsNum/totalNum)
	return nil
}

func (g *Generate) getWeight() int {
	rd := rand.New(rand.NewSource(time.Now().UnixNano()))
	n := rd.Intn(g.cfg.Weight)
	if n == 0 {
		n = g.getWeight()
	}
	return n
}
