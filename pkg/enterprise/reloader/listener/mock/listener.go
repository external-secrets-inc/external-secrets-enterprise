// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

/*
copyright External Secrets Inc. All Rights Reserved.
*/

package mock

import (
	"fmt"
	"sync"
	"time"

	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/events"
)

// MockNotificationListener is a mock implementation of a notification listener for secret rotation events.
type MockNotificationListener struct {
	events       []events.SecretRotationEvent
	emitInterval time.Duration
	mu           sync.Mutex
	stopped      bool
	eventChan    chan events.SecretRotationEvent
}

// Start initiates the emission of events from the MockNotificationListener. Returns an error if the listener has been stopped.
func (m *MockNotificationListener) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopped {
		return fmt.Errorf("listener has been stopped")
	}

	go func() {
		for _, event := range m.events {
			time.Sleep(m.emitInterval)
			m.eventChan <- event
		}
	}()

	return nil
}

// Stop signals the MockNotificationListener to stop emitting events.
func (m *MockNotificationListener) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopped = true
	return nil
}

// NewMockListener creates a new MockNotificationListener with specified events, emit interval, and event channel.
func NewMockListener(events []events.SecretRotationEvent, emitInterval time.Duration, eventChan chan events.SecretRotationEvent) *MockNotificationListener {
	return &MockNotificationListener{
		events:       events,
		emitInterval: emitInterval,
		eventChan:    eventChan,
	}
}
