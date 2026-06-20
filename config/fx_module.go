package config

import (
	"go.uber.org/fx"
)

var Module = fx.Module(
	"config",
	fx.Provide(
		LoadConfig,
		ProvideGRPCConfig,
		ProvideSchedulerConfig,
		ProvideDatabaseConfig,
		ProvideKafkaConfig,
		ProvideBotConfig,
		ToSchedulerConfig,
		ToGRPCConfig,
		ToKafkaConfig,
	),
)
