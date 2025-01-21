// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState} from 'react';
import {useDispatch, useSelector} from 'react-redux';
import {Modal} from 'react-bootstrap';
import {Theme} from 'mattermost-redux/types/preferences';

import {isCreateIssueModalVisible} from 'src/selectors';
import {closeCreateIssueModal} from 'src/actions';

import CreateIssueForm from './create_issue_form';

type PropTypes = {
    theme: Theme;
};

const CreateIssueModal = ({theme}: PropTypes) => {
    const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

    const dispatch = useDispatch();
    const handleClose = () => {
        setIsSubmitting(false);
        dispatch(closeCreateIssueModal());
    };

    const visible = useSelector(isCreateIssueModalVisible);
    if (!visible) {
        return null;
    }

    return (
        <Modal
            dialogClassName='modal--scroll'
            show={true}
            onHide={handleClose}
            onExited={handleClose}
            bsSize='large'
            backdrop='static'
        >
            <Modal.Header closeButton={true}>
                <Modal.Title>
                    {'Create GitLab Issue'}
                </Modal.Title>
            </Modal.Header>
            <CreateIssueForm
                handleClose={handleClose}
                setIsSubmitting={setIsSubmitting}
                isSubmitting={isSubmitting}
                theme={theme}
            />
        </Modal>
    );
};

export default CreateIssueModal;
