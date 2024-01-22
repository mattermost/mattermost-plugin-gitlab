import React, {useMemo, useState} from 'react';
import {Theme} from 'mattermost-redux/types/preferences';
import {useDispatch, useSelector} from 'react-redux';
import {Modal} from 'react-bootstrap';

import Input from 'src/components/input';
import Validator from 'src/components/validator';
import GitlabIssueSelector from 'src/components/gitlab_issue_selector';
import {getErrorMessage} from 'src/utils/user_utils';
import {attachCommentToIssue} from 'src/actions';
import {getAttachCommentModalContents} from 'src/selectors';
import FormButton from '../form_button';

interface PropTypes {
    theme: Theme;
    handleClose: () => void;
    setIsSubmitting: React.Dispatch<React.SetStateAction<boolean>>;
    isSubmitting: boolean;
}

const AttachCommentToIssueForm = ({ theme, handleClose, setIsSubmitting, isSubmitting }: PropTypes) => {
    const validator = useMemo(() => new Validator(), []);
    const [issueValue, setIssueValue] = useState<Issue | null>(null);
    const [error, setError] = useState<string>('');

    const post = useSelector(getAttachCommentModalContents)
    const [message, setMessage] = useState<string>(post.message);
    const [isMessageValid, setIsMessageValid] = useState<boolean>(true);

    const dispatch = useDispatch();

    const handleCreate = async (e: React.FormEvent<HTMLFormElement> | Event) => {
        e.preventDefault();
        if (!validator.validate() || !message) {
            setIsMessageValid(Boolean(message));
            return;
        }

        const comment = {
            project_id: issueValue?.project_id,
            iid: issueValue?.iid,
            comment: message,
            post_id: post.id,
            web_url: issueValue?.web_url,
        };

        setIsSubmitting(true)

        const created = await attachCommentToIssue(comment)(dispatch);
        if (created.error) {
            const errMessage = getErrorMessage((created as { error: ErrorType }).error.message);
            setError(errMessage)
            setIsSubmitting(false)
            return;
        }

        handleClose();
    };

    const handleIssueValueChange = (newValue: Issue | null) => {
        setIssueValue(newValue);
    };

    const handleMessageChange = (newValue: string) => {
        setMessage(newValue)
        if (newValue && !isMessageValid) {
            setIsMessageValid(true);
        }
    }

    const messageValidationError = (!isMessageValid) ? (
        <p className='help-text error-text'>
            <span>{'This field is required.'}</span>
        </p>
    ) : null;

    const style = getStyle(theme);

    return (
        <form
            role='form'
            onSubmit={handleCreate}
        >
            <Modal.Body
                style={style.modal}
            >
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
                    required={true}
                    value={message}
                    onChange={handleMessageChange}
                />
                {messageValidationError}
            </Modal.Body>
            <Modal.Footer>
                <FormButton
                    btnClass='btn-link'
                    defaultMessage='Cancel'
                    onClick={handleClose}
                />
                <FormButton
                    btnClass='btn btn-primary'
                    saving={isSubmitting}
                    defaultMessage='Attach'
                    savingMessage='Attaching'
                />
            </Modal.Footer>
        </form>
    );
}

const getStyle = (theme: Theme) => ({
    modal: {
        padding: '1em 2.3em 0 2.3em',
        color: theme.centerChannelColor,
        backgroundColor: theme.centerChannelBg,
    },
});

export default AttachCommentToIssueForm;
