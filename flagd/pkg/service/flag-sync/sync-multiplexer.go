package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/open-feature/flagd/core/pkg/model"
	"github.com/open-feature/flagd/core/pkg/store"
)

//nolint:errchkjson
var emptyConfigBytes, _ = json.Marshal(map[string]map[string]string{
	"flags": {},
})

// Multiplexer abstract subscription handling and storage processing.
// Flag configurations will be lazy loaded using reFill logic upon the calls to publish.
type Multiplexer struct {
	store   *store.State
	sources []string

	subs         map[interface{}]subscription            // subscriptions on all sources
	selectorSubs map[string]map[interface{}]subscription // source specific subscriptions

	allFlags      string            // pre-calculated all flags in store as a string
	selectorFlags map[string]string // pre-calculated selector scoped flags in store as strings

	mu sync.RWMutex
}

type subscription struct {
	id      interface{}
	channel chan store.Payload
}

// NewMux creates a new sync multiplexer
func NewMux(store *store.State, sources []string) (*Multiplexer, error) {
	m := &Multiplexer{
		store:         store,
		sources:       sources,
		subs:          map[interface{}]subscription{},
		selectorSubs:  map[string]map[interface{}]subscription{},
		selectorFlags: map[string]string{},
	}

	return m, m.reFill()
}

// Register a subscription
func (r *Multiplexer) Register(ctx context.Context, source string, con chan store.Payload, selector string) (store.Payload, error) {
	payload, _, _ := r.store.GetAll(ctx, con, selector)
	b, _ := json.Marshal(payload)
	return store.Payload{
		Flags: string(b),
	}, nil
}

// Publish sync updates to subscriptions
func (r *Multiplexer) Publish() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// perform a refill prior to publishing
	err := r.reFill()
	if err != nil {
		return err
	}

	// push to all source subs
	for _, sub := range r.subs {
		sub.channel <- store.Payload{Flags: r.allFlags}
	}

	// push to selector subs
	for source, flags := range r.selectorFlags {
		for _, s := range r.selectorSubs[source] {
			s.channel <- store.Payload{Flags: flags}
		}
	}

	return nil
}

// Unregister a subscription
func (r *Multiplexer) Unregister(id interface{}, selector string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var from map[interface{}]subscription

	if selector == "" {
		from = r.subs
	} else {
		from = r.selectorSubs[selector]
	}

	delete(from, id)
}

// GetAllFlags per specific source
func (r *Multiplexer) GetAllFlags(source string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if source == "" {
		return r.allFlags, nil
	}

	if !slices.Contains(r.sources, source) {
		return "", fmt.Errorf("no flag watcher setup for source %s", source)
	}

	return r.selectorFlags[source], nil
}

// SourcesAsMetadata returns all known sources, comma separated to be used as service metadata
func (r *Multiplexer) SourcesAsMetadata() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return strings.Join(r.sources, ",")
}

// reFill local configuration values
func (r *Multiplexer) reFill() error {
	clear(r.selectorFlags)
	// start all sources with empty config
	for _, source := range r.sources {
		r.selectorFlags[source] = string(emptyConfigBytes)
	}

	all, metadata, err := r.store.GetAll(context.Background(), nil, "")
	if err != nil {
		return fmt.Errorf("error retrieving flags from the store: %w", err)
	}

	bytes, err := json.Marshal(map[string]interface{}{"flags": all, "metadata": metadata})
	if err != nil {
		return fmt.Errorf("error marshalling: %w", err)
	}

	r.allFlags = string(bytes)

	collector := map[string]map[string]model.Flag{}

	for key, flag := range all {
		c, ok := collector[flag.Source]
		if ok {
			c[key] = flag
		} else {
			collector[flag.Source] = map[string]model.Flag{
				key: flag,
			}
		}
	}

	// for all flags, sort them into their correct selector
	for source, flags := range collector {
		// store the corresponding metadata
		metadata := r.store.GetMetadataForSource(source)
		bytes, err := json.Marshal(map[string]interface{}{"flags": flags, "metadata": metadata})
		if err != nil {
			return fmt.Errorf("unable to marshal flags: %w", err)
		}

		r.selectorFlags[source] = string(bytes)
	}

	return nil
}
