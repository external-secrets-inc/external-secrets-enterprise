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

// Package tcp implements TCP socket listener.
package tcp

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"time"

	v1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/reloader/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/events"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/schema"
	"github.com/go-logr/logr"
	"github.com/tidwall/gjson"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Socket represents a TCP socket listener. It utilizes a stop channel to manage its lifecycle.
type Socket struct {
	config    *v1alpha1.TCPSocketConfig
	context   context.Context
	cancel    context.CancelFunc
	client    client.Client
	eventChan chan events.SecretRotationEvent
	logger    logr.Logger
	processFn ProcessFn
	listener  net.Listener
}

// SetProcessFn sets the process function for the TCP socket.
func (h *Socket) SetProcessFn(p ProcessFn) {
	h.processFn = p
}

// Start initiates the Socket service, making it ready to accept incoming connections.
func (h *Socket) Start() error {
	if h.config == nil {
		return fmt.Errorf("config is nil")
	}
	addr := fmt.Sprintf("%v:%v", h.config.Host, h.config.Port)
	h.logger.V(1).Info("Starting listener", "address", addr)
	listener, err := net.Listen("tcp", addr)
	h.listener = listener
	if err != nil {
		h.logger.Error(err, "Error starting listener")
		return err
	}
	go h.handleConnection(listener)
	return nil
}

func (h *Socket) handleConnection(listener net.Listener) {
	for {
		if h.context.Err() != nil {
			return
		}
		conn, err := listener.Accept()
		if err != nil {
			h.logger.Error(err, "Error accepting connection")
		}
		go h.readMessage(conn)
	}
}

// ProcessFn is a function type for processing TCP messages.
type ProcessFn func(message []byte)

func (h *Socket) defaultProcess(message []byte) {
	msgString := string(message)
	h.logger.V(1).Info("Processing Message", "Message", msgString)
	if !gjson.Valid(msgString) {
		h.logger.Error(fmt.Errorf("invalid json"), "could not parse json", "Message", msgString)
		return
	}
	res := gjson.Get(msgString, h.config.SecretIdentifierOnPayload)
	if !res.Exists() {
		h.logger.Error(fmt.Errorf("secretIdentifier not found in message"), "error when finding path", "Message", msgString, "Secret Identifier", h.config.SecretIdentifierOnPayload)
		return
	}
	val := res.Value()
	switch v := val.(type) {
	case string:
		event := events.SecretRotationEvent{
			SecretIdentifier:  v,
			RotationTimestamp: time.Now().Format("2006-01-02-15-04-05.000"),
			TriggerSource:     schema.TCPSocket,
		}
		h.eventChan <- event
		h.logger.V(1).Info("Published event to eventChan", "Event", event)
	default:
		h.logger.Error(fmt.Errorf("secretIdentifier must be type string"), "Identifier", v)
	}
}
func (h *Socket) readMessage(conn net.Conn) {
	buf := make([]byte, 4096)
	for {
		if h.context.Err() != nil {
			return
		}
		if conn == nil {
			return
		}
		n, err := conn.Read(buf)
		if err != nil {
			err = conn.Close()
			if err != nil {
				h.logger.Error(err, "Error closing connection")
			}
			return
		}
		if n > 0 {
			messages := bytes.Split(buf[:n], []byte("\n"))
			for _, message := range messages {
				if len(message) > 0 {
					h.logger.V(2).Info("Received message", "Message", message)
					h.processFn(message)
				}
			}
			h.logger.V(2).Info("Raw message", "Message", buf[:n])
		}
	}
}

// Stop stops the TCP socket by closing the stop channel.
func (h *Socket) Stop() error {
	h.cancel()
	return h.listener.Close()
}
