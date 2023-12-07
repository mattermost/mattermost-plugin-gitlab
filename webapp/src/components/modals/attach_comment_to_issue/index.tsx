// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useState} from 'react';
import {Modal} from 'react-bootstrap';
import {Theme} from 'mattermost-redux/types/preferences';
import {useDispatch, useSelector} from 'react-redux';
import {getPost} from 'mattermost-redux/selectors/entities/posts';

import FormButton from 'src/components/form_button';
import Input from 'src/components/input';
import Validator from 'src/components/validator';
import {id as pluginId} from 'src/manifest';
import GitlabIssueSelector from 'src/components/gitlab_issue_selector';
import {getErrorMessage} from 'src/utils/user_utils';
import {closeAttachCommentToIssueModal, attachCommentToIssue} from 'src/actions';
import {GlobalState} from 'src/types/global_state';

interface PropTypes {  
    theme: Theme,
}

const AttachCommentToIssueModal = ({theme}: PropTypes) => {
    const validator = new Validator();
    const [submitting, setSubmitting] = useState(false);
    const [issueValue, setIssueValue] = useState<Issue | null>(null);
    const [error, setError] = useState<string>('')

    const {post, visible} = useSelector((state: GlobalState) => {
        const postId = state[`plugins-${pluginId}` as plugin].postIdForAttachCommentToIssueModal;
        const post = getPost(state, postId);
    
        return {
            visible: state[`plugins-${pluginId}` as plugin].attachCommentToIssueModalVisible,
            post,
        };
    })

    const dispatch = useDispatch();

    const handleCreate = async (e: React.FormEvent<HTMLFormElement> | Event) => {
        e.preventDefault();
        
        if (!validator.validate()) {
            return;
        }

        const comment = {
            project_id: issueValue?.project_id,
            iid: issueValue?.iid,
            comment: post.message,
            post_id: post.id,
            web_url: issueValue?.web_url,
        };

        setSubmitting(true);

        const created = await attachCommentToIssue(comment)(dispatch);
        if (created.error) {
            const errMessage = getErrorMessage((created as {error: ErrorType}).error.message);
            setError(errMessage);
            setSubmitting(false);
            return;
        }

        handleClose();
    };

    const handleClose = () => {
        setError('');
        setSubmitting(false);
        setIssueValue(null);
        dispatch(closeAttachCommentToIssueModal());
    };

    const handleIssueValueChange = (newValue: Issue | null) => {
        setIssueValue(newValue);
    };

    const style = getStyle(theme);

    if (!visible) {
        return null;
    }

    const component = (
        <div>
            <GitlabIssueSelector
                name={'issue'}
                onChange={handleIssueValueChange}
                required={true}
                theme={theme}
                error={error}
                value={issueValue}
                addValidate={validator.addComponent}
                removeValidate={validator.removeComponent}
            />
            <Input
                id={'comment'}
                label='Message Attached to GitLab Issue'
                type='textarea'
                required={false}
                value={post.message}
                disabled={false}
                readOnly={true}
            />
        </div>
    );

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
                    {component}
                </Modal.Body>
                <Modal.Footer>
                    <FormButton
                        btnClass='btn-link'
                        defaultMessage='Cancel'
                        onClick={handleClose}
                    />
                    <FormButton
                        btnClass='btn btn-primary'
                        saving={submitting}
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
