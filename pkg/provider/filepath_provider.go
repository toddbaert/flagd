package provider

import (
	"encoding/json"
	"errors"
	"io/ioutil"

	log "github.com/sirupsen/logrus"

	"github.com/fsnotify/fsnotify"
	"github.com/open-feature/flagd/pkg/model"
	"github.com/xeipuuv/gojsonschema"
)

type FilePathProvider struct {
	URI string
	Flags model.Flags
}

func (fp *FilePathProvider) watch () {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error(err)
	}
	defer watcher.Close()

	forever := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					flags, err := fp.parse()
					if err != nil {
						// occasionally I get an EOF here, I'm not sure why that is
						log.Error(err)
					}
					fp.Flags = flags;
					log.Printf("Flag values updated.")
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(fp.URI)
	if err != nil {
		log.Fatal(err)
	}
	<-forever
}

func (fp *FilePathProvider) parse () (model.Flags, error) {
	var flags = model.Flags{}
	if fp.URI == "" {
		return flags, errors.New("no filepath string set")
	}
	rawFile, err := ioutil.ReadFile(fp.URI)
	if err != nil {
		return flags, err
	}

	schemaLoader := gojsonschema.NewReferenceLoader("file://./schemas/json-schema/flagd-definitions.json")
	flagFile := gojsonschema.NewBytesLoader(rawFile)
	result, err := gojsonschema.Validate(schemaLoader, flagFile)
	if err != nil {
		return flags, err
	} else if !result.Valid() {
		err := errors.New("Invalid JSON file.")
		log.Error(err)
		return flags, err
	}
	json.Unmarshal(rawFile, &flags)
	return flags, nil
}

func (fp *FilePathProvider) Initialize () error {
	go fp.watch()
	flags, err := fp.parse()
	if err != nil {
		return err
	}
	fp.Flags = flags

	return nil
}

func (fp *FilePathProvider) ResolveBooleanValue (flagKey string, defaultValue bool) (bool, error) {
	var variant = fp.Flags.BooleanFlags[flagKey].DefaultVariant
	return fp.Flags.BooleanFlags[flagKey].Variants[variant], nil
}

func (fp *FilePathProvider) ResolveStringValue (flagKey string, defaultValue string) (string, error) {
	var variant = fp.Flags.StringFlags[flagKey].DefaultVariant
	return fp.Flags.StringFlags[flagKey].Variants[variant], nil
}

func (fp *FilePathProvider) ResolveNumberValue (flagKey string, defaultValue float32) (float32, error) {
	var variant = fp.Flags.NumericFlags[flagKey].DefaultVariant
	return fp.Flags.NumericFlags[flagKey].Variants[variant], nil
}

func (fp *FilePathProvider) ResolveObjectValue (flagKey string, defaultValue map[string]interface{}) (map[string]interface{}, error) {
	var variant = fp.Flags.ObjectFlags[flagKey].DefaultVariant
	return fp.Flags.ObjectFlags[flagKey].Variants[variant], nil
}
