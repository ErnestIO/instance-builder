/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats"
)

func wait(ch chan bool) error {
	return waitTime(ch, 500*time.Millisecond)
}

func waitTime(ch chan bool, timeout time.Duration) error {
	select {
	case <-ch:
		return nil
	case <-time.After(timeout):
	}
	return errors.New("timeout")
}

func TestInstancesCreateBasic(t *testing.T) {
	os.Setenv("NATS_URI", "nats://localhost:4222")
	os.Setenv("REDIS_ADDR", "localhost:6379")

	n := natsClient()
	r := redisClient()

	processRequest(n, r, "instances.create", "provision-instance")

	ch := make(chan bool)

	n.Subscribe("provision-instance", func(m *nats.Msg) {
		event := &instanceEvent{}
		json.Unmarshal(m.Data, event)
		if event.Type == "provision-instance" {
			log.Println("Message Received")
			if event.InstanceName != "name" {
				t.Fatal("Invalid instance name type")
			}
			if event.Resource.CPU != 1 {
				t.Fatal("Invalid cpu type")
			}
			if event.Resource.RAM != 1024 {
				t.Fatal("Invalid ram type")
			}
			if event.Resource.IP != "ip" {
				t.Fatal("Invalid ip type")
			}
			if event.Resource.Catalog != "reference_catalog" {
				t.Fatal("Invalid catalog type")
			}
			if event.Resource.Image != "reference_image" {
				t.Fatal("Invalid image type")
			}
			if event.NetworkName != "network_name" {
				t.Fatal("Invalid network_name type")
			}
			if event.RouterType != "router_type" {
				t.Fatal("Invalid router_type type")
			}
			if event.RouterIP != "router_ip" {
				t.Fatal("Invalid router_ip type")
			}
			if event.DatacenterName != "datacenter_name" {
				t.Fatal("Invalid datacenter_name type")
			}
			if event.DatacenterUsername != "datacenter_username" {
				t.Fatal("Invalid username type")
			}
			if event.DatacenterPassword != "datacenter_password" {
				t.Fatal("Invalid datacenter_password type")
			}
			if event.DatacenterRegion != "datacenter_region" {
				t.Fatal("Invalid datacenter_region type")
			}
			var key bytes.Buffer
			key.WriteString("GPBInstances_")
			key.WriteString(event.Service)
			key.WriteString("_create")
			message, _ := r.Get(key.String()).Result()
			stored := &InstancesCreate{}
			json.Unmarshal([]byte(message), stored)
			if stored.Service != event.Service {
				t.Fatal("Event is not persisted correctly")
			}
			ch <- true
		} else {
			log.Println(event.Type)
			t.Fatal("Invalid received type")
		}
	})

	message := "{\"service\":\"service\",\"instances\":[{\"name\":\"name\",\"client\":\"client\",\"network\":\"network\",\"cpus\":1,\"ram\":1024,\"ip\":\"ip\",\"reference_catalog\":\"reference_catalog\",\"reference_image\":\"reference_image\",\"disks\":\"disks\",\"router\":\"router_id\",\"network_name\":\"network_name\",\"network_type\":\"network_type\",\"network_netmask\":\"network_netmask\",\"network_start_address\":\"network_start_address\",\"network_end_address\":\"network_end_address\",\"network_gateway\":\"network_gateway\",\"datacenter_id\":\"datacenter_id\",\"router_name\":\"router_name\",\"router_type\":\"router_type\",\"router_ip\":\"router_ip\",\"client_id\":\"client_id\",\"datacenter_name\":\"datacenter_name\",\"datacenter_type\":\"datacenter_type\",\"datacenter_region\":\"datacenter_region\",\"datacenter_username\":\"datacenter_username\",\"datacenter_password\":\"datacenter_password\",\"client_name\":\"client_name\"}]}"

	n.Publish("instances.create", []byte(message))

	time.Sleep(500 * time.Millisecond)

	if e := wait(ch); e != nil {
		t.Fatal("Message not received for subscription")
	}
}

func TestInstancesCreateWithInvalidMessage(t *testing.T) {
	os.Setenv("NATS_URI", "nats://localhost:4222")
	os.Setenv("REDIS_ADDR", "localhost:6379")

	n := natsClient()
	r := redisClient()

	processRequest(n, r, "instances.create", "provision-instance")

	ch := make(chan bool)
	ch2 := make(chan bool)

	n.Subscribe("provision-instance", func(m *nats.Msg) {
		ch <- true
	})

	n.Subscribe("instances.create.error", func(m *nats.Msg) {
		ch2 <- true
	})

	message := "{\"service\":\"service\", \"isntances\":[{\"name\":\"\",\"router\":\"router\",\"network\":\"network\", \"client\": \"client\"}]}"
	n.Publish("instances.create.error", []byte(message))

	if e := wait(ch); e == nil {
		t.Fatal("Produced a provision-instance message when I shouldn't")
	}
	if e := wait(ch2); e != nil {
		t.Fatal("Should produce a instances.create.error message on nats")
	}
}

func TestInstancesCreateWithDifferentMessageType(t *testing.T) {
	os.Setenv("NATS_URI", "nats://localhost:4222")
	os.Setenv("REDIS_ADDR", "localhost:6379")

	n := natsClient()
	r := redisClient()

	processRequest(n, r, "instances.create", "provision-instance")

	ch := make(chan bool)

	n.Subscribe("provision-instance", func(m *nats.Msg) {
		ch <- true
	})

	message := "{\"service\":\"service\", \"firewalls\":[{\"name\":\"\",\"router\":\"router\",\"network\":\"network\", \"client\": \"client\"}]}"
	n.Publish("non-instances-create", []byte(message))

	if e := wait(ch); e == nil {
		t.Fatal("Produced a firewall.provision message when I shouldn't")
	}
}

func TestInstanceCreatedForAMultiRequest(t *testing.T) {
	os.Setenv("NATS_URI", "nats://localhost:4222")
	os.Setenv("REDIS_ADDR", "localhost:6379")

	n := natsClient()
	r := redisClient()

	processResponse(n, r, "instance.create.done", "instances.create.", "provision-instance", "completed")
	ch := make(chan bool)
	service := "sss"

	n.Subscribe("instances.create.done", func(msg *nats.Msg) {
		t.Fatal("Message received does not match")
	})

	original := "{\"service\":\"sss\", \"instances\":[{\"name\":\"name\",\"client\":\"client\",\"network\":\"network\", \"cpus\": 1, \"ram\": 1024, \"ip\":\"ip\",\"reference_catalog\":\"reference_catalog\",\"reference_image\":\"reference_image\",\"disks\":\"disks\"}]}"
	if err := r.Set("GPBInstances_sss_create", original, 0).Err(); err != nil {
		log.Println(err)
		t.Fatal("Can't write on redis")
	}
	message := fmt.Sprintf("{\"type\":\"instance.create.done\",\"service_id\":\"%s\",\"name\":\"name\"}", service)
	n.Publish("instance.create.done", []byte(message))

	if e := wait(ch); e != nil {
		return
	}
}

func TestInstanceCreatedSingle(t *testing.T) {
	os.Setenv("NATS_URI", "nats://localhost:4222")
	os.Setenv("REDIS_ADDR", "localhost:6379")

	n := natsClient()
	r := redisClient()

	processResponse(n, r, "instance.create.done", "instances.create.", "provision-instance", "completed")
	ch := make(chan bool)
	service := "sss"

	n.Subscribe("instances.create.done", func(msg *nats.Msg) {
		event := &InstancesCreate{}
		json.Unmarshal(msg.Data, event)
		log.Println(event)
		log.Println(event.Status)
		log.Println(len(event.Instances))
		if service == event.Service && event.Status == "completed" && len(event.Instances) == 1 {
			ch <- true
		} else {
			t.Fatal("Message received from does not match")
		}
	})

	original := "{\"service\":\"sss\", \"instances\":[{\"name\":\"name\",\"client\":\"client\",\"network\":\"network\", \"cpus\": 1, \"ram\": 1024, \"ip\":\"ip\",\"reference_catalog\":\"reference_catalog\",\"reference_image\":\"reference_image\",\"disks\":\"disks\"}]}"
	if err := r.Set("GPBInstances_sss_create", original, 0).Err(); err != nil {
		log.Println(err)
		t.Fatal("Can't write on redis")
	}
	message := fmt.Sprintf("{\"type\":\"instance.create.done\",\"service_id\":\"%s\",\"instance_name\":\"name\", \"instance_id\":\"2\"}", service)
	n.Publish("instance.create.done", []byte(message))

	if e := wait(ch); e != nil {
		t.Fatal("Message not received from for subscription")
	}
}

func TestProvisionInstancesError(t *testing.T) {
	os.Setenv("NATS_URI", "nats://localhost:4222")
	os.Setenv("REDIS_ADDR", "localhost:6379")

	n := natsClient()
	r := redisClient()

	processResponse(n, r, "instance.update.error", "instances.create.", "instance.update", "errored")
	ch := make(chan bool)
	service := "sss"

	n.Subscribe("instances.create.error", func(msg *nats.Msg) {
		event := &InstancesCreate{}
		json.Unmarshal(msg.Data, event)
		if service == event.Service && event.Status == "error" {
			ch <- true
		} else {
			t.Fatal("Message received from does not match")
		}
	})

	original := "{\"service\":\"sss\",\"instances\":[{\"name\":\"name\",\"client\":\"client\",\"network\":\"network\", \"cpus\": 1, \"ram\": 1024, \"ip\":\"ip\",\"reference_catalog\":\"reference_catalog\",\"reference_image\":\"reference_image\",\"disks\":\"disks\"}]}"
	if err := r.Set("GPBInstances_sss_update", original, 0).Err(); err != nil {
		log.Println(err)
		t.Fatal("Can't write on redis")
	}
	message := fmt.Sprintf("{\"type\":\"instance.update.error\",\"service_id\":\"%s\",\"instance_name\":\"name\"}", service)
	n.Publish("instance.update.error", []byte(message))

	if e := wait(ch); e != nil {
		t.Fatal("Message not received from for subscription")
	}
}
