package config

import (
	"github.com/spf13/viper"
)

func Create() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	setKeys()
	return nil
}

func setKeys() {
	viper.Set("binance-without-endtime", "https://api.binance.com/api/v3/klines?symbol=%s&interval=%s&limit=1000")
	viper.Set("binance-with-endtime", "https://api.binance.com/api/v3/klines?symbol=%s&interval=%s&endTime=%d&limit=1000")
}
