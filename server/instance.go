// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"slices"

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

func (p *Plugin) installInstance(instanceName string, config *InstanceConfiguration) error {
	if config == nil {
		return errors.New("config is nil")
	}

	var instanceNameList []string
	err := p.client.KV.Get(instanceConfigNameListKey, &instanceNameList)
	if err != nil {
		p.client.Log.Error("Failed to load instance name list while installing instance", "error", err)
		return fmt.Errorf("failed to load instance name list")
	}

	if slices.Contains(instanceNameList, instanceName) {
		return fmt.Errorf("instance name '%s' already exists", instanceName)
	}

	var instanceConfigMap map[string]InstanceConfiguration
	err = p.client.KV.Get(instanceConfigMapKey, &instanceConfigMap)
	if err != nil {
		p.client.Log.Error("Failed to load instance config map while installing instance", "error", err)
		return fmt.Errorf("failed to load instance config map")
	}

	setAsDefaultInstance := false

	if instanceConfigMap == nil {
		instanceConfigMap = make(map[string]InstanceConfiguration)
		setAsDefaultInstance = true
	}

	instanceConfigMap[instanceName] = *config

	_, err = p.client.KV.Set(instanceConfigMapKey, instanceConfigMap)
	if err != nil {
		p.client.Log.Error("Failed to save updated instance config map while installing instance", "error", err)
		return fmt.Errorf("failed to save updated instance config map")
	}

	instanceNameList = append(instanceNameList, instanceName)
	_, err = p.client.KV.Set(instanceConfigNameListKey, instanceNameList)
	if err != nil {
		p.client.Log.Error("Failed to save updated instance name list while installing instance", "error", err)
		return fmt.Errorf("failed to save updated instance name list")
	}

	if setAsDefaultInstance {
		if err := p.setDefaultInstance(instanceName); err != nil {
			return fmt.Errorf("failed to set default instance: %w", err)
		}
	}

	return nil
}

func (p *Plugin) getInstance(instanceName string) (*InstanceConfiguration, error) {
	var instanceNameList []string
	err := p.client.KV.Get(instanceConfigNameListKey, &instanceNameList)
	if err != nil {
		p.client.Log.Error("Failed to load instance name list while getting instance", "error", err)
		return nil, fmt.Errorf("failed to load instance name list")
	}

	if !slices.Contains(instanceNameList, instanceName) {
		return nil, fmt.Errorf("instance name '%s' does not exist", instanceName)
	}

	var instanceConfigMap map[string]InstanceConfiguration
	err = p.client.KV.Get(instanceConfigMapKey, &instanceConfigMap)
	if err != nil {
		p.client.Log.Error("Failed to load instance config map while getting instance", "error", err)
		return nil, fmt.Errorf("failed to load instance config map")
	}

	config, ok := instanceConfigMap[instanceName]
	if !ok {
		return nil, fmt.Errorf("instance config for '%s' not found", instanceName)
	}

	return &config, nil
}

func (p *Plugin) uninstallInstance(instanceName string) error {
	var instanceNameList []string
	if err := p.client.KV.Get(instanceConfigNameListKey, &instanceNameList); err != nil {
		p.client.Log.Error("Failed to load instance name list while uninstalling instance", "error", err)
		return fmt.Errorf("failed to load instance name list")
	}

	if !slices.Contains(instanceNameList, instanceName) {
		return fmt.Errorf("instance name '%s' not found in the list", instanceName)
	}

	var instanceConfigMap map[string]InstanceConfiguration
	if err := p.client.KV.Get(instanceConfigMapKey, &instanceConfigMap); err != nil {
		p.client.Log.Error("Failed to load instance config map while uninstalling instance", "error", err)
		return fmt.Errorf("failed to load instance config map")
	}
	if instanceConfigMap == nil {
		return fmt.Errorf("instance config map is empty")
	}

	if _, ok := instanceConfigMap[instanceName]; !ok {
		return fmt.Errorf("instance config for '%s' does not exist", instanceName)
	}

	delete(instanceConfigMap, instanceName)

	if _, err := p.client.KV.Set(instanceConfigMapKey, instanceConfigMap); err != nil {
		p.client.Log.Error("Failed to save updated instance config map while uninstalling instance", "error", err)
		return fmt.Errorf("failed to save updated config map")
	}

	instanceNameList = slices.DeleteFunc(instanceNameList, func(s string) bool { return s == instanceName })

	if _, err := p.client.KV.Set(instanceConfigNameListKey, instanceNameList); err != nil {
		p.client.Log.Error("Failed to save updated instance name list while uninstalling instance", "error", err)
		return fmt.Errorf("failed to save updated instance name list")
	}

	return nil
}

func (p *Plugin) setDefaultInstance(instanceName string) error {
	instanceList := p.getInstanceList()
	if instanceList == nil {
		return fmt.Errorf("failed to load instance list")
	}

	if !slices.Contains(instanceList, instanceName) {
		return fmt.Errorf("instance '%s' does not exist", instanceName)
	}

	config := p.getConfiguration()
	config.DefaultInstanceName = instanceName
	config.sanitize()

	configMap, err := config.ToMap()
	if err != nil {
		p.client.Log.Error("Failed to convert config to map while setting default instance", "error", err)
		return err
	}

	err = p.client.Configuration.SavePluginConfig(configMap)
	if err != nil {
		p.client.Log.Error("Failed to save default instance in plugin config", "error", err)
		return errors.Wrap(err, "failed to save default instance in plugin config")
	}

	return nil
}

func (p *Plugin) getInstanceList() []string {
	var instanceList []string
	err := p.client.KV.Get(instanceConfigNameListKey, &instanceList)
	if err != nil {
		p.client.Log.Error("Failed to load instance name list", "error", err)
		return nil
	}

	return instanceList
}

func (p *Plugin) getInstanceConfigMap() (map[string]InstanceConfiguration, error) {
	var instanceConfigMap map[string]InstanceConfiguration
	err := p.client.KV.Get(instanceConfigMapKey, &instanceConfigMap)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load instance config map")
	}

	if instanceConfigMap == nil {
		return nil, fmt.Errorf("instance config map is empty")
	}

	return instanceConfigMap, nil
}
