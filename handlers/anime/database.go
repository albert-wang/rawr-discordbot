package anime

import (
	"fmt"
	"log"
	"strings"

	"github.com/albert-wang/rawr-discordbot/config"
	"github.com/albert-wang/rawr-discordbot/storage"

	"github.com/jomei/notionapi"
)

const ANIME_DATABASE = "34e3933e040180609b87e246ab09784a"

type Database struct {
	animes map[string]Status

	// dirty holds the slugs whose Status changed since load. Save writes
	// only these rows back, so a stale snapshot of the rest of the table
	// can never overwrite changes that landed in Notion after we loaded.
	dirty map[string]struct{}
}

// Old version, stored in redis
func LoadDatabaseFromRedis() *Database {
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
		dirty:  map[string]struct{}{},
	}
}

// New, notion backed DB.
// Doesn't support anything but incr/decr
func LoadDatabase() *Database {
	client := notionapi.NewClient(notionapi.Token(config.NotionKey))
	animes, err := loadAnimesFromNotion(client)
	if err != nil {
		log.Fatal(err)
	}

	return &Database{
		animes: animes,
		dirty:  map[string]struct{}{},
	}
}

// put stores s under name and marks the row dirty so Save writes it back.
// Every mutation of animes must go through here.
func (a *Database) put(name string, s Status) {
	a.animes[name] = s
	a.dirty[name] = struct{}{}
}

func (a *Database) search(lookup string) (string, *Status) {
	exact, ok := a.animes[lookup]
	if ok {
		return lookup, &exact
	}

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
	_, ok := a.animes[""]
	if ok {
		delete(a.animes, "")
		delete(a.dirty, "")
	}

	if len(a.dirty) == 0 {
		return
	}

	conn := storage.Redis.Get()
	defer conn.Close()

	key := storage.MakeKey("animestatus")
	err := storage.Serialize(conn, key, &a.animes)
	if err != nil {
		log.Fatal(err)
	}

	for k, v := range a.animes {
		switch {
		case v.CurrentEpisode == 0:
			v.Status = "Not started"
		case v.CurrentEpisode >= 1:
			v.Status = "In progress"
		}
		a.animes[k] = v
	}

	changed := map[string]Status{}
	for k := range a.dirty {
		changed[k] = a.animes[k]
	}

	client := notionapi.NewClient(notionapi.Token(config.NotionKey))
	err = saveAll(client, changed)
	if err != nil {
		log.Fatal(err)
	}
}

func (db *Database) Handle(cmd string, args []string) (string, error) {
	switch cmd {
	case "del":
		return genericHandle(db, args, (*Database).Delete)
	case "set":
		return genericHandle(db, args, (*Database).Set)
	case "list":
		return genericHandle(db, args, (*Database).List)
	case "incr":
		return genericHandle(db, args, (*Database).Incr)
	case "decr":
		return genericHandle(db, args, (*Database).Decr)
	case "inspect":
		return genericHandle(db, args, (*Database).Inspect)
	case "src":
		return genericHandle(db, args, (*Database).Src)
	case "sub":
		return genericHandle(db, args, (*Database).Sub)
	default:
		return "", fmt.Errorf("unknown anime command %s", cmd)
	}
}
