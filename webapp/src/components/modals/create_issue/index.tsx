import React, {useCallback, useEffect, useMemo, useState} from 'react';
import {useDispatch, useSelector} from 'react-redux';
import {Modal} from 'react-bootstrap';
import {Theme} from 'mattermost-redux/types/preferences';
import {getPost} from 'mattermost-redux/selectors/entities/posts';

import {getErrorMessage} from 'src/utils/user_utils';
import GitlabLabelSelector from 'src/components/gitlab_label_selector';
import GitlabAssigneeSelector from 'src/components/gitlab_assignee_selector';
import GitlabMilestoneSelector from 'src/components/gitlab_milestone_selector';
import GitlabProjectSelector from 'src/components/gitlab_project_selector';
import Validator from 'src/components/validator';
import FormButton from 'src/components/form_button';
import Input from 'src/components/input';
import {id as pluginId} from 'src/manifest';
import {closeCreateIssueModal, createIssue} from 'src/actions';
import {GlobalState} from 'src/types/global_state';
import {usePrevious} from 'src/hooks/use_previous';

const MAX_TITLE_LENGTH = 256;

type PropTypes = {
    theme: Theme;
};

const CreateIssueModal = ({theme}: PropTypes) => {
    const validator = useMemo(() => (new Validator()), []);
    const [submitting, setSubmitting] = useState<boolean>(false);
    const [error, setError] = useState<string | null>(null);
    const [project, setProject] = useState<ProjectSelection | null>(null);
    const [issueTitle, setIssueTitle] = useState<string>('');
    const [issueDescription, setIssueDescription] = useState<string>('');
    const [labels, setLabels] = useState<SelectionType[]>([]);
    const [assignees, setAssignees] = useState<SelectionType[]>([]);
    const [milestone, setMilestone] = useState<SelectionType | null>(null);
    const [showErrors, setShowErrors] = useState<boolean>(false);
    const [issueTitleValid, setIssueTitleValid] = useState<boolean>(true);

    const {visible, post, channelId, title} = useSelector((state: GlobalState) => {
        const {postId, title, channelId} = state[`plugins-${pluginId}` as plugin].createIssueModal;
        
        const post = postId ? getPost(state, postId) : null;
        return {
            visible: state[`plugins-${pluginId}` as plugin].isCreateIssueModalVisible,
            post,
            title,
            channelId,
        };
    });

    const dispatch = useDispatch();
    const prevPost = usePrevious(post);
    const prevChannelId = usePrevious(channelId);
    const prevTitle = usePrevious(title);

    useEffect(() => {
        if (post && !prevPost) {
            setIssueDescription(post.message);
        } else if (channelId && (channelId !== prevChannelId || title !== prevTitle)) {
            setIssueTitle(title.substring(0, MAX_TITLE_LENGTH));
        }
    }, [channelId, title, post]);

    // handle issue creation after form is populated
    const handleCreate = useCallback(async (e: React.FormEvent<HTMLFormElement> | Event) => {
        e.preventDefault();

        if (!validator.validate() || !issueTitle) {            
            setIssueTitleValid(Boolean(issueTitle));
            setShowErrors(true);
            return;
        }

        const postId = post?.id ?? '';

        const issue = {
            title: issueTitle,
            description: issueDescription,
            project_id: project?.project_id,
            labels: labels.map((label) => label.value),
            assignees: assignees.map((assignee) => assignee.value),
            milestone: milestone?.value,
            post_id: postId,
            channel_id: channelId,
        };

        setSubmitting(true);

        const created = await createIssue(issue)(dispatch);
        if (created.error) {
            const errMessage = getErrorMessage((created as {error: ErrorType}).error.message);
            setError(errMessage);
            setShowErrors(true);
            setSubmitting(false);
            return;
        }

        handleClose();
    }, [issueTitle, issueTitleValid, issueDescription, validator, labels, milestone, assignees, project, channelId]);

    const handleClose = useCallback(() => {
        setError('');
        setSubmitting(false);
        setProject(null);
        setIssueTitle('');
        setIssueDescription('');
        setLabels([]);
        setAssignees([]);
        setMilestone(null);
        setShowErrors(false);
        setIssueTitleValid(true);
        dispatch(closeCreateIssueModal());
    }, []);

    const handleProjectChange = useCallback((project: ProjectSelection) => setProject(project), []);

    const handleLabelsChange = useCallback((newLabels: OnChangeType) => setLabels(newLabels as SelectionType[]), []);

    const handleAssigneesChange = useCallback((newAssignees: OnChangeType) => setAssignees(newAssignees as SelectionType[]), []);

    const handleMilestoneChange = useCallback((newMilestone: OnChangeType) => setMilestone(newMilestone as SelectionType), []);

    const handleIssueTitleChange = useCallback((issueTitle: string) => {        
        setIssueTitle(issueTitle);
        if (issueTitle && !issueTitleValid) {
            setIssueTitleValid(true);
        }
    }, [issueTitleValid]);

    const handleIssueDescriptionChange = useCallback((issueDescription: string) => setIssueDescription(issueDescription), []);

    const issueAttributeSelectors = useMemo(() => {        
        if (!project) {
            return null;
        }

        const dropdownProps = {
            projectID: project.project_id,
            projectName: project.name,
            theme: theme,
        }

        const labelProps = {
            selectedLabels: labels,
            onChange: handleLabelsChange,
        }

        const assigneeProps = {
            selectedAssignees: assignees,
            onChange: handleAssigneesChange,
        }

        const milestoneProps = {
            selectedMilestone: milestone,
            onChange: handleMilestoneChange,
        }

        return (
            <>
                <GitlabLabelSelector
                    {...dropdownProps}
                    {...labelProps}
                />

                <GitlabAssigneeSelector
                    {...dropdownProps}
                    {...assigneeProps}
                />

                <GitlabMilestoneSelector
                    {...dropdownProps}
                    {...milestoneProps}
                />
            </>
        );
    }, [project, milestone, assignees, labels]);

    if (!visible) {
        return null;
    }

    const style = getStyle(theme);

    const requiredMsg = 'This field is required.';
    const issueTitleValidationError = (showErrors && !issueTitleValid) ? (
        <p className='help-text error-text'>
            <span>{requiredMsg}</span>
        </p>
    ) : null;

    const submitError = error ? (
        <p className='help-text error-text'>
            <span>{error}</span>
        </p>
    ) : null;
    
    const component = (
        <div>
            <GitlabProjectSelector
                onChange={handleProjectChange}
                value={project?.name}
                required={true}
                theme={theme}
                addValidate={validator.addComponent}
                removeValidate={validator.removeComponent}
            />
            <Input
                id={'title'}
                label='Issue title'
                type='input'
                required={true}
                disabled={false}
                maxLength={MAX_TITLE_LENGTH}
                value={issueTitle}
                onChange={handleIssueTitleChange}
            />
            {issueTitleValidationError}
            {issueAttributeSelectors}
            <Input
                id={'description'}
                required={false}
                label='Issue description'
                type='textarea'
                value={issueDescription}
                onChange={handleIssueDescriptionChange}
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
                    {'Create GitLab Issue'}
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
                    {submitError}
                    <FormButton
                        btnClass='btn-link'
                        defaultMessage='Cancel'
                        onClick={handleClose}
                    />
                    <FormButton
                        btnClass='btn btn-primary'
                        saving={submitting}
                        defaultMessage='Submit'
                        savingMessage='Submitting'
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

export default CreateIssueModal;
