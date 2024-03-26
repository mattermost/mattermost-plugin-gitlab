import {GlobalState as ReduxGlobalState} from 'mattermost-redux/types/store';

import combinedReducers from '../reducers';

export type GlobalState = ReduxGlobalState & {
    'plugins-com.github.manland.mattermost-plugin-gitlab': PluginState
};

export type PluginState = ReturnType<typeof combinedReducers>

export type pluginStateKey = 'plugins-com.github.manland.mattermost-plugin-gitlab'
