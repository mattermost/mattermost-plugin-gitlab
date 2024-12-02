import {GlobalState as ReduxGlobalState} from 'mattermost-redux/types/store';

import type combinedReducers from '../reducers';

export type GlobalState = ReduxGlobalState & {
    'plugins-com.github.manland.mattermost-plugin-gitlab': PluginState
};

export type PluginState = ReturnType<typeof combinedReducers>

export const pluginStateKey = 'plugins-com.github.manland.mattermost-plugin-gitlab' as const;
