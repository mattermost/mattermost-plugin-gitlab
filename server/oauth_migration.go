package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-plugin-api/cluster"
	"github.com/pkg/errors"
)

const (
	oauthMigrationStoreKey = "oauth_migration"
	oauthMigrationMutexKey = "oauth_migration_mutex"

	oauthMigrationStatusInProgress = "IN_PROGRESS"
	oauthMigrationStatusComplete   = "COMPLETE"
	oauthMigrationStatusError      = "ERROR"
)

func (p *Plugin) checkAndPerformOAuthTokenMigration() error {
	mutexAPI := cluster.MutexPluginAPI(p.API)
	mutex, err := cluster.NewMutex(mutexAPI, oauthMigrationMutexKey)
	if err != nil {
		return err
	}

	mutex.Lock()
	defer mutex.Unlock()

	var status string
	err = p.client.KV.Get(oauthMigrationStoreKey, &status)
	if err != nil {
		return err
	}

	// Migration is in progress or already completed
	if status != "" {
		return nil
	}

	_, err = p.client.KV.Set(oauthMigrationStoreKey, oauthMigrationStatusInProgress)
	if err != nil {
		p.client.Log.Warn("error setting migration status to "+oauthMigrationStatusInProgress, "error", err.Error())
	}

	err = p.notifyAllConnectedUsersToReconnect()
	if err != nil {
		_, kvErr := p.client.KV.Set(oauthMigrationStoreKey, oauthMigrationStatusError)
		if kvErr != nil {
			p.client.Log.Warn("error setting migration status to "+oauthMigrationStatusError, "error", err.Error())
		}

		return err
	}

	_, err = p.client.KV.Set(oauthMigrationStoreKey, oauthMigrationStatusComplete)
	if err != nil {
		p.client.Log.Warn("error setting migration status to "+oauthMigrationStatusComplete, "error", err.Error())
	}

	return nil
}

func (p *Plugin) notifyAllConnectedUsersToReconnect() error {
	allKeys := []string{}
	page := 0
	for {
		keys, err := p.client.KV.ListKeys(page, 100)
		if err != nil {
			return errors.Wrap(err, "error listing keys for connected users")
		}

		if len(keys) == 0 {
			break
		}

		keysToAdd := []string{}
		for _, key := range keys {
			if strings.HasSuffix(key, GitlabTokenKey) {
				keysToAdd = append(keysToAdd, key)
			}
		}

		allKeys = append(allKeys, keysToAdd...)
		page++
	}

	updateMessage := "An update for this integration requires you to reconnect your account."
	connectMessage := fmt.Sprintf(gitlabConnectMessage, *p.client.Configuration.GetConfig().ServiceSettings.SiteURL, manifest.Id)
	fullMessage := fmt.Sprintf("%s %s", updateMessage, connectMessage)

	numErrors := 0
	for _, key := range allKeys {
		index := strings.Index(key, GitlabTokenKey)
		userID := key[:index]

		var logError = func(msg string, err error) {
			numErrors++
			if numErrors < 10 {
				p.client.Log.Warn(msg, "user_id", userID, "err", err.Error())
			}
		}

		_, err := p.poster.DM(userID, fullMessage)
		if err != nil {
			logError("error notifying user to reconnect", err)
		}

		err = p.client.KV.Delete(key)
		if err != nil {
			logError("error deleting key for connected user", err)
		}
	}

	return nil
}
