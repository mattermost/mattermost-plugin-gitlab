// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

type PropTypes = {
    disabled?: boolean;
    defaultMessage?: string,
    btnClass?: string;
    saving?: boolean;
    savingMessage?: string;
    onClick?: () => void;
};

const FormButton = ({saving, disabled, savingMessage, defaultMessage, btnClass, onClick}: PropTypes) => {
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
};

export default FormButton;
