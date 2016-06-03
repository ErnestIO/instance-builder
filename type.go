/* This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/. */

package main

import (
	"bytes"
	"encoding/json"
	"log"

	"gopkg.in/redis.v3"
)

// InstancesCreate : Event to create multiple instances
type InstancesCreate struct {
	Service              string     `json:"service"`
	Status               string     `json:"status"`
	ErrorCode            string     `json:"error_code"`
	ErrorMessage         string     `json:"error_message"`
	Instances            []instance `json:"instances"`
	SequentialProcessing bool       `json:"sequential_processing"`
	Action               string     `json:"action"`
}

func (e *InstancesCreate) toJSON() string {
	message, _ := json.Marshal(e)
	return string(message)
}

func (e *InstancesCreate) cacheKey() string {
	return composeCacheKey(e.Service, e.Action)
}

type instance struct {
	Name               string `json:"name"`
	Type               string `json:"type"`
	CPU                int    `json:"cpus"`
	RAM                int    `json:"ram"`
	IP                 string `json:"ip"`
	Catalog            string `json:"reference_catalog"`
	Image              string `json:"reference_image"`
	Disks              []disk `json:"disks"`
	RouterName         string `json:"router_name"`
	RouterType         string `json:"router_type"`
	RouterIP           string `json:"router_ip"`
	ClientName         string `json:"client_name"`
	DatacenterName     string `json:"datacenter_name"`
	DatacenterPassword string `json:"datacenter_password"`
	DatacenterRegion   string `json:"datacenter_region"`
	DatacenterType     string `json:"datacenter_type"`
	DatacenterUsername string `json:"datacenter_username"`
	NetworkName        string `json:"network_name"`
	VCloudURL          string `json:"vcloud_url"`
	Status             string `json:"status"`
	ErrorCode          string `json:"error_code"`
	ErrorMessage       string `json:"error_message"`
}

type disk struct {
	ID   int `json:"id"`
	Size int `json:"size"`
}

func (r *instance) Valid() (bool, string) {
	if r.Name == "" {
		return false, "Instance name is empty"
	}
	if r.NetworkName == "" {
		return false, "Network name for instances is empty"
	}

	return true, ""
}

func (r *instance) fail() {
	r.Status = "errored"
}

func (r *instance) complete() {
	r.Status = "completed"
}

func (r *instance) processing() {
	r.Status = "processed"
}

func (r *instance) errored() bool {
	return r.Status == "errored"
}

func (r *instance) completed() bool {
	return r.Status == "completed"
}

func (r *instance) isProcessed() bool {
	return r.Status == "processed"
}

func (r *instance) isErrored() bool {
	return r.Status == "errored"
}

func (r *instance) toBeProcessed() bool {
	return r.Status != "processed" && r.Status != "completed" && r.Status != "errored"
}

type instances struct {
	Collection []instance
}

type instanceResource struct {
	CPU     int    `json:"cpus"`
	RAM     int    `json:"ram"`
	IP      string `json:"ip"`
	Catalog string `json:"reference_catalog"`
	Image   string `json:"reference_image"`
	Disks   []disk `json:"disks"`
}

type instanceEvent struct {
	Service            string           `json:"service_id"`
	Type               string           `json:"type"`
	InstanceName       string           `json:"instance_name"`
	InstanceType       string           `json:"instance_type"`
	Resource           instanceResource `json:"instance_resource"`
	RouterName         string           `json:"router_name"`
	RouterType         string           `json:"router_type"`
	RouterIP           string           `json:"router_ip"`
	ClientName         string           `json:"client_name"`
	DatacenterName     string           `json:"datacenter_name"`
	DatacenterPassword string           `json:"datacenter_password"`
	DatacenterRegion   string           `json:"datacenter_region"`
	DatacenterType     string           `json:"datacenter_type"`
	DatacenterUsername string           `json:"datacenter_username"`
	NetworkName        string           `json:"network_name"`
	VCloudURL          string           `json:"vcloud_url"`
}

func (e *instanceEvent) load(i instance, t string, s string) {
	resource := instanceResource{
		CPU:     i.CPU,
		RAM:     i.RAM,
		IP:      i.IP,
		Catalog: i.Catalog,
		Image:   i.Image,
		Disks:   i.Disks,
	}

	e.Service = s
	e.Type = t
	e.InstanceType = i.DatacenterType
	e.InstanceName = i.Name
	e.Resource = resource
	e.RouterType = i.RouterType
	e.RouterIP = i.RouterIP
	e.RouterName = i.RouterName
	e.ClientName = i.ClientName
	e.DatacenterName = i.DatacenterName
	e.DatacenterPassword = i.DatacenterPassword
	e.DatacenterUsername = i.DatacenterUsername
	e.DatacenterRegion = i.DatacenterRegion
	e.DatacenterType = i.DatacenterType
	e.NetworkName = i.NetworkName
	e.VCloudURL = i.VCloudURL
}

func (e *instanceEvent) toJSON() string {
	message, _ := json.Marshal(e)
	return string(message)
}

// Error : Error component
type Error struct {
	Code    json.Number `json:"code,Number"`
	Message string      `json:"message"`
}

type instanceCreatedEvent struct {
	Type         string `json:"type"`
	Service      string `json:"service_id"`
	InstanceID   string `json:"instance_id"`
	InstanceName string `json:"instance_name"`
	Error        Error  `json:"error"`
	Action       string `json:"action"`
}

func (e *instanceCreatedEvent) cacheKey() string {
	return composeCacheKey(e.Service, e.Action)
}

func persistEvent(redisClient *redis.Client, event *InstancesCreate) {
	if err := redisClient.Set(event.cacheKey(), event.toJSON(), 0).Err(); err != nil {
		log.Println(err)
	}
}

func composeCacheKey(service string, action string) string {
	var key bytes.Buffer
	key.WriteString("GPBInstances_")
	key.WriteString(service)
	key.WriteString("_")
	key.WriteString(action)

	return key.String()
}
