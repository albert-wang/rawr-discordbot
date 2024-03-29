package config

// package config contains instance specific global settings and objects.

import (
	"encoding/json"
	"io/ioutil"
)

// The port that this application listens on for commands.
var InternalBindPort string

// The authorization token for the bot, obtained during bot user creation.
var BotToken string

// Redis server used as a cache and rate limiter
var RedisServerAddress string

// Access key to get s3 stuff.
var AWSAccessKey string

// Secret key to s3 stuff.
var AWSSecret string

// Secret key for ChatGPT
var CPTKey string

// configData is used to temporarily hold values read from a config file
type configData struct {
	InternalBindPort   string
	BotToken           string
	RedisServerAddress string
	AWSAccessKey       string
	AWSSecret          string
	CPTKey             string
}

// LoadConfigFromFileAndENV creates a new Config object by first reading in
//  configuration seetings from the supplied file path, and then overwriting
//  any values set from environment vartiables.
func LoadConfigFromFileAndENV(path string) error {
	jsonBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	conf := &configData{}
	err = json.Unmarshal(jsonBytes, conf)
	if err != nil {
		return err
	}

	InternalBindPort = conf.InternalBindPort
	BotToken = conf.BotToken
	RedisServerAddress = conf.RedisServerAddress
	AWSAccessKey = conf.AWSAccessKey
	AWSSecret = conf.AWSSecret
	CPTKey = conf.CPTKey
	return nil
}
