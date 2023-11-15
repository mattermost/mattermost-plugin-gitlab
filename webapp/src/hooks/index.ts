// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import { isDesktopApp } from 'src/utils/user_agent';
import {handleConnectFlow} from '../actions';

type ContextArgs = {channel_id: string};

const connectCommand = '/gitlab connect';

export default class Hooks {
    private store: any;

    constructor(store: any) {
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

        if (message.startsWith(connectCommand)) {
            if (isDesktopApp()){
                this.store.dispatch(handleConnectFlow());
                return Promise.resolve({});
            }
        }

        return Promise.resolve({message, args: contextArgs});
    }
}
