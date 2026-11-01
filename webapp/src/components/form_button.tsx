// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

type Props = {
    type?: 'button' | 'submit' | 'reset';
    disabled?: boolean;
    defaultMessage?: string;
    btnClass?: string;
    saving?: boolean;
    savingMessage?: string;
    onClick?: () => void;
};

const FormButton = ({type = 'button', saving, disabled, savingMessage, defaultMessage, btnClass, onClick}: Props) => {
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
            type={type}
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
