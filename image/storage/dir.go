package storage

import "fmt"

type Dir struct {
	bucket         string
	namingStrategy NamingStrategy
}

func (d *Dir) Bucket() string {
	return d.bucket
}

func (d *Dir) NamingStrategy() NamingStrategy {
	return d.namingStrategy
}

func NewDir(bucket string, config NamingStrategyConfig) (*Dir, error) {

	var strategy NamingStrategy
	switch config.Strategy {
	case NamingStrategyTypeSerial:
		strategy = NamingStrategySerial{
			deep: config.Options.Deep,
		}
	case NamingStrategyTypePattern:
		strategy = NamingStrategyPattern{}
	}

	if strategy == nil {
		return nil, fmt.Errorf("unknown naming strategy")
	}

	return &Dir{
		bucket:         bucket,
		namingStrategy: strategy,
	}, nil
}
