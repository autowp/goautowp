package storage

import (
	"github.com/autowp/goautowp/image/sampler"
)

type NamingStrategyConfig struct {
	Strategy string `mapstructure:"strategy"`
	Options  struct {
		Deep int `mapstructure:"deep"`
	} `mapstructure:"options"`
}

type DirConfig struct {
	NamingStrategy NamingStrategyConfig `mapstructure:"naming-strategy"`
	Bucket         string               `mapstructure:"bucket"`
}

type Config struct {
	Dirs    map[string]DirConfig            `mapstructure:"dirs"`
	Formats map[string]sampler.FormatConfig `mapstructure:"formats"`
	S3      struct {
		Region      string `mapstructure:"region"`
		Endpoint    string `mapstructure:"endpoint"`
		Credentials struct {
			Key    string `mapstructure:"key"`
			Secret string `mapstructure:"secret"`
		} `mapstructure:"credentials"`
		UsePathStyleEndpoint bool `mapstructure:"use_path_style_endpoint"`
	} `mapstructure:"s3"`
}
