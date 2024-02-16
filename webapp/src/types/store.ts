import {GlobalState as ReduxGlobalState} from 'mattermost-redux/types/store';

import {Item} from './gitlab_items';

import {LHSData} from '.';

export type GlobalState = ReduxGlobalState & {
    'plugins-com.github.manland.mattermost-plugin-gitlab': PluginState
};

export type PluginState = {
    connected: boolean,
    gitlabURL: string,
    organization: string,
    username: string,
    reviewDetails: Item[],
    yourPrDetails: Item[],
    lhsData: LHSData,
    rhsState: string,
};

export type pluginStateKey = 'plugins-com.github.manland.mattermost-plugin-gitlab'
