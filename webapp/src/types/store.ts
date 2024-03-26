import {GlobalState as ReduxGlobalState} from 'mattermost-redux/types/store';

import combinedReducers from '../reducers';

import {Item} from './gitlab_items';

import {GitlabUsersData, LHSData, ShowRhsPluginActionData, UserSettingsData} from '.';

export type GlobalState = ReduxGlobalState & {
    'plugins-com.github.manland.mattermost-plugin-gitlab': PluginState
};

export type PluginState = ReturnType<typeof combinedReducers>

export type pluginStateKey = 'plugins-com.github.manland.mattermost-plugin-gitlab'
