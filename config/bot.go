package config

type BotConfig struct {
	Token   string `env:"TG_TOKEN" envDefault:""`
	WMSAddr string `env:"WMS_GRPC_ADDR" envDefault:"localhost:50051"`
}

func ProvideBotConfig(cfg *AppConfig) *BotConfig {
	return &cfg.Bot
}
