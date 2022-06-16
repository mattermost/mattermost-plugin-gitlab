// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';

interface PropTypes {
    disabled?: boolean;
    defaultMessage?: string,
    btnClass?: string;
    saving?: boolean;
    savingMessage?: string;
    onClick?: () => void;
};

export default class FormButton extends PureComponent<PropTypes> {
    render() {
        const {saving, disabled, savingMessage, defaultMessage, btnClass, onClick} = this.props;

        const contents = saving ? (
            <span>
                <span
                    className='fa fa-spin fa-spinner'
                    title={'Loading Icon'}
                />
                {savingMessage}
            </span>
        ) : defaultMessage;

        const className = `save-button btn ${btnClass}`;

        return (
            <button
                id='saveSetting'
                className={className}
                disabled={disabled}
                onClick={onClick}
            >
                {contents}
            </button>
        );
    }
}
