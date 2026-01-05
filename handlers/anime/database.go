package anime

import (
	"log"
	"strings"

	"github.com/albert-wang/rawr-discordbot/chat"
	"github.com/albert-wang/rawr-discordbot/storage"
	"github.com/bwmarrin/discordgo"
)

type Database struct {
	animes map[string]Status
}

func LoadDatabase() *Database {
	conn := storage.Redis.Get()
	defer conn.Close()

	key := storage.MakeKey("animestatus")
	animes := map[string]Status{}
	err := storage.Deserialize(conn, key, &animes)
	if err != nil {
		log.Fatal(err)
	}

	return &Database{
		animes: animes,
	}
}

func (a *Database) search(lookup string) (string, *Status) {
	candidates := []Status{}

	key := ""
	for k, v := range a.animes {
		if strings.HasPrefix(k, lookup) {
			candidates = append(candidates, v)
			key = k
		}
	}

	if len(candidates) != 1 {
		return "", nil
	}

	return key, &candidates[0]
}

func (a *Database) Save() {
	conn := storage.Redis.Get()
	defer conn.Close()

	_, ok := a.animes[""]
	if ok {
		delete(a.animes, "")
	}

	key := storage.MakeKey("animestatus")
	err := storage.Serialize(conn, key, &a.animes)
	if err != nil {
		log.Fatal(err)
	}
}

func (db *Database) Handle(cmd string, m *discordgo.MessageCreate, args []string) error {
	switch cmd {
	case "del":
		return genericHandle(db, m, args, (*Database).Delete)
	case "mv":
		return genericHandle(db, m, args, (*Database).Move)
	case "set":
		return genericHandle(db, m, args, (*Database).Set)
	case "list":
		return genericHandle(db, m, args, (*Database).List)
	case "incr":
		return genericHandle(db, m, args, (*Database).Incr)
	case "decr":
		return genericHandle(db, m, args, (*Database).Decr)
	case "inspect":
		return genericHandle(db, m, args, (*Database).Inspect)
	case "src":
		return genericHandle(db, m, args, (*Database).Src)
	case "suggest":
		return genericHandle(db, m, args, (*Database).Suggest)
	default:
		db.Help(m, args)
		return nil
	}
}

func (db *Database) Help(m *discordgo.MessageCreate, args []string) {
	chat.SendPrivateMessageTo(m.Author.ID, "Usage: !anime <del|mv|incr|decr|set|list> <name> [<value>]")
}
