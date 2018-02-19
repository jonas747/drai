package drai

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/bwmarrin/discordgo"
	"io/ioutil"
	"os"
	"reflect"
)

var (
	RegisteredApps        = make(map[string]reflect.Type)
	InverseRegisteredApps = make(map[reflect.Type]string)
)

// Registers an app, this is needed for deserialization of the app's state
func RegisterApp(id string, app App) {
	t := reflect.TypeOf(app)
	RegisteredApps[id] = t
	InverseRegisteredApps[t] = id
}

type StorageBackend interface {
	// Saves all application states
	SaveApps(apps []*Instance) error

	// Loads all application states
	LoadApps(engine *Engine, session *discordgo.Session) ([]*Instance, error)
}

type FSStorageBackend struct {
	Path string
}

type SerializedAppState struct {
	AppID     string          `json:"app_id"`
	ChannelID string          `json:"channel_id"`
	GuildID   string          `json:"guild_id"`
	Actions   []*Action       `json:"actions"`
	AppData   json.RawMessage `json:"app_data"`
}

func (f *FSStorageBackend) SaveApps(apps []*Instance) error {
	outFile, err := os.Create(f.Path)
	if err != nil {
		return err
	}

	serializedApps := make([]*SerializedAppState, 0, len(apps))
	for _, v := range apps {
		v.Lock()
		t := reflect.TypeOf(v.App)
		id, ok := InverseRegisteredApps[t]
		if !ok {
			logrus.WithField("app_name", t.Name()).Warn("Unknown app")
			continue
		}

		serialized, err := v.App.SerializeState()
		if err != nil {
			logrus.WithError(err).Error("Failed serializing app")
			continue
		}

		serializedApps = append(serializedApps, &SerializedAppState{
			AppID:     id,
			ChannelID: v.ChannelID,
			GuildID:   v.GuildID,
			Actions:   v.Actions,
			AppData:   serialized,
		})
	}

	allEncoded, err := json.Marshal(serializedApps)
	if err != nil {
		return err
	}

	_, err = outFile.Write(allEncoded)
	outFile.Close()
	return err
}

func (f *FSStorageBackend) LoadApps(engine *Engine, session *discordgo.Session) ([]*Instance, error) {
	data, err := ioutil.ReadFile(f.Path)
	if err != nil {
		return nil, err
	}

	if os.IsNotExist(err) {
		logrus.Info("No apps to load")
		return nil, nil
	}

	var decoded []*SerializedAppState
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		return nil, err
	}

	result := make([]*Instance, 0, len(decoded))

	for _, sas := range decoded {
		t, ok := RegisteredApps[sas.AppID]
		if !ok {
			logrus.WithField("app_id", sas.AppID).Warn("Unknown app id")
			continue
		}

		appDecoded := reflect.New(t).Interface().(App)
		instance := &Instance{
			ChannelID: sas.ChannelID,
			GuildID:   sas.GuildID,
			Actions:   sas.Actions,
			Session:   session,
			Engine:    engine,

			App: appDecoded,
		}

		err = appDecoded.LoadState(instance, sas.AppData)
		if err != nil {
			logrus.Error("Failed loading app state")
			continue
		}

		result = append(result, instance)
	}

	return result, nil
}
