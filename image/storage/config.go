package storage

import "github.com/autowp/goautowp/image/sampler"

type NamingStrategyConfig struct {
	Strategy string `mapstructure:"strategy"`
	Options  struct {
		Deep int `mapstructure:"deep"`
	} `mapstructure:"options"`
}

type StorageDirConfig struct {
	NamingStrategy NamingStrategyConfig `mapstructure:"naming-strategy"`
	Bucket         string               `mapstructure:"bucket"`
}

type StorageConfig struct {
	Dirs    map[string]StorageDirConfig     `mapstructure:"dirs"`
	Formats map[string]sampler.FormatConfig `mapstructure:"formats"`
}
