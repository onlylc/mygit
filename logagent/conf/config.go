package conf

type AppConf struct {
	KafkaConf `ini:"kafka"`
	// TaillogConf `ini:"taillog"`
	EtcdConf `ini:"etcd"`
}

type KafkaConf struct {
	Address     string `ini:"address"`
	ChanMaxSize int    `ini:"chan_max_size"`
}

type EtcdConf struct {
	Address string `ini:"address"`
	Key     string `ini:"collect_log_key"`
	TimeTou int    `ini:"timeout"`
}

type TaillogConf struct {
	FileName string `ini:"filename"`
}
