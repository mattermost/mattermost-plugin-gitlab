// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"

	"github.com/pkg/errors"
)

type InstanceConfiguration struct {
	GitlabURL               string `json:"gitlaburl"`
	GitlabOAuthClientID     string `json:"gitlaboauthclientid"`
	GitlabOAuthClientSecret string `json:"gitlaboauthclientsecret"`
}

const (
	instanceConfigMapKey      = "Gitlab_Instance_Configuration_Map"
	instanceConfigNameListKey = "Gitlab_Instance_Configuration_Name_List"
)

func (p *Plugin) saveInstanceDetails(instanceName string, config *InstanceConfiguration) error {
	if config == nil {
		return errors.New("config is nil")
	}

	var instanceNameList []string
	err := p.client.KV.Get(instanceConfigNameListKey, &instanceNameList)
	if err != nil {
		return fmt.Errorf("failed to load instance name list: %w", err)
	}

	if containsString(instanceNameList, instanceName) {
		return fmt.Errorf("instance name '%s' already exists", instanceName)
	}

	var instanceConfigMap map[string]InstanceConfiguration
	err = p.client.KV.Get(instanceConfigMapKey, &instanceConfigMap)
	if err != nil {
		return fmt.Errorf("failed to load instance config map: %w", err)
	}

	setAsDefaultInstance := false

	if instanceConfigMap == nil {
		instanceConfigMap = make(map[string]InstanceConfiguration)
		setAsDefaultInstance = true
	}

	instanceConfigMap[instanceName] = *config

	_, err = p.client.KV.Set(instanceConfigMapKey, instanceConfigMap)
	if err != nil {
		return fmt.Errorf("failed to save updated instance config map: %w", err)
	}

	instanceNameList = append(instanceNameList, instanceName)
	_, err = p.client.KV.Set(instanceConfigNameListKey, instanceNameList)
	if err != nil {
		return fmt.Errorf("failed to save updated instance name list: %w", err)
	}

	if setAsDefaultInstance {
		p.setDefaultInstance(instanceName)
	}

	return nil
}

func (p *Plugin) setDefaultInstance(instanceName string) error {
	config := p.getConfiguration()
	config.DefaultInstanceName = instanceName
	config.sanitize()

	configMap, err := config.ToMap()
	if err != nil {
		return err
	}

	err = p.client.Configuration.SavePluginConfig(configMap)
	if err != nil {
		return errors.Wrap(err, "failed to save default instance in plugin config")
	}

	return nil
}

func (p *Plugin) getInstanceDetails(instanceName string) (*InstanceConfiguration, error) {
	var instanceNameList []string
	err := p.client.KV.Get(instanceConfigNameListKey, &instanceNameList)
	if err != nil {
		return nil, fmt.Errorf("failed to load instance name list: %w", err)
	}

	if !containsString(instanceNameList, instanceName) {
		return nil, fmt.Errorf("instance name '%s' does not exist", instanceName)
	}

	var instanceConfigMap map[string]InstanceConfiguration
	err = p.client.KV.Get(instanceConfigMapKey, &instanceConfigMap)
	if err != nil {
		return nil, fmt.Errorf("failed to load instance config map: %w", err)
	}

	config, ok := instanceConfigMap[instanceName]
	if !ok {
		return nil, fmt.Errorf("instance config for '%s' not found", instanceName)
	}

	return &config, nil
}

//nolint:unused
func (p *Plugin) deleteInstanceDetails(instanceName string) error {
	var instanceNameList []string
	err := p.client.KV.Get(instanceConfigNameListKey, &instanceNameList)
	if err != nil {
		return fmt.Errorf("failed to load instance name list: %w", err)
	}

	found := false
	for _, name := range instanceNameList {
		if name == instanceName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("instance name '%s' not found in name list", instanceName)
	}

	var instanceConfigMap map[string]InstanceConfiguration
	err = p.client.KV.Get(instanceConfigMapKey, &instanceConfigMap)
	if err != nil {
		return fmt.Errorf("failed to load instance config map")
	}
	if instanceConfigMap == nil {
		return fmt.Errorf("instance config map is empty")
	}

	if _, ok := instanceConfigMap[instanceName]; !ok {
		return fmt.Errorf("instance config for '%s' does not exist", instanceName)
	}

	delete(instanceConfigMap, instanceName)

	_, err = p.client.KV.Set(instanceConfigMapKey, instanceConfigMap)
	if err != nil {
		return fmt.Errorf("failed to save updated config map")
	}

	instanceNameList = removeStringFromSlice(instanceNameList, instanceName)

	_, err = p.client.KV.Set(instanceConfigNameListKey, instanceNameList)
	if err != nil {
		return fmt.Errorf("failed to save updated instance name list")
	}

	return nil
}
