package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/bwmarrin/discordgo"
	"github.com/garyburd/redigo/redis"
	"github.com/mitchellh/goamz/s3"
)

type CommandHandler func(*discordgo.MessageCreate, []string) error

var Redis *redis.Pool
var S3Client *s3.S3

func RandomKey(chars int) string {
	if chars%4 != 0 {
		chars = chars + 4 - (chars % 4)
	}

	bytes := make([]byte, (chars/4)*3)

	io.ReadFull(rand.Reader, bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

func makeKey(f string, args ...interface{}) string {
	return fmt.Sprintf("rawr-discordbot.%s", fmt.Sprintf(f, args...))
}

func deserialize(conn redis.Conn, key string, out interface{}) error {
	bytes, err := redis.Bytes(conn.Do("GET", key))
	if err != nil {
		return err
	}

	if bytes == nil {
		return nil
	}

	return json.Unmarshal(bytes, out)
}

func serialize(conn redis.Conn, key string, in interface{}) error {
	bytes, err := json.Marshal(in)
	if err != nil {
		return err
	}

	_, err = conn.Do("SET", key, string(bytes))
	return err
}

func cached(key string, timeout int, out interface{}, gen func() (interface{}, error)) error {
	conn := Redis.Get()
	defer conn.Close()

	bytes, err := redis.Bytes(conn.Do("GET", key))
	if bytes == nil {
		res, err := gen()
		if err != nil {
			return err
		}

		encoded, err := json.Marshal(res)
		if err != nil {
			return err
		}

		if timeout > 0 {
			_, err = conn.Do("SET", key, string(encoded), "EX", timeout)
		} else {
			_, err = conn.Do("SET", key, string(encoded))
		}

		if err != nil {
			return err
		}

		return json.Unmarshal(encoded, out)
	} else {
		if err != nil {
			return err
		}

		return json.Unmarshal(bytes, out)
	}
}
