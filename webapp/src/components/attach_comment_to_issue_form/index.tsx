import React, {useEffect, useMemo, useState} from 'react';
import {Theme} from 'mattermost-redux/types/preferences';
import {useDispatch, useSelector} from 'react-redux';
import {getPost} from 'mattermost-redux/selectors/entities/posts';

import Input from 'src/components/input';
import Validator from 'src/components/validator';
import {id as pluginId} from 'src/manifest';
import GitlabIssueSelector from 'src/components/gitlab_issue_selector';
import {getErrorMessage} from 'src/utils/user_utils';
import {attachCommentToIssue} from 'src/actions';
import {GlobalState} from 'src/types/global_state';

interface PropTypes {  
    theme: Theme;
    handleClose: () => void;
    setFormSubmission: React.Dispatch<React.SetStateAction<FormSubmission>>;
    formSubmission: FormSubmission;
}

const AttachCommentToIssueForm = ({theme, handleClose, setFormSubmission, formSubmission}: PropTypes) => {
    const validator = useMemo(() => new Validator(), []);
    const [issueValue, setIssueValue] = useState<Issue | null>(null);

    const post = useSelector((state: GlobalState) => {
        const postId = state[`plugins-${pluginId}` as pluginReduxStoreKey].postIdForAttachCommentToIssueModal;
        const post = getPost(state, postId);
    
        return post;
    })

    const dispatch = useDispatch();

    const handleCreate = async () => {       
        if (!validator.validate()) {
            setFormSubmission({
                ...formSubmission,
                isSubmitted: false,
            })
            return;
        }

        const comment = {
            project_id: issueValue?.project_id,
            iid: issueValue?.iid,
            comment: post.message,
            post_id: post.id,
            web_url: issueValue?.web_url,
        };

        setFormSubmission({
            ...formSubmission,
            isSubmitting: true
        })

        const created = await attachCommentToIssue(comment)(dispatch);
        if (created.error) {
            const errMessage = getErrorMessage((created as {error: ErrorType}).error.message);
            setFormSubmission({
                isSubmitted: false,
                isSubmitting: false,
                error: errMessage
            })
            return;
        }

        handleClose();
    };

    useEffect(() => {                    
        if (formSubmission.isSubmitted){
          handleCreate();
        }
      }, [formSubmission.isSubmitted])

    const handleIssueValueChange = (newValue: Issue | null) => {
        setIssueValue(newValue);
    };

    return (
        <div>
            <GitlabIssueSelector
                name={'issue'}
                onChange={handleIssueValueChange}
                required={true}
                theme={theme}
                error={formSubmission.error}
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
}

export default AttachCommentToIssueForm;
