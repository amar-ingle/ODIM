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

package mgrcommon

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/ODIM-Project/ODIM/lib-utilities/common"
	"github.com/ODIM-Project/ODIM/lib-utilities/config"
	"github.com/ODIM-Project/ODIM/lib-utilities/errors"
	l "github.com/ODIM-Project/ODIM/lib-utilities/logs"
	"github.com/ODIM-Project/ODIM/lib-utilities/response"
	"github.com/ODIM-Project/ODIM/svc-managers/mgrmodel"
)

var (
	GetPluginTokenFunc = GetPluginToken
	GetPluginDataFunc  = mgrmodel.GetPluginData
	StringEqualFold    = strings.EqualFold
	ContactPluginFunc  = ContactPlugin
	JSON_UnmarshalFunc = json.Unmarshal
)

// PluginContactRequest  hold the request of contact plugin
type PluginContactRequest struct {
	Token          string
	OID            string
	DeviceInfo     interface{}
	BasicAuth      map[string]string
	ContactClient  func(context.Context, string, string, string, string, interface{}, map[string]string) (*http.Response, error)
	Plugin         mgrmodel.Plugin
	HTTPMethodType string
}

// ResponseStatus holds the response of Contact Plugin
type ResponseStatus struct {
	StatusCode    int32
	StatusMessage string
}

// ResourceInfoRequest  hold the request of getting  Resource
type ResourceInfoRequest struct {
	URL                   string
	UUID                  string
	SystemID              string
	ContactClient         func(context.Context, string, string, string, string, interface{}, map[string]string) (*http.Response, error)
	DecryptDevicePassword func([]byte) ([]byte, error)
	HTTPMethod            string
	RequestBody           []byte
	BmcUpdatedCreds       *BmcUpdatedCreds
}

type BmcUpdatedCreds struct {
	UserName        string
	UpdatedPassword string
}

// PluginToken interface to hold the token
type PluginToken struct {
	Tokens map[string]string
	lock   sync.Mutex
}

// Token variable hold the all the XAuthToken  against the plguin ID
var Token PluginToken

// DBInterface hold interface for db functions
type DBInterface struct {
	AddManagertoDBInterface func(mgrmodel.RAManager) error
	GenericSave             func([]byte, string, string) error
}

// StoreToken to store the token ioto the  map
func (p *PluginToken) StoreToken(plguinID, token string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.Tokens[plguinID] = token
}

// GetToken to get the token from map
func (p *PluginToken) GetToken(pluginID string) string {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.Tokens[pluginID]
}

// DeviceCommunication to connect with device with all the params
func DeviceCommunication(ctx context.Context, req ResourceInfoRequest) response.RPC {
	var resp response.RPC
	target, gerr := mgrmodel.GetTarget(req.UUID)
	if gerr != nil {
		return common.GeneralError(http.StatusInternalServerError, response.InternalError, gerr.Error(), nil, nil)
	}
	// Get the Plugin info
	plugin, gerr := GetPluginDataFunc(target.PluginID)
	if gerr != nil {
		return common.GeneralError(http.StatusInternalServerError, response.InternalError, gerr.Error(), nil, nil)
	}
	var contactRequest PluginContactRequest
	contactRequest.ContactClient = req.ContactClient
	contactRequest.Plugin = plugin
	if StringEqualFold(plugin.PreferredAuthType, "XAuthToken") {
		token := GetPluginTokenFunc(ctx, contactRequest)
		if token == "" {
			var errorMessage = "error while trying to create session with plugin " + plugin.ID
			return common.GeneralError(http.StatusInternalServerError, response.InternalError, fmt.Sprintf(errorMessage), nil, nil)
		}
		contactRequest.Token = token
	} else {
		contactRequest.BasicAuth = map[string]string{
			"UserName": plugin.Username,
			"Password": string(plugin.Password),
		}
	}
	decryptedPasswordByte, err := req.DecryptDevicePassword(target.Password)
	if err != nil {
		errorMessage := "error while trying to decrypt device password: " + err.Error()
		return common.GeneralError(http.StatusInternalServerError, response.InternalError, fmt.Sprintf(errorMessage), nil, nil)
	}

	contactRequest.DeviceInfo = map[string]interface{}{
		"ManagerAddress": target.ManagerAddress,
		"UserName":       target.UserName,
		"Password":       decryptedPasswordByte,
		"PostBody":       req.RequestBody,
	}
	//replace the uuid:id with the manager id
	contactRequest.OID = strings.Replace(req.URL, req.UUID+"."+req.SystemID, req.SystemID, -1)
	contactRequest.HTTPMethodType = req.HTTPMethod
	//target.PostBody = req.RequestBody
	body, _, getResp, err := ContactPluginFunc(ctx, contactRequest, "error while performing virtual media actions "+contactRequest.OID+": ")
	if err != nil {
		resp.StatusCode = getResp.StatusCode
		json.Unmarshal(body, &resp.Body)
		return resp
	}
	resp.StatusCode = http.StatusOK
	resp.StatusMessage = response.Success
	err = JSON_UnmarshalFunc(body, &resp.Body)
	if err != nil {
		return common.GeneralError(http.StatusInternalServerError, response.InternalError, err.Error(), nil, nil)
	}
	respBody := fmt.Sprintf("%v", resp.Body)
	l.LogWithFields(ctx).Debugf("Outgoing device communication response to northbound: %s", string(respBody))
	return resp
}

// GetResourceInfoFromDevice will contact to the and gets the Particual resource info from device
func GetResourceInfoFromDevice(ctx context.Context, req ResourceInfoRequest) (string, error) {
	target, gerr := mgrmodel.GetTarget(req.UUID)
	if gerr != nil {
		return "", gerr
	}
	// Get the Plugin info
	plugin, gerr := GetPluginDataFunc(target.PluginID)
	if gerr != nil {
		return "", gerr
	}
	var contactRequest PluginContactRequest

	contactRequest.ContactClient = req.ContactClient
	contactRequest.Plugin = plugin

	if strings.EqualFold(plugin.PreferredAuthType, "XAuthToken") {
		token := GetPluginToken(ctx, contactRequest)
		if token == "" {
			var errorMessage = "error while trying to create session with plugin " + plugin.ID
			return "", fmt.Errorf(errorMessage)
		}

		contactRequest.Token = token
	} else {
		contactRequest.BasicAuth = map[string]string{
			"UserName": plugin.Username,
			"Password": string(plugin.Password),
		}

	}
	decryptedPasswordByte, err := req.DecryptDevicePassword(target.Password)
	if err != nil {
		errorMessage := "error while trying to decrypt device password: " + err.Error()
		return "", fmt.Errorf(errorMessage)
	}

	contactRequest.DeviceInfo = map[string]interface{}{
		"ManagerAddress": target.ManagerAddress,
		"UserName":       target.UserName,
		"Password":       decryptedPasswordByte,
	}
	if req.BmcUpdatedCreds != nil {
		if req.BmcUpdatedCreds.UserName == target.UserName && req.BmcUpdatedCreds.UpdatedPassword != string(decryptedPasswordByte) {
			contactRequest.DeviceInfo = map[string]interface{}{
				"ManagerAddress": target.ManagerAddress,
				"UserName":       target.UserName,
				"Password":       []byte(req.BmcUpdatedCreds.UpdatedPassword),
			}
		}

	}

	//replace the uuid:system id with the system to the @odata.id from request url
	contactRequest.OID = strings.Replace(req.URL, req.UUID+"."+req.SystemID, req.SystemID, -1)
	contactRequest.HTTPMethodType = http.MethodGet
	body, _, getResp, err := ContactPlugin(ctx, contactRequest, "error while getting the details "+contactRequest.OID+": ")
	if err != nil {
		if getResp.StatusCode == http.StatusUnauthorized && strings.EqualFold(contactRequest.Plugin.PreferredAuthType, "XAuthToken") {
			if body, _, _, err = RetryManagersOperation(ctx, contactRequest, "error while getting the details "+contactRequest.OID+": "); err != nil {
				return "", fmt.Errorf("error while trying to get data from plugin: %v", err)
			}
		} else {
			return "", fmt.Errorf("error while trying to get data from plugin: %v", err)
		}
	}
	var updatedData = strings.Replace(string(body), "/redfish/v1/Systems/", "/redfish/v1/Systems/"+req.UUID+".", -1)
	updatedData = strings.Replace(updatedData, "/redfish/v1/systems/", "/redfish/v1/systems/"+req.UUID+".", -1)
	// to replace the id in managers
	updatedData = strings.Replace(updatedData, "/redfish/v1/Managers/", "/redfish/v1/Managers/"+req.UUID+".", -1)
	// to replace id in chassis
	updatedData = strings.Replace(updatedData, "/redfish/v1/Chassis/", "/redfish/v1/Chassis/"+req.UUID+".", -1)

	return updatedData, nil
}

// ContactPlugin is commons which handles the request and response of Contact Plugin usage
func ContactPlugin(ctx context.Context, req PluginContactRequest, errorMessage string) ([]byte, string, ResponseStatus, error) {
	var resp ResponseStatus
	var response *http.Response
	var err error
	response, err = callPlugin(ctx, req)
	if err != nil {
		if getPluginStatus(ctx, req.Plugin) {
			response, err = callPlugin(ctx, req)
		}
		if err != nil {
			errorMessage = errorMessage + err.Error()
			resp.StatusCode = http.StatusInternalServerError
			resp.StatusMessage = errors.InternalError
			l.LogWithFields(ctx).Error(errorMessage)
			return nil, "", resp, fmt.Errorf(errorMessage)
		}
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		errorMessage := "error while trying to read response body: " + err.Error()
		resp.StatusCode = http.StatusInternalServerError
		resp.StatusMessage = errors.InternalError
		l.LogWithFields(ctx).Error(errorMessage)
		return nil, "", resp, fmt.Errorf(errorMessage)
	}

	if !(response.StatusCode == http.StatusOK || response.StatusCode == http.StatusCreated) {
		resp.StatusCode = int32(response.StatusCode)
		l.LogWithFields(ctx).Error(errorMessage)
		return body, "", resp, fmt.Errorf(errorMessage)
	}
	data := string(body)
	//replacing the resposne with north bound translation URL
	for key, value := range config.Data.URLTranslation.NorthBoundURL {
		data = strings.Replace(data, key, value, -1)
	}
	return []byte(data), response.Header.Get("X-Auth-Token"), resp, nil
}

// getPluginStatus checks the status of given plugin in configured interval
func getPluginStatus(ctx context.Context, plugin mgrmodel.Plugin) bool {
	var pluginStatus = common.PluginStatus{
		Method: http.MethodGet,
		RequestBody: common.StatusRequest{
			Comment: "",
		},
		ResponseWaitTime:        config.Data.PluginStatusPolling.ResponseTimeoutInSecs,
		Count:                   config.Data.PluginStatusPolling.MaxRetryAttempt,
		RetryInterval:           config.Data.PluginStatusPolling.RetryIntervalInMins,
		PluginIP:                plugin.IP,
		PluginPort:              plugin.Port,
		PluginUsername:          plugin.Username,
		PluginUserPassword:      string(plugin.Password),
		PluginPrefferedAuthType: plugin.PreferredAuthType,
		CACertificate:           &config.Data.KeyCertConf.RootCACertificate,
	}
	status, _, _, err := pluginStatus.CheckStatus()
	if err != nil && !status {
		l.LogWithFields(ctx).Error("Error While getting the status for plugin " + plugin.ID + err.Error())
		return status
	}
	l.LogWithFields(ctx).Error("Status of plugin" + plugin.ID + strconv.FormatBool(status))
	return status
}

func callPlugin(ctx context.Context, req PluginContactRequest) (*http.Response, error) {
	var oid string
	for key, value := range config.Data.URLTranslation.SouthBoundURL {
		oid = strings.Replace(req.OID, key, value, -1)
	}
	var reqURL = "https://" + req.Plugin.IP + ":" + req.Plugin.Port + oid
	if strings.EqualFold(req.Plugin.PreferredAuthType, "BasicAuth") {
		return req.ContactClient(ctx, reqURL, req.HTTPMethodType, "", oid, req.DeviceInfo, req.BasicAuth)
	}
	return req.ContactClient(ctx, reqURL, req.HTTPMethodType, req.Token, oid, req.DeviceInfo, nil)
}

// GetPluginToken will verify the if any token present to the plugin else it will create token for the new plugin
func GetPluginToken(ctx context.Context, req PluginContactRequest) string {
	authToken := Token.GetToken(req.Plugin.ID)
	if authToken == "" {
		return createToken(ctx, req)
	}
	return authToken
}

func createToken(ctx context.Context, req PluginContactRequest) string {
	var contactRequest PluginContactRequest

	contactRequest.ContactClient = req.ContactClient
	contactRequest.Plugin = req.Plugin
	contactRequest.HTTPMethodType = http.MethodPost
	contactRequest.DeviceInfo = map[string]interface{}{
		"Username": req.Plugin.Username,
		"Password": string(req.Plugin.Password),
	}
	contactRequest.OID = "/ODIM/v1/Sessions"
	_, token, _, err := ContactPlugin(ctx, contactRequest, "error while logging in to plugin: ")
	if err != nil {
		l.LogWithFields(ctx).Error(err.Error())
	}
	if token != "" {
		Token.StoreToken(req.Plugin.ID, token)
	}
	return token
}

// RetryManagersOperation will be called whenever  the unauthorized status code during the plugin call
// This function will create a new session token reexcutes the plugin call
func RetryManagersOperation(ctx context.Context, req PluginContactRequest, errorMessage string) ([]byte, string, ResponseStatus, error) {
	var resp response.RPC
	var token = createToken(ctx, req)
	if token == "" {
		var tokenErrorMessage = "error: Unable to create session with plugin " + req.Plugin.ID
		resp = common.GeneralError(http.StatusUnauthorized, response.NoValidSession, tokenErrorMessage,
			[]interface{}{}, nil)
		data, _ := json.Marshal(resp.Body)
		return data, "", ResponseStatus{
			StatusCode: resp.StatusCode,
		}, fmt.Errorf(tokenErrorMessage)
	}
	req.Token = token
	return ContactPlugin(ctx, req, errorMessage)

}

// TrackConfigFileChanges monitors the odim config changes using fsnotfiy
func TrackConfigFileChanges(configFilePath string, dbInterface DBInterface, errChan chan error) {
	eventChan := make(chan interface{})
	format := config.Data.LogFormat
	go common.TrackConfigFileChanges(configFilePath, eventChan, errChan)
	for {
		select {
		case info := <-eventChan: // new data arrives through eventChan channel
			l.Log.Info(info)
			config.TLSConfMutex.RLock()
			mgr := mgrmodel.RAManager{
				Name:            "odimra",
				ManagerType:     "Service",
				FirmwareVersion: config.Data.FirmwareVersion,
				ID:              config.Data.RootServiceUUID,
				UUID:            config.Data.RootServiceUUID,
				State:           "Enabled",
			}
			config.TLSConfMutex.RUnlock()
			err := dbInterface.AddManagertoDBInterface(mgr)
			if err != nil {
				l.Log.Error(err)
			}
			if l.Log.Level != config.Data.LogLevel {
				l.Log.Info("Log level is updated, new log level is ", config.Data.LogLevel)
				l.Log.Logger.SetLevel(config.Data.LogLevel)
			}
			if format != config.Data.LogFormat {
				l.SetFormatter(config.Data.LogFormat)
				format = config.Data.LogFormat
				l.Log.Info("Log format is updated, new log format is ", config.Data.LogFormat)
			}
		case err := <-errChan:
			l.Log.Error(err)
		}
	}
}

// TranslateToSouthBoundURL translates the url to southbound URL
func TranslateToSouthBoundURL(url string) string {
	for key, value := range config.Data.URLTranslation.SouthBoundURL {
		url = strings.Replace(url, key, value, -1)
	}
	return url
}
