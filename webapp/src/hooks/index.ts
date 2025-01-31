// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Store} from 'redux';

import {isDesktopApp} from 'src/utils/user_agent';
import {connectUsingBrowserMessage} from 'src/constants';

import {sendEphemeralPost} from '../actions';
import {ContextArgs} from '../types/mattermost-webapp';

const connectCommand = '/gitlab connect';

export default class Hooks {
    private store: Store;

    constructor(store: Store) {
        this.store = store;
    }

    slashCommandWillBePostedHook = (rawMessage: string, contextArgs: ContextArgs) => {
        let message;
        if (rawMessage) {
            message = rawMessage.trim();
        }

        if (!message) {
            return Promise.resolve({message, args: contextArgs});
        }

        if (message.startsWith(connectCommand) && isDesktopApp()) {
            sendEphemeralPost(connectUsingBrowserMessage)(this.store.dispatch, this.store.getState);
            return Promise.resolve({});
        }

        return Promise.resolve({message, args: contextArgs});
    }
}
