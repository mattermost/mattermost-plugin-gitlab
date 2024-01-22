// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState} from 'react';
import {Modal} from 'react-bootstrap';
import {Theme} from 'mattermost-redux/types/preferences';
import {useDispatch, useSelector} from 'react-redux';

import AttachCommentToIssueForm from './attach_comment_to_issue_form';
import {isAttachCommentToIssueModalVisible} from 'src/selectors';
import {closeAttachCommentToIssueModal} from 'src/actions';

interface PropTypes {  
    theme: Theme,
}

const AttachCommentToIssueModal = ({theme}: PropTypes) => {
    const [isSubmitting, setIsSubmitting] = useState<boolean>(false);

    const dispatch = useDispatch();
    const handleClose = () => {
        setIsSubmitting(false)
        dispatch(closeAttachCommentToIssueModal());
    };

    const visible = useSelector(isAttachCommentToIssueModalVisible);
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
                    {'Attach Message to GitLab Issue'}
                </Modal.Title>
            </Modal.Header>
            <AttachCommentToIssueForm
                isSubmitting={isSubmitting}
                setIsSubmitting={setIsSubmitting}
                handleClose={handleClose}
                theme={theme}
            />
        </Modal>
    );
}

export default AttachCommentToIssueModal;
