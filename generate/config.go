package generate

// Config 配置信息
type Config struct {
	ClientNum      int    // 客户端数量
	RelationsLimit int    // 客户端所拥有的关系数量限制
	Weight         int    // 为每个客户端分配权重值的最大范围
	Level          int    // 为每个客户端建立关系时所采用的层级
	MongoURL       string // MongoDB连接字符串
}
