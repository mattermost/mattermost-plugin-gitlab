// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/pkg/errors"
)

type InstanceConfiguration struct {
	GitlabURL               string `json:"gitlaburl"`
	GitlabOAuthClientID     string `json:"gitlaboauthclientid"`
	GitlabOAuthClientSecret string `json:"gitlaboauthclientsecret"`
}

func (c *InstanceConfiguration) Validate() error {
	if c == nil {
		return errors.New("instance configuration is nil")
	}

	if err := isValidURL(c.GitlabURL); err != nil {
		return errors.Wrap(err, "must have a valid GitLab URL")
	}

	if c.GitlabOAuthClientID == "" {
		return errors.New("must have a GitLab OAuth client ID")
	}

	if c.GitlabOAuthClientSecret == "" {
		return errors.New("must have a GitLab OAuth client secret")
	}

	return nil
}

const (
	instanceConfigMapKey      = "Gitlab_Instance_Configuration_Map"
	instanceConfigNameListKey = "Gitlab_Instance_Configuration_Name_List"
)

// ErrInstanceNotFound reports that the named instance is absent from the instance store, as
// opposed to the store being unreadable. Callers use it to tell a missing instance apart from
// a storage failure, which must not be treated as "nothing is configured".
var ErrInstanceNotFound = errors.New("instance not found")

func (p *Plugin) installInstance(instanceName string, config *InstanceConfiguration) error {
	if config == nil {
		return errors.New("config is nil")
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid instance configuration: %w", err)
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

	// installInstance only mutates the KV store; refresh here since OnConfigurationChange won't
	// run unless setDefaultInstance also changed plugin settings.
	p.refreshEffectiveInstance()

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
		return nil, fmt.Errorf("instance name '%s' does not exist: %w", instanceName, ErrInstanceNotFound)
	}

	var instanceConfigMap map[string]InstanceConfiguration
	err = p.client.KV.Get(instanceConfigMapKey, &instanceConfigMap)
	if err != nil {
		p.client.Log.Error("Failed to load instance config map while getting instance", "error", err)
		return nil, fmt.Errorf("failed to load instance config map")
	}

	config, ok := instanceConfigMap[instanceName]
	if !ok {
		return nil, fmt.Errorf("instance config for '%s' not found: %w", instanceName, ErrInstanceNotFound)
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

	remaining := slices.DeleteFunc(slices.Clone(instanceNameList), func(s string) bool { return s == instanceName })

	// Update DefaultInstanceName before mutating KV so a SavePluginConfig failure leaves the
	// instance intact and allows the admin to retry uninstall cleanly.
	if p.getConfiguration().DefaultInstanceName == instanceName {
		nextDefault := ""
		if len(remaining) > 0 {
			nextDefault = remaining[0]
		}
		if err := p.saveDefaultInstanceName(nextDefault); err != nil {
			return fmt.Errorf("failed to update default instance: %w", err)
		}
	}

	err := p.client.KV.SetAtomicWithRetries(instanceConfigMapKey, func(oldValue []byte) (any, error) {
		var instanceConfigMap map[string]InstanceConfiguration
		if len(oldValue) > 0 {
			if unmarshalErr := json.Unmarshal(oldValue, &instanceConfigMap); unmarshalErr != nil {
				return nil, fmt.Errorf("failed to unmarshal instance config map: %w", unmarshalErr)
			}
		}
		if instanceConfigMap == nil {
			return nil, fmt.Errorf("instance config map is empty")
		}
		if _, ok := instanceConfigMap[instanceName]; !ok {
			return nil, fmt.Errorf("instance config for '%s' does not exist", instanceName)
		}

		delete(instanceConfigMap, instanceName)
		return instanceConfigMap, nil
	})
	if err != nil {
		p.client.Log.Error("Failed to save updated instance config map while uninstalling instance", "error", err)
		return err
	}

	if _, err := p.client.KV.Set(instanceConfigNameListKey, remaining); err != nil {
		p.client.Log.Error("Failed to save updated instance name list while uninstalling instance", "error", err)
		return fmt.Errorf("failed to save updated instance name list")
	}

	// uninstallInstance only mutates the KV store; refresh in case the uninstalled instance was
	// the default one.
	p.refreshEffectiveInstance()

	return nil
}

// updateInstance updates the URL and OAuth credentials of an existing named instance.
// If config.GitlabOAuthClientSecret is empty, the existing secret is preserved atomically.
func (p *Plugin) updateInstance(instanceName string, config *InstanceConfiguration) error {
	if config == nil {
		return errors.New("config is nil")
	}

	var instanceNameList []string
	if err := p.client.KV.Get(instanceConfigNameListKey, &instanceNameList); err != nil {
		p.client.Log.Error("Failed to load instance name list while updating instance", "error", err)
		return fmt.Errorf("failed to load instance name list")
	}

	if !slices.Contains(instanceNameList, instanceName) {
		return fmt.Errorf("instance name '%s' does not exist: %w", instanceName, ErrInstanceNotFound)
	}

	err := p.client.KV.SetAtomicWithRetries(instanceConfigMapKey, func(oldValue []byte) (any, error) {
		var instanceConfigMap map[string]InstanceConfiguration
		if len(oldValue) > 0 {
			if unmarshalErr := json.Unmarshal(oldValue, &instanceConfigMap); unmarshalErr != nil {
				return nil, fmt.Errorf("failed to unmarshal instance config map: %w", unmarshalErr)
			}
		}

		existing, ok := instanceConfigMap[instanceName]
		if !ok {
			return nil, fmt.Errorf("instance config for '%s' not found: %w", instanceName, ErrInstanceNotFound)
		}

		updated := *config
		if updated.GitlabOAuthClientSecret == "" {
			updated.GitlabOAuthClientSecret = existing.GitlabOAuthClientSecret
		}
		if validateErr := updated.Validate(); validateErr != nil {
			return nil, fmt.Errorf("invalid instance configuration: %w", validateErr)
		}

		instanceConfigMap[instanceName] = updated
		return instanceConfigMap, nil
	})
	if err != nil {
		p.client.Log.Error("Failed to save updated instance config map while updating instance", "error", err)
		return err
	}

	p.refreshEffectiveInstance()

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

	return p.saveDefaultInstanceName(instanceName)
}

func (p *Plugin) saveDefaultInstanceName(instanceName string) error {
	config := p.getConfiguration().Clone()
	config.DefaultInstanceName = instanceName
	config.sanitize()

	configMap, err := config.ToMap()
	if err != nil {
		p.client.Log.Error("Failed to convert config to map while updating default instance", "error", err)
		return err
	}

	if err := p.client.Configuration.SavePluginConfig(configMap); err != nil {
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
