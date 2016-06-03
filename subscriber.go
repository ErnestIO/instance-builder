/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/nats-io/nats"
	"gopkg.in/redis.v3"
)

func updateInstance(n *nats.Conn, instance instance, s string, t string) {
	e := instanceEvent{}
	e.load(instance, t, s)
	n.Publish(t, []byte(e.toJSON()))
}

func processNext(n *nats.Conn, r *redis.Client, subject string, procSubject string, body []byte, status string) (*InstancesCreate, bool) {
	event := &instanceCreatedEvent{}
	json.Unmarshal(body, event)

	parts := strings.Split(subject, ".")
	event.Action = parts[1]
	println(event.cacheKey())
	message, err := r.Get(event.cacheKey()).Result()
	if err != nil {
		log.Println(err)
	}
	stored := &InstancesCreate{}
	err = json.Unmarshal([]byte(message), stored)
	stored.Action = event.Action

	if err != nil {
		log.Println(err)
	}
	completed := true
	scheduled := false
	for i := range stored.Instances {
		if stored.Instances[i].Name == event.InstanceName {
			stored.Instances[i].Status = status
			if stored.Instances[i].errored() == true {
				stored.Instances[i].ErrorCode = string(event.Error.Code)
				stored.Instances[i].ErrorMessage = event.Error.Message
			}
		}
		if stored.Instances[i].completed() == false && stored.Instances[i].errored() == false {
			completed = false
		}
		if stored.Instances[i].toBeProcessed() && scheduled == false {
			scheduled = true
			completed = false
			stored.Instances[i].processing()
			updateInstance(n, stored.Instances[i], event.Service, procSubject)
		}
	}
	persistEvent(r, stored)

	return stored, completed
}

func processResponse(n *nats.Conn, r *redis.Client, s string, res string, p string, t string) {
	n.Subscribe(s, func(m *nats.Msg) {
		stored, completed := processNext(n, r, s, p, m.Data, t)
		if completed {
			complete(n, stored, res)
		}
	})
}

func complete(n *nats.Conn, stored *InstancesCreate, subject string) {
	if isErrored(stored) == true {
		stored.Status = "error"
		stored.ErrorCode = "0002"
		stored.ErrorMessage = "Some instances could not been successfully processed"
		n.Publish(subject+"error", []byte(stored.toJSON()))
	} else {
		stored.Status = "completed"
		stored.ErrorCode = ""
		stored.ErrorMessage = ""
		n.Publish(subject+"done", []byte(stored.toJSON()))
	}
}

func isErrored(stored *InstancesCreate) bool {
	for _, v := range stored.Instances {
		if v.isErrored() {
			return true
		}
	}
	return false
}

func processRequest(n *nats.Conn, r *redis.Client, subject string, resSubject string) {
	n.Subscribe(subject, func(m *nats.Msg) {
		event := InstancesCreate{}
		json.Unmarshal(m.Data, &event)

		parts := strings.Split(subject, ".")
		event.Action = parts[1]

		println(event.cacheKey())
		persistEvent(r, &event)

		if len(event.Instances) == 0 || event.Status == "completed" {
			event.Status = "completed"
			event.ErrorCode = ""
			event.ErrorMessage = ""
			n.Publish(subject+".done", []byte(event.toJSON()))
			return
		}

		for _, instance := range event.Instances {
			if ok, msg := instance.Valid(); ok == false {
				event.Status = "error"
				event.ErrorCode = "0001"
				event.ErrorMessage = msg
				n.Publish(subject+".error", []byte(event.toJSON()))
				return
			}
		}

		sw := false
		for i, instance := range event.Instances {
			if event.Instances[i].completed() == false {
				sw = true
				event.Instances[i].processing()
				updateInstance(n, instance, event.Service, resSubject)
				if true == event.SequentialProcessing {
					break
				}
			}
		}

		if sw == false {
			event.Status = "completed"
			event.ErrorCode = ""
			event.ErrorMessage = ""
			n.Publish(subject+".done", []byte(event.toJSON()))
			return
		}

		persistEvent(r, &event)
	})
}
