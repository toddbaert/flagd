package store

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"sync"

	"github.com/hashicorp/go-memdb"
	"github.com/open-feature/flagd/core/pkg/logger"
	"github.com/open-feature/flagd/core/pkg/model"
)

type del = struct{}

type Payload struct {
	Flags string
}

var deleteMarker *del

type IStore interface {
	GetAll(ctx context.Context, watcher chan Payload, selector string) (map[string]model.Flag, model.Metadata, error)
	Get(ctx context.Context, key string) (model.Flag, model.Metadata, bool)
	GetForFlagSet(ctx context.Context, key string, flagSetId string) (model.Flag, model.Metadata, bool)
	SelectorForFlag(ctx context.Context, flag model.Flag) string
}

type State struct {
	mx                sync.RWMutex
	Flags             map[string]model.Flag `json:"flags"`
	FlagSources       []string
	SourceDetails     map[string]SourceDetails  `json:"sourceMetadata,omitempty"`
	MetadataPerSource map[string]model.Metadata `json:"metadata,omitempty"`
	db                *memdb.MemDB
	logger            *logger.Logger
}

type SourceDetails struct {
	Source   string
	Selector string
}

func (f *State) hasPriority(stored string, new string) bool {
	if stored == new {
		return true
	}
	for i := len(f.FlagSources) - 1; i >= 0; i-- {
		switch f.FlagSources[i] {
		case stored:
			return false
		case new:
			return true
		}
	}
	return true
}

func NewFlags(logger *logger.Logger) *State {

	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"flags": {
				Name: "flags",
				Indexes: map[string]*memdb.IndexSchema{
					"id": {
						Name:   "id",
						Unique: true,
						Indexer: &memdb.CompoundIndex{
							Indexes: []memdb.Indexer{
								&memdb.StringFieldIndex{Field: "Key", Lowercase: false},
								&memdb.StringFieldIndex{Field: "FlagSetId", Lowercase: false},
							},
							AllowMissing: false,
						},
					},
					"flagSetId": {
						Name:   "flagSetId",
						Unique: false,
						Indexer: &memdb.StringFieldIndex{
							Field: "FlagSetId",
						},
					},
					"keyOnly": {
						Name:   "keyOnly",
						Unique: true,
						Indexer: &memdb.CompoundIndex{
							Indexes: []memdb.Indexer{
								&memdb.StringFieldIndex{Field: "Key", Lowercase: false},
								&memdb.ConditionalIndex{
									Conditional: func(flag interface{}) (bool, error) {
										return flag.(model.Flag).FlagSetId == "placeholder", nil
									},
								},
							},
							AllowMissing: false,
						},
					},
				},
			},
		},
	}

	// Create a new data base
	db, err := memdb.NewMemDB(schema)
	if err != nil {
		panic(err)
	}

	return &State{
		Flags:             map[string]model.Flag{},
		SourceDetails:     map[string]SourceDetails{},
		MetadataPerSource: map[string]model.Metadata{},
		db:                db,
		logger:            logger,
	}
}

func (f *State) Set(key string, flag model.Flag) {
	txn := f.db.Txn(true)

	var flagSetId string
	if flag.Metadata["flagSetId"] != nil && flag.Metadata["flagSetId"] != "" {
		flagSetId = flag.Metadata["flagSetId"].(string)
		flag.FlagSetId = flagSetId
	} else {
		flag.FlagSetId = "placeholder"
	}

	flag.Key = key
	txn.Insert("flags", flag)
	txn.Commit()
}

func (f *State) Get(_ context.Context, key string) (model.Flag, model.Metadata, bool) {
	txn := f.db.Txn(false)
	defer txn.Abort()

	raw, err := txn.First("flags", "keyOnly", key, true)
	if err != nil {
		panic(err)
	}

	flag, ok := raw.(model.Flag)
	if !ok {
		return model.Flag{}, model.Metadata{}, false
	}

	return flag, model.Metadata{}, ok
}

func (f *State) GetForFlagSet(_ context.Context, key string, flagSetId string) (model.Flag, model.Metadata, bool) {
	txn := f.db.Txn(false)
	defer txn.Abort()

	raw, err := txn.First("flags", "id", key, flagSetId)
	if err != nil {
		panic(err)
	}

	flag, ok := raw.(model.Flag)
	if !ok {
		return model.Flag{}, model.Metadata{}, false
	}

	return flag, model.Metadata{}, ok
}

func (f *State) SelectorForFlag(_ context.Context, flag model.Flag) string {
	f.mx.RLock()
	defer f.mx.RUnlock()

	return f.SourceDetails[flag.Source].Selector
}

func (f *State) Delete(key string) {
	f.mx.Lock()
	defer f.mx.Unlock()
	delete(f.Flags, key)
}

func (f *State) String() (string, error) {
	f.mx.RLock()
	defer f.mx.RUnlock()
	bytes, err := json.Marshal(f)
	if err != nil {
		return "", fmt.Errorf("unable to marshal flags: %w", err)
	}

	return string(bytes), nil
}

// GetAll returns a copy of the store's state (copy in order to be concurrency safe)
func (f *State) GetAll(ctx context.Context, watcher chan Payload, selector string) (map[string]model.Flag, model.Metadata, error) {
	txn := f.db.Txn(false)
	var it memdb.ResultIterator
	var err error

	if selector == "" {
		it, err = txn.Get("flags", "flagSetId", selector)

	} else {
		it, err = txn.Get("flags", "flagSetId", selector)

	}
	if err != nil {
		panic(err)
	}

	flags := make(map[string]model.Flag)
	for obj := it.Next(); obj != nil; obj = it.Next() {
		flag := obj.(model.Flag)
		flags[flag.Key] = flag
	}

	f.logger.Debug("Get all...")

	if watcher != nil {
		changes := it.WatchCh()

		go func() {
			select {
			case <-changes:
				f.logger.Debug("flags store has changed, notifying watchers")
				a, _, _ := f.GetAll(ctx, watcher, selector)
				b, _ := json.Marshal(a)
				watcher <- Payload{
					Flags: string(b),
				}
				res := it.Next()
				f.logger.Debug(fmt.Sprintf("%v", res))

			}
		}()
	}

	return flags, f.getMetadata(), nil
}

// func thing(it memdb.ResultIterator, watcher chan Payload) {
// 	changes := it.WatchCh()

// 	go func() {
// 		//for {
// 		select {
// 		case <-changes:
// 			//f.logger.Debug("flags store has changed, notifying watchers")
// 			a, _ := f.queryAll()
// 			b, _ := json.Marshal(a)
// 			watcher <- Payload{
// 				Flags: string(b),
// 			}
// 			res := it.Next()
// 			f.logger.Debug(fmt.Sprintf("%v", res))

// 		}
// 		//}
// 	}()
// }

// Add new flags from source.
func (f *State) Add(logger *logger.Logger, source string, selector string, flags map[string]model.Flag,
) map[string]interface{} {
	notifications := map[string]interface{}{}
	txn := f.db.Txn(true)

	for k, newFlag := range flags {
		txn.Insert("flags", newFlag)

		logger.Debug(
			fmt.Sprintf("adding flag %s from source %s with selector %s", k, source, selector),
		)

		notifications[k] = map[string]interface{}{
			"type":   string(model.NotificationCreate),
			"source": source,
		}

		// Store the new version of the flag
		newFlag.Source = source
		newFlag.Selector = selector
		f.Set(k, newFlag)
	}
	txn.Commit()

	return notifications
}

func (f *State) delete(key string) {
	txn := f.db.Txn(true)
	defer txn.Abort()
	txn.DeleteAll("flags", "keyOnly", key, false)

	txn.DeleteAll("flags", "id", key, "2")
	txn.Commit()

}

// Update the flag state with the provided flags.
func (f *State) Update(
	logger *logger.Logger,
	source string,
	selector string,
	flags map[string]model.Flag,
	metadata model.Metadata,
) (map[string]interface{}, bool) {
	notifications := map[string]interface{}{}
	resyncRequired := false
	f.mx.Lock()
	f.setSourceMetadata(source, metadata)

	storedFlags, _, _ := f.GetAll(context.Background(), nil, "2")

	for k, v := range storedFlags {
		if v.Source == source && v.Selector == selector {
			if _, ok := flags[k]; !ok {
				// flag has been deleted
				f.delete(k)
				notifications[k] = map[string]interface{}{
					"type":   string(model.NotificationDelete),
					"source": source,
				}
				resyncRequired = true
				logger.Debug(
					fmt.Sprintf(
						"store resync triggered: flag %s has been deleted from source %s",
						k, source,
					),
				)
				continue
			}
		}
	}
	f.mx.Unlock()
	for k, newFlag := range flags {
		newFlag.Source = source
		newFlag.Selector = selector
		storedFlag, _, ok := f.Get(context.Background(), k)
		if ok {
			// if !f.hasPriority(storedFlag.Source, source) {
			// 	logger.Debug(
			// 		fmt.Sprintf(
			// 			"not merging: flag %s from source %s does not have priority over %s",
			// 			k, source, storedFlag.Source,
			// 		),
			// 	)
			// 	continue
			// }
			if reflect.DeepEqual(storedFlag, newFlag) {
				continue
			}
		}
		if !ok {
			notifications[k] = map[string]interface{}{
				"type":   string(model.NotificationCreate),
				"source": source,
			}
		} else {
			notifications[k] = map[string]interface{}{
				"type":   string(model.NotificationUpdate),
				"source": source,
			}
		}
		// Store the new version of the flag
		f.Set(k, newFlag)
	}
	return notifications, resyncRequired
}

func (f *State) GetMetadataForSource(source string) model.Metadata {
	perSource, ok := f.MetadataPerSource[source]
	if ok && perSource != nil {
		return maps.Clone(perSource)
	}
	return model.Metadata{}
}

func (f *State) getMetadata() model.Metadata {
	metadata := model.Metadata{}
	for _, perSource := range f.MetadataPerSource {
		for key, entry := range perSource {
			_, exists := metadata[key]
			if !exists {
				metadata[key] = entry
			} else {
				metadata[key] = deleteMarker
			}
		}
	}

	// keys that exist across multiple sources are deleted
	maps.DeleteFunc(metadata, func(key string, _ interface{}) bool {
		return metadata[key] == deleteMarker
	})

	return metadata
}

func (f *State) setSourceMetadata(source string, metadata model.Metadata) {
	if f.MetadataPerSource == nil {
		f.MetadataPerSource = map[string]model.Metadata{}
	}

	f.MetadataPerSource[source] = metadata
}
