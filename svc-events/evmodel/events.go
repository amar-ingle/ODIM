//(C) Copyright [2020] Hewlett Packard Enterprise Development LP
//
//Licensed under the Apache License, Version 2.0 (the "License"); you may
//not use this file except in compliance with the License. You may obtain
//a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
//WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
//License for the specific language governing permissions and limitations
// under the License.

// Package evmodel have the struct models and DB functionalties
package evmodel

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ODIM-Project/ODIM/lib-utilities/common"
	"github.com/ODIM-Project/ODIM/lib-utilities/errors"
	l "github.com/ODIM-Project/ODIM/lib-utilities/logs"
)

const (
	// EventFormatType is set to Event (MetricReport is not supporting now)
	EventFormatType = "Event"

	// SubscriptionType is set to RedfishEvent (make it as array of SubscritpionType)
	SubscriptionType = "RedfishEvent"

	// Context is set to default if its empty
	Context = "Default"

	// SubscriptionName is set to default name incase if its empty
	SubscriptionName = "Event Subscription"

	// SubscriptionIndex is a index name which required for indexing of event subscriptions
	SubscriptionIndex = common.SubscriptionIndex

	// DeviceSubscriptionIndex is a index name which required for indexing
	// subscription of device
	DeviceSubscriptionIndex = common.DeviceSubscriptionIndex

	// UndeliveredEvents holds table for UndeliveredEvent
	UndeliveredEvents = "UndeliveredEvents"

	// ReadInProgres holds table for ReadInProgres
	ReadInProgres = "ReadInProgres"
	// DeliveryRetryPolicy is set to default value incase if its empty
	DeliveryRetryPolicy = "RetryForever"

	// AggregateSubscriptionIndex is a index name which required for indexing
	// subscription of device
	AggregateSubscriptionIndex = common.AggregateSubscriptionIndex
)

var (
	//GetDbConnection alies for common.GetDBConnection
	GetDbConnection = common.GetDBConnection
)

// OdataIDLink containes link to a resource
type OdataIDLink struct {
	OdataID string `json:"@odata.id"`
}

//RequestBody is required to receive the post request payload
type RequestBody struct {
	Name                 string        `json:"Name"`
	Destination          string        `json:"Destination" validate:"required"`
	EventTypes           []string      `json:"EventTypes,omitempty"`
	MessageIds           []string      `json:"MessageIds,omitempty"`
	ResourceTypes        []string      `json:"ResourceTypes,omitempty"`
	Context              string        `json:"Context"`
	Protocol             string        `json:"Protocol" validate:"required"`
	SubscriptionType     string        `json:"SubscriptionType"`
	EventFormatType      string        `json:"EventFormatType"`
	SubordinateResources bool          `json:"SubordinateResources"`
	OriginResources      []OdataIDLink `json:"OriginResources"`
	DeliveryRetryPolicy  string        `json:"DeliveryRetryPolicy"`
}

//Subscription is a model to store the subscription details
type Subscription struct {
	UserName             string   `json:"UserName"`
	SubscriptionID       string   `json:"SubscriptionID"`
	Destination          string   `json:"Destination"`
	Name                 string   `json:"Name"`
	Context              string   `json:"Context"`
	EventTypes           []string `json:"EventTypes"`
	MessageIds           []string `json:"MessageIds"`
	Protocol             string   `json:"Protocol"`
	SubscriptionType     string   `json:"SubscriptionType"`
	EventFormatType      string   `json:"EventFormatType"`
	SubordinateResources bool     `json:"SubordinateResources"`
	ResourceTypes        []string `json:"ResourceTypes"`
	// To store origin resource
	OriginResource string `json:"OriginResource,omitempty"`
	// To store multiple origin resource
	OriginResources []string `json:"OriginResources"`
	// To store all Device address
	Hosts []string `json:"Hosts"`
	// Remove Location and EventHostIP
	Location                string   `json:"location,omitempty"`
	EventHostIP             string   `json:"EventHostIP,omitempty"`
	ExcludeMessageIds       []string `json:"ExcludeMessageIds,omitempty"`
	ExcludeRegistryPrefixes []string `json:"ExcludeRegistryPrefixes,omitempty"`
	DeliveryRetryPolicy     string   `json:"DeliveryRetryPolicy"`
}

//DeviceSubscription is a model to store the subscription details of a device
type DeviceSubscription common.DeviceSubscription

//EvtSubPost is required to frame the post payload for the target device (South Bound)
type EvtSubPost struct {
	Name                 string        `json:"Name"`
	Destination          string        `json:"Destination"`
	EventTypes           []string      `json:"EventTypes,omitempty"`
	MessageIds           []string      `json:"MessageIds,omitempty"`
	ResourceTypes        []string      `json:"ResourceTypes,omitempty"`
	Protocol             string        `json:"Protocol"`
	EventFormatType      string        `json:"EventFormatType"`
	SubscriptionType     string        `json:"SubscriptionType"`
	SubordinateResources bool          `json:"SubordinateResources"`
	HTTPHeaders          []HTTPHeaders `json:"HttpHeaders"`
	Context              string        `json:"Context"`
	OriginResources      []OdataIDLink `json:"OriginResources"`
	DeliveryRetryPolicy  string        `json:"DeliveryRetryPolicy,omitempty"`
}

//HTTPHeaders required for the subscribing for events
type HTTPHeaders struct {
	ContentType string `json:"Content-Type"`
}

//Target is for sending the request to south bound/plugin
type Target struct {
	ManagerAddress string `json:"ManagerAddress"`
	Password       []byte `json:"Password"`
	UserName       string `json:"UserName"`
	PostBody       []byte `json:"PostBody"`
	DeviceUUID     string `json:"DeviceUUID"`
	PluginID       string `json:"PluginID"`
	Location       string `json:"Location"`
}

// Plugin is the model for plugin information
type Plugin struct {
	IP                string
	Port              string
	Username          string
	Password          []byte
	ID                string
	PluginType        string
	PreferredAuthType string
}

// Fabric is the model for fabrics information
type Fabric struct {
	FabricUUID string
	PluginID   string
}

//Aggregate is the model for Aggregate information
type Aggregate struct {
	Elements []OdataIDLink `json:"Elements"`
}

//GetResource fetches a resource from database using table and key
func GetResource(Table, key string) (string, *errors.Error) {
	conn, err := GetDbConnection(common.InMemory)
	if err != nil {
		return "", errors.PackError(err.ErrNo(), err)
	}
	resourceData, err := conn.Read(Table, key)
	if err != nil {
		return "", errors.PackError(err.ErrNo(), "error while trying to get resource details: ", err.Error())
	}
	var resource string
	if errs := json.Unmarshal([]byte(resourceData), &resource); errs != nil {
		return "", errors.PackError(errors.UndefinedErrorType, errs)
	}
	return resource, nil
}

//GetTarget fetches the System(Target Device Credentials) table details
func GetTarget(deviceUUID string) (*Target, error) {
	var target Target
	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return nil, err
	}

	data, err := conn.Read("System", deviceUUID)
	if err != nil {
		return nil, fmt.Errorf("error while trying to get compute details: %v", err.Error())
	}
	if errs := json.Unmarshal([]byte(data), &target); errs != nil {
		return nil, errs
	}
	return &target, nil

}

//GetPluginData will fetch plugin details
func GetPluginData(pluginID string) (*Plugin, *errors.Error) {
	var plugin Plugin

	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return nil, err
	}

	plugindata, err := conn.Read("Plugin", pluginID)
	if err != nil {
		return nil, errors.PackError(err.ErrNo(), "error while trying to fetch plugin data: ", err.Error())
	}

	if err := json.Unmarshal([]byte(plugindata), &plugin); err != nil {
		return nil, errors.PackError(errors.JSONUnmarshalFailed, err)
	}

	bytepw, errs := common.DecryptWithPrivateKey([]byte(plugin.Password))
	if errs != nil {
		return nil, errors.PackError(errors.DecryptionFailed, "error: "+pluginID+" plugin password decryption failed: "+errs.Error())
	}
	plugin.Password = bytepw

	return &plugin, nil
}

//GetAllPlugins gets all the Plugin from the db
func GetAllPlugins() ([]Plugin, *errors.Error) {
	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return nil, err
	}
	keys, err := conn.GetAllDetails("Plugin")
	if err != nil {
		return nil, err
	}
	var plugins []Plugin
	for _, key := range keys {
		var plugin Plugin
		plugindata, err := conn.Read("Plugin", key)
		if err != nil {
			return nil, errors.PackError(err.ErrNo(), "error while trying to fetch plugin data: ", err.Error())
		}

		if err := json.Unmarshal([]byte(plugindata), &plugin); err != nil {
			return nil, errors.PackError(errors.JSONUnmarshalFailed, err)
		}

		bytepw, errs := common.DecryptWithPrivateKey([]byte(plugin.Password))
		if errs != nil {
			return nil, errors.PackError(errors.DecryptionFailed, "error: "+plugin.ID+" plugin password decryption failed: "+errs.Error())
		}
		plugin.Password = bytepw

		plugins = append(plugins, plugin)

	}
	return plugins, nil
}

//GetAllKeysFromTable return all matching data give table name
func GetAllKeysFromTable(table string) ([]string, error) {
	conn, err := GetDbConnection(common.InMemory)
	if err != nil {
		return nil, err
	}
	keysArray, err := conn.GetAllDetails(table)
	if err != nil {
		return nil, fmt.Errorf("error while trying to get all keys from table - %v: %v", table, err.Error())
	}
	return keysArray, nil
}

//GetAllSystems retrieves all the compute systems in odimra
func GetAllSystems() ([]string, error) {
	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return nil, err
	}
	keysArray, err := conn.GetAllDetails("System")
	if err != nil {
		return nil, fmt.Errorf("error while trying to get all keys from table - System: %v", err)
	}
	return keysArray, nil
}

//GetSingleSystem retrieves specific compute system in odimra based on the ID
func GetSingleSystem(id string) (string, error) {
	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return "", errors.PackError(errors.UndefinedErrorType, err)
	}

	data, rerr := conn.Read("System", id)
	if rerr != nil {
		return "", errors.PackError(rerr.ErrNo(), "error while trying to get compute details: ", rerr.Error())
	}
	return data, nil
}

// GetFabricData  will fetch fabric details
func GetFabricData(fabricID string) (Fabric, error) {
	var fabric Fabric

	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return fabric, err
	}

	fabricdata, err := conn.Read("Fabric", fabricID)
	if err != nil {
		return fabric, fmt.Errorf("error while trying to get user: %v", err.Error())
	}

	if errs := json.Unmarshal([]byte(fabricdata), &fabric); errs != nil {
		return fabric, errs
	}

	return fabric, nil
}

// GetAggregateData  will fetch aggregate details
func GetAggregateData(aggreagetKey string) (Aggregate, error) {
	var aggregate Aggregate
	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return aggregate, err
	}
	aggregatedata, err := conn.Read("Aggregate", aggreagetKey)
	if err != nil {
		return aggregate, fmt.Errorf("error while trying to get user: %v", err.Error())
	}
	if errs := json.Unmarshal([]byte(aggregatedata), &aggregate); errs != nil {
		return aggregate, errs
	}

	return aggregate, nil
}

//GetAllFabrics return all Fabrics
func GetAllFabrics() ([]string, error) {
	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return nil, err
	}
	keysArray, err := conn.GetAllDetails("Fabric")
	if err != nil {
		return nil, fmt.Errorf("error while trying to get all keys from table -Fabric: %v", err.Error())
	}
	for i := 0; i < len(keysArray); i++ {
		keysArray[i] = "/redfish/v1/Fabrics/" + keysArray[i]
	}
	return keysArray, nil
}

// GetDeviceSubscriptions is to get subscription details of device
func GetDeviceSubscriptions(hostIP string) (*DeviceSubscription, error) {

	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return nil, err
	}
	devSubscription, gerr := conn.GetDeviceSubscription(DeviceSubscriptionIndex, hostIP+"*")
	if gerr != nil {
		return nil, fmt.Errorf("error while trying to get subscription of device %v", gerr.Error())
	}
	devSub := strings.Split(devSubscription[0], "||")
	var deviceSubscription = &DeviceSubscription{
		EventHostIP:     devSub[0],
		Location:        devSub[1],
		OriginResources: getSliceFromString(devSub[2]),
	}

	return deviceSubscription, nil
}

// UpdateDeviceSubscriptionLocation is to update subscription details of device
func UpdateDeviceSubscriptionLocation(devSubscription DeviceSubscription) error {
	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return err
	}
	uerr := conn.UpdateDeviceSubscription(DeviceSubscriptionIndex, devSubscription.EventHostIP, devSubscription.Location, devSubscription.OriginResources)
	if uerr != nil {
		return fmt.Errorf("error while trying to update subscription of device %v", uerr.Error())
	}
	return nil
}

// SaveDeviceSubscription is to save subscription details of device
func SaveDeviceSubscription(devSubscription DeviceSubscription) error {
	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return err
	}
	cerr := conn.CreateDeviceSubscriptionIndex(DeviceSubscriptionIndex, devSubscription.EventHostIP, devSubscription.Location, devSubscription.OriginResources)
	if cerr != nil {
		return fmt.Errorf("error while trying to save subscription of device %v", cerr.Error())
	}
	return nil
}

// DeleteDeviceSubscription is to delete subscription details of device
func DeleteDeviceSubscription(hostIP string) error {
	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return err
	}
	derr := conn.DeleteDeviceSubscription(DeviceSubscriptionIndex, hostIP)
	if derr != nil {
		return fmt.Errorf("error while trying to delete subscription of device %v", derr.Error())
	}
	return nil
}

// getSliceFromString is to convert the string to array
func getSliceFromString(sliceString string) []string {
	// EX : array stored in db in string("[alert statuschange]")
	// to convert into an array removing "[" ,"]" and splitting
	slice := strings.Replace(sliceString, "[", "", -1)
	slice = strings.Replace(slice, "]", "", -1)
	if len(slice) < 1 {
		return []string{}
	}
	return strings.Split(slice, " ")
}

// SaveEventSubscription is to save event subscription details
func SaveEventSubscription(evtSubscription Subscription) error {
	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return err
	}
	subscription, merr := json.Marshal(evtSubscription)
	if merr != nil {
		return fmt.Errorf("error while trying marshall event subscriptions %v", merr.Error())
	}
	cerr := conn.CreateEvtSubscriptionIndex(SubscriptionIndex, string(subscription))
	if cerr != nil {
		return fmt.Errorf("error while trying to save event subscriptions %v", cerr.Error())
	}
	return nil
}

// GetEvtSubscriptions is to get event subscription details
func GetEvtSubscriptions(searchKey string) ([]Subscription, error) {
	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return nil, err
	}
	evtSub, gerr := conn.GetEvtSubscriptions(SubscriptionIndex, "*"+searchKey+"*")
	if gerr != nil {
		return nil, fmt.Errorf("error while trying to get subscription of device %v", gerr.Error())
	}
	var eventSubscriptions []Subscription
	for _, value := range evtSub {
		var eventSub Subscription
		if err := json.Unmarshal([]byte(value), &eventSub); err != nil {
			return nil, fmt.Errorf("error while unmarshalling event subscriptions: %v", err.Error())
		}
		eventSubscriptions = append(eventSubscriptions, eventSub)
	}

	return eventSubscriptions, nil
}

// DeleteEvtSubscription is to delete event subscription details
func DeleteEvtSubscription(key string) error {
	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return err
	}
	derr := conn.DeleteEvtSubscriptions(SubscriptionIndex, "*"+key+"*")
	if derr != nil {
		return fmt.Errorf("error while trying to delete subscription of device %v", derr.Error())
	}
	return nil
}

// UpdateEventSubscription is to update event subscription details
func UpdateEventSubscription(evtSubscription Subscription) error {
	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return err
	}
	subscription, merr := json.Marshal(evtSubscription)
	if merr != nil {
		return fmt.Errorf("error while trying marshall event subscriptions %v", merr.Error())
	}
	uerr := conn.UpdateEvtSubscriptions(SubscriptionIndex, "*"+evtSubscription.SubscriptionID+"*", string(subscription))
	if uerr != nil {
		return fmt.Errorf("error while trying to update subscription of device %v", uerr.Error())
	}
	return nil
}

//GetAllMatchingDetails accepts the table name ,pattern and DB type and return all the keys which mathces the pattern
func GetAllMatchingDetails(table, pattern string, dbtype common.DbType) ([]string, *errors.Error) {
	conn, err := GetDbConnection(dbtype)
	if err != nil {
		return []string{}, err
	}
	return conn.GetAllMatchingDetails(table, pattern)
}

// SaveUndeliveredEvents accepts the undelivered event and destination with unique eventid and saves it
func SaveUndeliveredEvents(key string, event []byte) error {
	connPool, err := GetDbConnection(common.OnDisk)
	if err != nil {
		l.Log.Error("While trying to get DB Connection : " + err.Error())
		return fmt.Errorf("error while trying to connecting to DB: %v", err.Error())
	}
	if err = connPool.AddResourceData(UndeliveredEvents, key, string(event)); err != nil {
		l.Log.Error(" while trying to add Undelivered Events to DB: " + err.Error())
		return fmt.Errorf("error while trying to add Undelivered Events to DB: %v", err.Error())
	}
	return nil
}

// GetUndeliveredEvents read the undelivered events for the destination
func GetUndeliveredEvents(destination string) (string, error) {
	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return "", fmt.Errorf("error: while trying to create connection with DB: %v", err.Error())
	}

	eventData, err := conn.Read(UndeliveredEvents, destination)
	if err != nil {
		return "", fmt.Errorf("error: while trying to fetch details: %v", err.Error())
	}

	return eventData, nil
}

// DeleteUndeliveredEvents deletes the undelivered events for the destination
func DeleteUndeliveredEvents(destination string) error {
	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return fmt.Errorf("error: while trying to create connection with DB: %v", err.Error())
	}
	if err := conn.Delete(UndeliveredEvents, destination); err != nil {
		return fmt.Errorf("%v", err.Error())
	}
	return nil
}

// SetUndeliveredEventsFlag will set the flag to maintain one instance already picked up
// the undelivered events for the destination
func SetUndeliveredEventsFlag(destination string) error {
	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return fmt.Errorf("error: while trying to create connection with DB: %v", err.Error())
	}
	if err = conn.AddResourceData(ReadInProgres, destination, "true"); err != nil {
		return fmt.Errorf("error while trying to create new %v resource: %v", ReadInProgres, err.Error())
	}
	_, err = conn.Read(ReadInProgres, destination)
	if err != nil {
		l.Log.Error(err)
	}
	return nil
}

// GetUndeliveredEventsFlag will get the flag to maintain one instance already picked up
// the undelivered events for the destination
func GetUndeliveredEventsFlag(destination string) (bool, error) {
	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return false, fmt.Errorf("error: while trying to create connection with DB: %v", err.Error())
	}
	_, err = conn.Read(ReadInProgres, destination)
	if err != nil {
		return false, err
	}
	return true, nil
}

// DeleteUndeliveredEventsFlag deletes the PickUpUndeliveredEventsFlag key from the DB, return error if any
func DeleteUndeliveredEventsFlag(destination string) error {
	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return fmt.Errorf("error: while trying to create connection with DB: %v", err.Error())
	}
	if err := conn.Delete(ReadInProgres, destination); err != nil {
		return fmt.Errorf("%v", err.Error())
	}
	return nil
}

// SaveAggregateSubscription is to save subscription details of device
func SaveAggregateSubscription(aggregateID string, hostIP []string) error {
	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return err
	}
	cerr := conn.CreateAggregateHostIndex(AggregateSubscriptionIndex, aggregateID, hostIP)
	if cerr != nil {
		return fmt.Errorf("error while trying to save subscription of device %v", cerr.Error())
	}
	return nil
}

// UpdateAggregateHosts is to update aggregate hosts details of device
func UpdateAggregateHosts(aggregateID string, hostIP []string) error {
	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return err
	}
	cerr := conn.UpdateAggregateHosts(AggregateSubscriptionIndex, aggregateID, hostIP)
	if cerr != nil {
		return fmt.Errorf("error while trying to save subscription of device %v", cerr.Error())
	}
	return nil
}

// GetAggregateHosts is to get subscription details of device
func GetAggregateHosts(aggregateID string) ([]string, error) {

	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return nil, err
	}
	aggregateList, gerr := conn.GetAggregateHosts(AggregateSubscriptionIndex, aggregateID+"[^0-9]*")
	if gerr != nil {
		return nil, fmt.Errorf("error while trying to get aggregate host of device %v", gerr.Error())
	}
	devSub := strings.Split(aggregateList[0], "||")
	hostsIP := getSliceFromString(devSub[1])
	return hostsIP, nil
}

// GetAggregateList  will fetch aggregate list
func GetAggregateList(hostIP string) ([]string, error) {
	conn, err := GetDbConnection(common.OnDisk)
	if err != nil {
		return nil, err
	}
	aggregateList, gerr := conn.GetAggregateHosts(AggregateSubscriptionIndex, "*"+hostIP+"*")
	if gerr != nil {
		return nil, fmt.Errorf("error while trying to get aggregate host list of device %v", gerr.Error())
	}
	aggregates := []string{}
	for _, v := range aggregateList {
		devSub := strings.Split(v, "||")
		if devSub[0] == "0" {
			continue
		}
		aggregates = append(aggregates, devSub[0])
	}
	return aggregates, nil
}
