// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/mattermost/mattermost-plugin-gitlab/server/gitlab"
)

const (
	// 32-byte keys required by AES.
	testOldEncryptionKey = "oldKey0123456789012345678901234x"
	testNewEncryptionKey = "newKey0123456789012345678901234x"
	testUserID           = "testUserID"
	testGitlabUsername   = "testGitlabUser"
	testGitlabUserID     = 42
)

// isNilBytes matches KVSetWithOptions calls with nil payload (i.e. Delete operations).
var isNilBytes = mock.MatchedBy(func(b []byte) bool { return b == nil })

// isNonNilBytes matches KVSetWithOptions calls with a non-nil payload (i.e. Set operations).
var isNonNilBytes = mock.MatchedBy(func(b []byte) bool { return b != nil })

func makeReencryptPlugin(t *testing.T, api *plugintest.API) *Plugin {
	t.Helper()
	p := &Plugin{
		configuration: &configuration{
			EncryptionKey: testNewEncryptionKey,
		},
		BotUserID: "bot-user-id",
	}
	p.SetAPI(api)
	p.client = pluginapi.NewClient(api, p.Driver)
	return p
}

func encryptedTokenWithKey(t *testing.T, key string) []byte {
	t.Helper()
	token := &oauth2.Token{AccessToken: "test-access-token", TokenType: "Bearer"}
	tokenJSON, err := json.Marshal(token)
	require.NoError(t, err)
	encrypted, err := encrypt([]byte(key), string(tokenJSON))
	require.NoError(t, err)
	return []byte(encrypted)
}

func marshaledUserInfo(t *testing.T) []byte {
	t.Helper()
	info := gitlab.UserInfo{
		UserID:         testUserID,
		GitlabUsername: testGitlabUsername,
		GitlabUserID:   testGitlabUserID,
	}
	b, err := json.Marshal(info)
	require.NoError(t, err)
	return b
}

// mockForceDisconnectUser sets up the mocks for a forceDisconnectUser call where user info
// is available. All expectations use Maybe() since forceDisconnectUser is best-effort.
func mockForceDisconnectUser(t *testing.T, api *plugintest.API, userID string) {
	t.Helper()
	infoKey := userID + GitlabUserInfoKey
	api.On("KVGet", infoKey).Return(marshaledUserInfo(t), nil).Maybe()
	// Delete token and info via KVSetWithOptions with nil payload.
	api.On("KVSetWithOptions", userID+GitlabUserTokenKey, isNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Maybe()
	api.On("KVSetWithOptions", userID+GitlabUserInfoKey, isNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Maybe()
	// Username and GitLab ID mapping deletes.
	api.On("KVSetWithOptions", testGitlabUsername+GitlabUsernameKey, isNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Maybe()
	api.On("KVSetWithOptions", fmt.Sprintf("%d%s", testGitlabUserID, GitlabIDUsernameKey), isNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Maybe()
	api.On("GetUser", userID).Return(&model.User{Props: model.StringMap{"git_user": testGitlabUsername}}, nil).Maybe()
	api.On("UpdateUser", mock.Anything).Return(&model.User{}, nil).Maybe()
	api.On("PublishWebSocketEvent", WsEventDisconnect, mock.Anything, mock.Anything).Return(nil).Maybe()
	api.On("GetDirectChannel", userID, "bot-user-id").Return(&model.Channel{Id: "dm-ch"}, nil).Maybe()
	api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Maybe()
}

// mockCommonLogCalls silences log calls that are not under test.
func mockCommonLogCalls(api *plugintest.API) {
	api.On("LogWarn", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe()
}

// ---------------------------------------------------------------------------
// TestUnpad_StrictValidation
// ---------------------------------------------------------------------------

func TestUnpad_StrictValidation(t *testing.T) {
	t.Run("empty input returns error", func(t *testing.T) {
		_, err := unpad([]byte{})
		require.Error(t, err)
	})

	t.Run("padding value zero is rejected", func(t *testing.T) {
		src := []byte{0x41, 0x42, 0x00}
		_, err := unpad(src)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unpad error")
	})

	t.Run("padding value exceeding block size is rejected", func(t *testing.T) {
		// Last byte claims 17 bytes of padding (> aes.BlockSize=16).
		src := make([]byte, 32)
		src[31] = 17
		_, err := unpad(src)
		require.Error(t, err)
	})

	t.Run("mismatched padding bytes are rejected", func(t *testing.T) {
		// Last byte claims 4 bytes of padding, but the third-from-last byte is wrong.
		src := []byte{0x41, 0x42, 0x03, 0x04, 0x04, 0x04}
		_, err := unpad(src)
		require.Error(t, err)
	})

	t.Run("valid PKCS7 padding succeeds", func(t *testing.T) {
		src := []byte{0x41, 0x42, 0x03, 0x03, 0x03}
		result, err := unpad(src)
		require.NoError(t, err)
		assert.Equal(t, []byte{0x41, 0x42}, result)
	})

	t.Run("wrong-key decrypt garbage does not pass unpad", func(t *testing.T) {
		// Encrypt with old key, attempt to decrypt with new key. The stricter unpad
		// validation should reliably reject the garbage output.
		token := &oauth2.Token{AccessToken: "secret-token", TokenType: "Bearer"}
		tokenJSON, marshalErr := json.Marshal(token)
		require.NoError(t, marshalErr)
		ciphertext, encErr := encrypt([]byte(testOldEncryptionKey), string(tokenJSON))
		require.NoError(t, encErr)

		_, err := decrypt([]byte(testNewEncryptionKey), ciphertext)
		assert.Error(t, err, "decrypting with the wrong key should fail with the stricter unpad")
	})
}

// ---------------------------------------------------------------------------
// TestReEncryptUserToken
// ---------------------------------------------------------------------------

func TestReEncryptUserToken_HappyPath(t *testing.T) {
	api := &plugintest.API{}
	p := makeReencryptPlugin(t, api)
	mockCommonLogCalls(api)

	tokenBytes := encryptedTokenWithKey(t, testOldEncryptionKey)
	kvKey := testUserID + GitlabUserTokenKey

	api.On("KVGet", kvKey).Return(tokenBytes, nil).Once()
	var storedBytes []byte
	api.On("KVSetWithOptions", kvKey, isNonNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).
		Run(func(args mock.Arguments) {
			storedBytes = args.Get(1).([]byte)
		}).
		Return(true, nil).Once()

	migrated, err := p.reEncryptUserToken(kvKey, testNewEncryptionKey, testOldEncryptionKey)
	require.NoError(t, err)
	assert.True(t, migrated)

	// Verify the stored ciphertext can be decrypted with the new key and yields the original token.
	require.NotNil(t, storedBytes)
	decrypted, decErr := decrypt([]byte(testNewEncryptionKey), string(storedBytes))
	require.NoError(t, decErr)
	var tok oauth2.Token
	require.NoError(t, json.Unmarshal([]byte(decrypted), &tok))
	assert.Equal(t, "test-access-token", tok.AccessToken)

	api.AssertExpectations(t)
}

func TestReEncryptUserToken_AlreadyMigrated(t *testing.T) {
	api := &plugintest.API{}
	p := makeReencryptPlugin(t, api)

	// Token already encrypted with the new key.
	tokenBytes := encryptedTokenWithKey(t, testNewEncryptionKey)
	kvKey := testUserID + GitlabUserTokenKey

	api.On("KVGet", kvKey).Return(tokenBytes, nil).Once()

	migrated, err := p.reEncryptUserToken(kvKey, testNewEncryptionKey, testOldEncryptionKey)
	require.NoError(t, err)
	assert.False(t, migrated, "should report not migrated when token is already using the new key")

	api.AssertNotCalled(t, "KVSetWithOptions", kvKey, isNonNilBytes, mock.AnythingOfType("model.PluginKVSetOptions"))
	api.AssertExpectations(t)
}

func TestReEncryptUserToken_NilToken(t *testing.T) {
	api := &plugintest.API{}
	p := makeReencryptPlugin(t, api)

	kvKey := testUserID + GitlabUserTokenKey
	api.On("KVGet", kvKey).Return(nil, nil).Once()

	migrated, err := p.reEncryptUserToken(kvKey, testNewEncryptionKey, testOldEncryptionKey)
	require.NoError(t, err)
	assert.False(t, migrated)
	api.AssertExpectations(t)
}

func TestReEncryptUserToken_DecryptFailure(t *testing.T) {
	api := &plugintest.API{}
	p := makeReencryptPlugin(t, api)
	mockCommonLogCalls(api)

	kvKey := testUserID + GitlabUserTokenKey
	// Random garbage that can't be decrypted by either key.
	api.On("KVGet", kvKey).Return([]byte("bm90LXZhbGlkLWJhc2U2NA=="), nil).Once()

	mockForceDisconnectUser(t, api, testUserID)

	migrated, err := p.reEncryptUserToken(kvKey, testNewEncryptionKey, testOldEncryptionKey)
	assert.Error(t, err)
	assert.False(t, migrated)

	// Verify disconnect side-effects were triggered.
	api.AssertCalled(t, "PublishWebSocketEvent", WsEventDisconnect, mock.Anything, mock.Anything)
	api.AssertExpectations(t)
}

func TestReEncryptUserToken_StoreFailure(t *testing.T) {
	api := &plugintest.API{}
	p := makeReencryptPlugin(t, api)
	mockCommonLogCalls(api)

	tokenBytes := encryptedTokenWithKey(t, testOldEncryptionKey)
	kvKey := testUserID + GitlabUserTokenKey

	api.On("KVGet", kvKey).Return(tokenBytes, nil).Once()
	// Re-encrypt store fails.
	api.On("KVSetWithOptions", kvKey, isNonNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).
		Return(false, model.NewAppError("test", "test.store_error", nil, "kv store error", 500)).Once()

	mockForceDisconnectUser(t, api, testUserID)

	migrated, err := p.reEncryptUserToken(kvKey, testNewEncryptionKey, testOldEncryptionKey)
	assert.Error(t, err)
	assert.False(t, migrated)

	api.AssertCalled(t, "PublishWebSocketEvent", WsEventDisconnect, mock.Anything, mock.Anything)
	api.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// TestReEncryptUserData
// ---------------------------------------------------------------------------

func setupReEncryptUserDataPlugin(t *testing.T, api *plugintest.API) *Plugin {
	t.Helper()
	p := makeReencryptPlugin(t, api)
	mockCommonLogCalls(api)
	// Cluster mutex lock/unlock: cluster.NewMutex prepends "mutex_" to the key.
	api.On("KVSetWithOptions", "mutex_gitlab-reencrypt-lock", isNonNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).
		Return(true, nil).Maybe()
	api.On("KVSetWithOptions", "mutex_gitlab-reencrypt-lock", isNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).
		Return(true, nil).Maybe()
	// Audit logging.
	api.On("LogAuditRec", mock.Anything).Maybe()
	return p
}

func TestReEncryptUserData_NoUsers(t *testing.T) {
	api := &plugintest.API{}
	p := setupReEncryptUserDataPlugin(t, api)

	api.On("KVList", 0, 1000).Return([]string{}, nil).Once()

	p.reEncryptUserData(testNewEncryptionKey, testOldEncryptionKey)

	api.AssertExpectations(t)
}

func TestReEncryptUserData_HappyPath(t *testing.T) {
	api := &plugintest.API{}
	p := setupReEncryptUserDataPlugin(t, api)

	user1Key := "user1" + GitlabUserTokenKey
	user2Key := "user2" + GitlabUserTokenKey

	api.On("KVList", 0, 1000).Return([]string{user1Key, user2Key}, nil).Once()

	token1Bytes := encryptedTokenWithKey(t, testOldEncryptionKey)
	token2Bytes := encryptedTokenWithKey(t, testOldEncryptionKey)

	api.On("KVGet", user1Key).Return(token1Bytes, nil).Once()
	api.On("KVGet", user2Key).Return(token2Bytes, nil).Once()

	api.On("KVSetWithOptions", user1Key, isNonNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Once()
	api.On("KVSetWithOptions", user2Key, isNonNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Once()

	p.reEncryptUserData(testNewEncryptionKey, testOldEncryptionKey)

	api.AssertExpectations(t)
}

func TestReEncryptUserData_MultiplePages(t *testing.T) {
	api := &plugintest.API{}
	p := setupReEncryptUserDataPlugin(t, api)

	page0Keys := make([]string, 1000)
	for i := range page0Keys {
		page0Keys[i] = fmt.Sprintf("user%d%s", i, GitlabUserTokenKey)
	}
	api.On("KVList", 0, 1000).Return(page0Keys, nil).Once()
	api.On("KVList", 1, 1000).Return([]string{}, nil).Once()

	// Use Maybe() for per-token operations — the pagination behavior is what this test verifies.
	tokenBytes := encryptedTokenWithKey(t, testOldEncryptionKey)
	api.On("KVGet", mock.AnythingOfType("string")).Return(tokenBytes, nil).Maybe()
	api.On("KVSetWithOptions", mock.AnythingOfType("string"), isNonNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Maybe()

	p.reEncryptUserData(testNewEncryptionKey, testOldEncryptionKey)

	// Verify both pages were fetched.
	api.AssertCalled(t, "KVList", 0, 1000)
	api.AssertCalled(t, "KVList", 1, 1000)
}

func TestReEncryptUserData_ListKeysError(t *testing.T) {
	api := &plugintest.API{}
	p := setupReEncryptUserDataPlugin(t, api)

	api.On("KVList", 0, 1000).Return(nil, model.NewAppError("test", "test.list_error", nil, "kv list error", 500)).Once()

	p.reEncryptUserData(testNewEncryptionKey, testOldEncryptionKey)

	// No token operations should be attempted when the key list fails.
	api.AssertNotCalled(t, "KVGet", testUserID+GitlabUserTokenKey)
	api.AssertExpectations(t)
}

func TestReEncryptUserData_PageErrorAfterFirstPage(t *testing.T) {
	api := &plugintest.API{}
	p := setupReEncryptUserDataPlugin(t, api)

	page0Keys := make([]string, 1000)
	for i := range page0Keys {
		page0Keys[i] = fmt.Sprintf("user%d%s", i, GitlabUserTokenKey)
	}
	api.On("KVList", 0, 1000).Return(page0Keys, nil).Once()
	api.On("KVList", 1, 1000).Return(nil, model.NewAppError("test", "test.list_error", nil, "page 1 error", 500)).Once()

	// Use Maybe() for per-token operations — the key assertion is that page 0 keys
	// are processed even when page 1 returns an error.
	tokenBytes := encryptedTokenWithKey(t, testOldEncryptionKey)
	api.On("KVGet", mock.AnythingOfType("string")).Return(tokenBytes, nil).Maybe()
	api.On("KVSetWithOptions", mock.AnythingOfType("string"), isNonNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Maybe()

	p.reEncryptUserData(testNewEncryptionKey, testOldEncryptionKey)

	// Verify page 0 was fetched and page 1 was attempted.
	api.AssertCalled(t, "KVList", 0, 1000)
	api.AssertCalled(t, "KVList", 1, 1000)
}

func TestReEncryptUserData_DecryptFailureContinuesOtherUsers(t *testing.T) {
	api := &plugintest.API{}
	p := setupReEncryptUserDataPlugin(t, api)

	user1Key := "user1" + GitlabUserTokenKey
	user2Key := "user2" + GitlabUserTokenKey

	api.On("KVList", 0, 1000).Return([]string{user1Key, user2Key}, nil).Once()

	// user1: corrupted ciphertext triggers force disconnect.
	api.On("KVGet", user1Key).Return([]byte("bm90LXZhbGlkLWJhc2U2NA=="), nil).Once()
	mockForceDisconnectUser(t, api, "user1")

	// user2: succeeds.
	token2Bytes := encryptedTokenWithKey(t, testOldEncryptionKey)
	api.On("KVGet", user2Key).Return(token2Bytes, nil).Once()
	api.On("KVSetWithOptions", user2Key, isNonNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Once()

	p.reEncryptUserData(testNewEncryptionKey, testOldEncryptionKey)

	api.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// TestForceDisconnectUser
// ---------------------------------------------------------------------------

func TestForceDisconnectUser_HappyPath(t *testing.T) {
	api := &plugintest.API{}
	p := makeReencryptPlugin(t, api)
	mockCommonLogCalls(api)

	infoKey := testUserID + GitlabUserInfoKey
	api.On("KVGet", infoKey).Return(marshaledUserInfo(t), nil).Once()

	api.On("KVSetWithOptions", testUserID+GitlabUserTokenKey, isNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Once()
	api.On("KVSetWithOptions", testUserID+GitlabUserInfoKey, isNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Once()
	api.On("KVSetWithOptions", testGitlabUsername+GitlabUsernameKey, isNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Once()
	api.On("KVSetWithOptions", fmt.Sprintf("%d%s", testGitlabUserID, GitlabIDUsernameKey), isNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Once()
	api.On("GetUser", testUserID).Return(&model.User{Props: model.StringMap{"git_user": testGitlabUsername}}, nil).Once()
	api.On("UpdateUser", mock.Anything).Return(&model.User{}, nil).Once()
	api.On("PublishWebSocketEvent", WsEventDisconnect, mock.Anything, mock.Anything).Return(nil).Once()
	api.On("GetDirectChannel", testUserID, "bot-user-id").Return(&model.Channel{Id: "dm-ch"}, nil).Once()
	api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Once()

	p.forceDisconnectUser(testUserID)

	api.AssertExpectations(t)
}

func TestForceDisconnectUser_UserInfoNotFound(t *testing.T) {
	api := &plugintest.API{}
	p := makeReencryptPlugin(t, api)
	mockCommonLogCalls(api)

	// User info missing — partial cleanup using only userID.
	infoKey := testUserID + GitlabUserInfoKey
	api.On("KVGet", infoKey).Return(nil, nil).Once()
	// getGitlabUserInfoByMattermostID falls back to the migration key when _userinfo is nil.
	api.On("KVGet", testUserID+GitlabMigrationTokenKey).Return(nil, nil).Once()

	api.On("KVSetWithOptions", testUserID+GitlabUserTokenKey, isNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Once()
	api.On("KVSetWithOptions", testUserID+GitlabUserInfoKey, isNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Once()
	// No username/ID mapping deletes since userInfo is nil (GitlabUsername="" and GitlabUserID=0).
	api.On("GetUser", testUserID).Return(&model.User{Props: model.StringMap{}}, nil).Once()
	api.On("PublishWebSocketEvent", WsEventDisconnect, mock.Anything, mock.Anything).Return(nil).Once()
	api.On("GetDirectChannel", testUserID, "bot-user-id").Return(&model.Channel{Id: "dm-ch"}, nil).Once()
	api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Once()

	p.forceDisconnectUser(testUserID)

	// Mapping deletes must NOT be called when we have no GitLab identity info.
	api.AssertNotCalled(t, "KVSetWithOptions", testGitlabUsername+GitlabUsernameKey, isNilBytes, mock.Anything)
	api.AssertExpectations(t)
}

func TestForceDisconnectUser_DMFailure(t *testing.T) {
	api := &plugintest.API{}
	p := makeReencryptPlugin(t, api)
	mockCommonLogCalls(api)

	infoKey := testUserID + GitlabUserInfoKey
	api.On("KVGet", infoKey).Return(marshaledUserInfo(t), nil).Once()
	api.On("KVSetWithOptions", testUserID+GitlabUserTokenKey, isNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Once()
	api.On("KVSetWithOptions", testUserID+GitlabUserInfoKey, isNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Once()
	api.On("KVSetWithOptions", testGitlabUsername+GitlabUsernameKey, isNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Once()
	api.On("KVSetWithOptions", fmt.Sprintf("%d%s", testGitlabUserID, GitlabIDUsernameKey), isNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Once()
	api.On("GetUser", testUserID).Return(&model.User{Props: model.StringMap{}}, nil).Once()
	api.On("PublishWebSocketEvent", WsEventDisconnect, mock.Anything, mock.Anything).Return(nil).Once()

	// DM channel lookup fails — must be handled gracefully without panicking.
	api.On("GetDirectChannel", testUserID, "bot-user-id").Return(nil, &model.AppError{Message: "channel not found"}).Once()

	p.forceDisconnectUser(testUserID)

	api.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// TestDisconnectGitlabAccount_FallbackWhenUserInfoMissing
// ---------------------------------------------------------------------------

func TestDisconnectGitlabAccount_FallbackWhenUserInfoMissing(t *testing.T) {
	api := &plugintest.API{}
	p := makeReencryptPlugin(t, api)
	mockCommonLogCalls(api)

	// getGitlabUserInfoByMattermostID reads _userinfo then _gitlabtoken; both return nil → not connected.
	// This is called twice: once in disconnectGitlabAccount and once inside forceDisconnectUser.
	api.On("KVGet", testUserID+GitlabUserInfoKey).Return(nil, nil)
	api.On("KVGet", testUserID+GitlabMigrationTokenKey).Return(nil, nil)

	// forceDisconnectUser best-effort cleanup (no user info available).
	api.On("KVSetWithOptions", testUserID+GitlabUserTokenKey, isNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Once()
	api.On("KVSetWithOptions", testUserID+GitlabUserInfoKey, isNilBytes, mock.AnythingOfType("model.PluginKVSetOptions")).Return(true, nil).Once()
	api.On("GetUser", testUserID).Return(&model.User{Props: model.StringMap{"git_user": "someUser"}}, nil).Once()
	api.On("UpdateUser", mock.Anything).Return(&model.User{}, nil).Once()
	api.On("PublishWebSocketEvent", WsEventDisconnect, mock.Anything, mock.Anything).Return(nil).Once()
	api.On("GetDirectChannel", testUserID, "bot-user-id").Return(&model.Channel{Id: "dm-ch"}, nil).Once()
	api.On("CreatePost", mock.Anything).Return(&model.Post{}, nil).Once()

	p.disconnectGitlabAccount(testUserID)

	// The websocket disconnect event must still be published despite user info being missing.
	api.AssertCalled(t, "PublishWebSocketEvent", WsEventDisconnect, mock.Anything, mock.Anything)
	api.AssertExpectations(t)
}
