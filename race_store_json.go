package servermanager

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	customRacesDir    = "custom_races"
	championshipsDir  = "championships"
	entrantsFile      = "entrants.json"
	serverOptionsFile = "server_options.json"
)

func NewJSONRaceStore(dir string) RaceStore {
	return &JSONRaceStore{
		base: dir,
	}
}

type JSONRaceStore struct {
	base string

	mutex sync.RWMutex
}

func (rs *JSONRaceStore) listFiles(dir string) ([]string, error) {
	dir = filepath.Join(rs.base, dir)

	files, err := ioutil.ReadDir(dir)

	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	var list []string

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		list = append(list, strings.TrimSuffix(file.Name(), ".json"))
	}

	return list, nil
}

func (rs *JSONRaceStore) encodeFile(filename string, data interface{}) error {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	filename = filepath.Join(rs.base, filename)

	dir := filepath.Dir(filename)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)

		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	f, err := os.Create(filename)

	if err != nil {
		return err
	}

	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")

	return enc.Encode(data)
}

func (rs *JSONRaceStore) decodeFile(filename string, out interface{}) error {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	filename = filepath.Join(rs.base, filename)

	f, err := os.Open(filename)

	if err != nil {
		return err
	}

	defer f.Close()

	enc := json.NewDecoder(f)

	return enc.Decode(out)
}

func (rs *JSONRaceStore) UpsertCustomRace(race *CustomRace) error {
	return rs.encodeFile(filepath.Join(customRacesDir, race.UUID.String()+".json"), race)
}

func (rs *JSONRaceStore) FindCustomRaceByID(uuid string) (*CustomRace, error) {
	var customRace *CustomRace

	err := rs.decodeFile(filepath.Join(customRacesDir, uuid+".json"), &customRace)

	if err != nil {
		return nil, err
	}

	return customRace, nil
}

func (rs *JSONRaceStore) ListCustomRaces() ([]*CustomRace, error) {
	files, err := rs.listFiles(customRacesDir)

	if err != nil {
		return nil, err
	}

	var customRaces []*CustomRace

	for _, file := range files {
		race, err := rs.FindCustomRaceByID(file)

		if err != nil || !race.Deleted.IsZero() {
			continue
		}

		customRaces = append(customRaces, race)
	}

	return customRaces, nil
}

func (rs *JSONRaceStore) DeleteCustomRace(race *CustomRace) error {
	race.Deleted = time.Now()

	return rs.UpsertCustomRace(race)
}

func (rs *JSONRaceStore) UpsertEntrant(entrant Entrant) error {
	entrants, err := rs.ListEntrants()

	if err != nil {
		return err
	}

	entrants = append(entrants, &entrant)

	return rs.encodeFile(entrantsFile, entrants)
}

func (rs *JSONRaceStore) ListEntrants() ([]*Entrant, error) {
	var entrants []*Entrant

	err := rs.decodeFile(entrantsFile, &entrants)

	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return entrants, nil
}

func (rs *JSONRaceStore) UpsertServerOptions(so *GlobalServerConfig) error {
	return rs.encodeFile(serverOptionsFile, so)
}

func (rs *JSONRaceStore) LoadServerOptions() (*GlobalServerConfig, error) {
	var out *GlobalServerConfig

	err := rs.decodeFile(serverOptionsFile, &out)

	if os.IsNotExist(err) {
		return &ConfigIniDefault.GlobalServerConfig, rs.UpsertServerOptions(&ConfigIniDefault.GlobalServerConfig)
	} else if err != nil {
		return nil, err
	}

	return out, err
}

func (rs *JSONRaceStore) UpsertChampionship(c *Championship) error {
	return rs.encodeFile(filepath.Join(championshipsDir, c.ID.String()+".json"), c)
}

func (rs *JSONRaceStore) ListChampionships() ([]*Championship, error) {
	files, err := rs.listFiles(championshipsDir)

	if err != nil {
		return nil, err
	}

	var championships []*Championship

	for _, file := range files {
		c, err := rs.LoadChampionship(file)

		if err != nil || !c.Deleted.IsZero() {
			continue
		}

		championships = append(championships, c)
	}

	return championships, nil
}

func (rs *JSONRaceStore) LoadChampionship(id string) (*Championship, error) {
	var championship *Championship

	err := rs.decodeFile(filepath.Join(championshipsDir, id+".json"), &championship)

	if err != nil {
		return nil, err
	}

	return championship, nil
}

func (rs *JSONRaceStore) DeleteChampionship(id string) error {
	c, err := rs.LoadChampionship(id)

	if err != nil {
		return err
	}

	c.Deleted = time.Now()

	return rs.UpsertChampionship(c)
}
