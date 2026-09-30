// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
)

// configuration captures the plugin's external configuration as exposed in the Mattermost server
// configuration, as well as values computed from the configuration. Any public fields will be
// deserialized from the Mattermost server configuration in OnConfigurationChange.
//
// As plugins are inherently concurrent (hooks being called asynchronously), and the plugin
// configuration can change at any time, access to the configuration must be synchronized. The
// strategy used in this plugin is to guard a pointer to the configuration, and clone the entire
// struct whenever it changes. You may replace this with whatever strategy you choose.
//
// If you add non-reference types to your configuration struct, be sure to rewrite Clone as a deep
// copy appropriate for your types.
type configuration struct {
	GitlabURL                        string `json:"gitlaburl"`
	GitlabOAuthClientID              string `json:"gitlaboauthclientid"`
	GitlabOAuthClientSecret          string `json:"gitlaboauthclientsecret"`
	DefaultInstanceName              string `json:"defaultinstancename"`
	WebhookSecret                    string `json:"webhooksecret"`
	EncryptionKey                    string `json:"encryptionkey"`
	GitlabGroup                      string `json:"gitlabgroup"`
	EnablePrivateRepo                bool   `json:"enableprivaterepo"`
	EnableCodePreview                string `json:"enablecodepreview"`
	UsePreregisteredApplication      bool   `json:"usepreregisteredapplication"`
	EnableChildPipelineNotifications bool   `json:"enablechildpipelinenotifications"`

	// PreviousEncryptionKey is set internally during key rotation so that token
	// reads can fall back to the old key while background re-encryption runs.
	// It is never persisted to the plugin settings.
	PreviousEncryptionKey string `json:"-"`
}

// Clone shallow copies the configuration. Your implementation may require a deep copy if
// your configuration has reference types.
func (c *configuration) Clone() *configuration {
	clone := *c
	return &clone
}

func (c *configuration) ToMap() (map[string]any, error) {
	var out map[string]any
	data, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (c *configuration) setDefaults(isCloud bool) (bool, error) {
	changed := false

	if c.EncryptionKey == "" {
		secret, err := generateSecret()
		if err != nil {
			return false, err
		}

		c.EncryptionKey = secret
		changed = true
	}

	if c.WebhookSecret == "" {
		secret, err := generateSecret()
		if err != nil {
			return false, err
		}

		c.WebhookSecret = secret
		changed = true
	}

	if isCloud && !c.UsePreregisteredApplication && !c.IsOAuthConfigured() {
		c.UsePreregisteredApplication = true
		changed = true
	}

	return changed, nil
}

func (c *configuration) sanitize() {
	// Ensure GitlabURL ends with a slash
	c.GitlabURL = strings.TrimRight(c.GitlabURL, "/")

	// Trim spaces around org and OAuth credentials
	c.GitlabGroup = strings.TrimSpace(c.GitlabGroup)
	c.GitlabOAuthClientID = strings.TrimSpace(c.GitlabOAuthClientID)
	c.GitlabOAuthClientSecret = strings.TrimSpace(c.GitlabOAuthClientSecret)
}

func (c *configuration) IsOAuthConfigured() bool {
	return (c.GitlabOAuthClientID != "" && c.GitlabOAuthClientSecret != "") ||
		c.UsePreregisteredApplication
}

// IsSASS return if SASS gitlab at https://gitlab.com is used
func (c *configuration) IsSASS() bool {
	return c.GitlabURL == "https://gitlab.com"
}

// IsValid checks the invariants that hold regardless of where OAuth credentials for the
// effective GitLab instance live (legacy plugin settings or the KV-backed instance store).
// Callers that need to know whether a GitLab instance is actually configured should use
// Plugin.isConfigured, which additionally resolves OAuth credentials.
func (c *configuration) IsValid() error {
	if c.UsePreregisteredApplication && !c.IsSASS() {
		return errors.New("pre-registered application can only be used with official public GitLab")
	}

	if c.EncryptionKey == "" {
		return errors.New("must have an encryption key")
	}

	return nil
}

// getConfiguration retrieves the active configuration under lock, making it safe to use
// concurrently. The active configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (p *Plugin) getConfiguration() *configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &configuration{}
	}

	return p.configuration
}

// setConfiguration replaces the active configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex is not
// reentrant. In particular, avoid using the plugin API entirely, as this may in turn trigger a
// hook back into the plugin. If that hook attempts to acquire this lock, a deadlock may occur.
//
// This method panics if setConfiguration is called with the existing configuration. This almost
// certainly means that the configuration was modified without being cloned and may result in
// an unsafe access.
func (p *Plugin) setConfiguration(configuration *configuration, serverConfiguration *model.Config) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if configuration != nil && p.configuration == configuration {
		// Ignore assignment if the configuration struct is empty. Go will optimize the
		// allocation for same to point at the same memory address, breaking the check
		// above.
		if reflect.ValueOf(*configuration).NumField() == 0 {
			return
		}

		panic("setConfiguration called with the existing configuration")
	}

	p.configuration = configuration
}

// OnConfigurationChange is invoked when configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	if p.client == nil {
		p.client = pluginapi.NewClient(p.API, p.Driver)
	}

	configuration := new(configuration)

	// Load the public configuration fields from the Mattermost server configuration.
	if err := p.client.Configuration.LoadPluginConfiguration(configuration); err != nil {
		return errors.Wrap(err, "failed to load plugin configuration")
	}

	configuration.sanitize()

	serverConfiguration := p.client.Configuration.GetConfig()
	p.configurationLock.RLock()
	hadConfig := p.configuration != nil
	var previousGitlabGroup string
	var previousEncryptionKey string
	if hadConfig {
		previousGitlabGroup = strings.TrimSpace(p.configuration.GitlabGroup)
		previousEncryptionKey = p.configuration.EncryptionKey
	}
	p.configurationLock.RUnlock()
	newGitlabGroup := strings.TrimSpace(configuration.GitlabGroup)

	if previousEncryptionKey != "" && configuration.EncryptionKey != "" &&
		previousEncryptionKey != configuration.EncryptionKey {
		configuration.PreviousEncryptionKey = previousEncryptionKey
	}

	p.setConfiguration(configuration, serverConfiguration)

	if configuration.PreviousEncryptionKey != "" {
		newKey := configuration.EncryptionKey
		prevKey := configuration.PreviousEncryptionKey
		go p.reEncryptUserData(newKey, prevKey)
	}

	if hadConfig && p.BotUserID != "" && newGitlabGroup != "" && newGitlabGroup != previousGitlabGroup {
		p.notifyUsersOfDisallowedSubscriptions()
	}

	// Resolve the effective instance before registering the command, since the autocomplete
	// data depends on whether a GitLab instance is configured.
	p.refreshGitlabClient()

	return p.registerCommand()
}

// refreshEffectiveInstance re-resolves everything that depends on the effective GitLab instance
// and tells the other cluster nodes to do the same. Call it after KV mutations such as
// installing or uninstalling an instance, which do not go through OnConfigurationChange.
func (p *Plugin) refreshEffectiveInstance() {
	p.refreshLocalEffectiveInstance()
	p.sendInstanceChangedEvent()
}

func (p *Plugin) refreshLocalEffectiveInstance() {
	p.refreshGitlabClient()

	if err := p.registerCommand(); err != nil {
		p.client.Log.Error("Failed to re-register slash command after instance change", "error", err.Error())
	}
}

func (p *Plugin) registerCommand() error {
	command, err := p.getCommand(p.getConfiguration())
	if err != nil {
		return errors.Wrap(err, "failed to get command")
	}

	if err := p.client.SlashCommand.Register(command); err != nil {
		return errors.Wrap(err, "failed to register command")
	}

	return nil
}

// refreshGitlabClient re-resolves the effective GitLab instance and caches it along with an API
// client built from its URL. The instance may come from the KV-backed default instance rather
// than legacy plugin settings, so call this after any change that could affect it.
func (p *Plugin) refreshGitlabClient() {
	config := p.getConfiguration()

	if config.UsePreregisteredApplication {
		effective := &effectiveConfig{GitlabURL: config.GitlabURL}
		p.setEffective(effective, gitlab.New(effective.GitlabURL, config.GitlabGroup, p.isNamespaceAllowed))
		return
	}

	effective, err := p.resolveEffectiveConfig(config)
	switch {
	case err == nil:
		p.setEffective(effective, gitlab.New(effective.GitlabURL, config.GitlabGroup, p.isNamespaceAllowed))
	case errors.Is(err, ErrNotConfigured):
		// Nothing is configured, so report that, but keep the existing client: rebuilding it
		// from legacy settings would aim it at a host the stored tokens were not issued for.
		p.setEffective(nil, nil)
	default:
		// The instance store is unreadable, leaving the effective instance unknown. Keep the
		// last known good state and retry on the next refresh.
		p.client.Log.Warn("Keeping last resolved GitLab instance, failed to read instance store", "error", err.Error())
	}
}

func generateSecret() (string, error) {
	b := make([]byte, 256)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	s := base64.RawStdEncoding.EncodeToString(b)

	s = s[:32]

	return s, nil
}
