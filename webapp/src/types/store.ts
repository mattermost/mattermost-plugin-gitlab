import {GlobalState as ReduxGlobalState} from 'mattermost-redux/types/store';

import {Item} from './gitlab_items';

import {GitlabUsersData, LHSData, ShowRhsPluginActionData, UserSettingsData} from '.';

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
    settings: UserSettingsData,
    gitlab_client_id: string;
    gitlabUsers: Record<string, GitlabUsersData>,
    rhsPluginAction: ShowRhsPluginActionData,
};

export type pluginStateKey = 'plugins-com.github.manland.mattermost-plugin-gitlab'
