// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState} from 'react';
import {Modal} from 'react-bootstrap';
import {Theme} from 'mattermost-redux/types/preferences';
import {useDispatch, useSelector} from 'react-redux';

import FormButton from 'src/components/form_button';
import {id as pluginId} from 'src/manifest';
import {closeAttachCommentToIssueModal} from 'src/actions';
import {GlobalState} from 'src/types/global_state';
import AttachCommentToIssueForm from 'src/components/attach_comment_to_issue_form';

interface PropTypes {  
    theme: Theme,
}

const AttachCommentToIssueModal = ({theme}: PropTypes) => {
    const [formSubmission, setFormSubmission] = useState<FormSubmission>({
        isSubmitted: false,
        isSubmitting: false,
        error: '',
    });

    const visible = useSelector((state: GlobalState) => state[`plugins-${pluginId}` as plugin].attachCommentToIssueModalVisible);

    const handleCreate = async (e: React.FormEvent<HTMLFormElement> | Event) => {
        e.preventDefault();
        setFormSubmission({
            ...formSubmission,
            isSubmitted: true,
        })
    };

    const dispatch = useDispatch();
    const handleClose = () => {
        setFormSubmission({
            isSubmitted: false,
            isSubmitting: false,
            error: '',
        })
        dispatch(closeAttachCommentToIssueModal());
    };

    const style = getStyle(theme);

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
            <form
                role='form'
                onSubmit={handleCreate}
            >
                <Modal.Body
                    style={style.modal}
                >
                    <AttachCommentToIssueForm
                        formSubmission={formSubmission}
                        setFormSubmission={setFormSubmission}
                        handleClose={handleClose}
                        theme={theme}
                    />
                </Modal.Body>
                <Modal.Footer>
                    <FormButton
                        btnClass='btn-link'
                        defaultMessage='Cancel'
                        onClick={handleClose}
                    />
                    <FormButton
                        btnClass='btn btn-primary'
                        saving={formSubmission.isSubmitting}
                        defaultMessage='Attach'
                        savingMessage='Attaching'
                    />
                </Modal.Footer>
            </form>
        </Modal>
    );
}

const getStyle = (theme: Theme) => ({
    modal: {
        padding: '2em 2em 3em',
        color: theme.centerChannelColor,
        backgroundColor: theme.centerChannelBg,
    },
});

export default AttachCommentToIssueModal;
