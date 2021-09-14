/*
Copyright © 2021 Pete Cornish <outofcoffee@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package debounce

import (
	"sync"
)

type AtMostOnceEvent struct {
	Id  string
	Err error
}

// Debouncer dispatches events with at-most-once semantics for a given ID.
type Debouncer interface {
	// Register records the ID for later debouncing.
	Register(id string)

	// Notify dispatches an event to a channel if the event's ID is registered,
	// otherwise it is dropped.
	// If an event is dispatched, the ID is deregistered to avoid future
	// events being dispatched for the same ID.
	Notify(c chan AtMostOnceEvent, event AtMostOnceEvent)
}

type registrations struct {
	mutex *sync.Mutex
	ids   map[string]bool
}

// Build creates a new Debouncer
func Build() Debouncer {
	return registrations{
		mutex: &sync.Mutex{},
		ids:   make(map[string]bool),
	}
}

func (d registrations) Register(id string) {
	d.mutex.Lock()
	d.ids[id] = true
	d.mutex.Unlock()
}

func (d registrations) Notify(c chan AtMostOnceEvent, event AtMostOnceEvent) {
	if d.ids[event.Id] {
		d.mutex.Lock()
		if d.ids[event.Id] { // double-guard
			delete(d.ids, event.Id)
		}
		d.mutex.Unlock()
		c <- event
	}
}
