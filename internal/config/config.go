package config

import (
	"github.com/ellavs/tg-bot-golang/internal/logger"
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const configFile = "data/config.yaml"

type Config struct {
	Token                       string   `yaml:"token"`                       // Токен бота в телеграме.
	MainCurrency                string   `yaml:"mainCurrency"`                // Основная валюта, в которой хранятся данные.
	CurrenciesName              []string `yaml:"CurrenciesName"`              // Список используемых валют.
	CurrenciesUpdatePeriod      int64    `yaml:"CurrenciesUpdatePeriod"`      // Периодичность обновления курсов валют (в минутах).
	CurrenciesUpdateCachePeriod int64    `yaml:"CurrenciesUpdateCachePeriod"` // Периодичность кэширования курсов валют из базы данных (в минутах).
	ConnectionStringDB          string   `yaml:"ConnectionStringDB"`          // Строка подключения в базе данных.
	KafkaTopic                  string   `yaml:"KafkaTopic"`                  // Наименование топика Kafka.
	BrokersList                 []string `yaml:"BrokersList"`                 // Список адресов брокеров сообщений (адрес Kafka).
}

type Service struct {
	config Config
}

func New() (*Service, error) {
	s := &Service{}

	rawYAML, err := os.ReadFile(configFile)
	if err != nil {
		logger.Error("Ошибка reading config file", "err", err)
		return nil, errors.Wrap(err, "reading config file")
	}

	err = yaml.Unmarshal(rawYAML, &s.config)
	if err != nil {
		logger.Error("Ошибка parsing yaml", "err", err)
		return nil, errors.Wrap(err, "parsing yaml")
	}

	return s, nil
}

func (s *Service) Token() string {
	return s.config.Token
}

func (s *Service) GetConfig() Config {
	return s.config
}
