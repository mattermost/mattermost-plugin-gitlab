import React, {useEffect, useMemo, useState} from 'react';
import {useDispatch, useSelector} from 'react-redux';
import {Modal} from 'react-bootstrap';
import {Theme} from 'mattermost-redux/types/preferences';

import {getErrorMessage} from 'src/utils/user_utils';
import GitlabLabelSelector from 'src/components/gitlab_label_selector';
import GitlabAssigneeSelector from 'src/components/gitlab_assignee_selector';
import GitlabMilestoneSelector from 'src/components/gitlab_milestone_selector';
import GitlabProjectSelector from 'src/components/gitlab_project_selector';
import Validator from 'src/components/validator';
import Input from 'src/components/input';
import {createIssue} from 'src/actions';
import {getCreateIssueModalContents} from 'src/selectors';
import FormButton from '../form_button';

const MAX_TITLE_LENGTH = 255;

type PropTypes = {
    theme: Theme;
    handleClose: () => void;
    setIsSubmitting: React.Dispatch<React.SetStateAction<boolean>>;
    isSubmitting: boolean;
};

const CreateIssueForm = ({theme, handleClose, isSubmitting, setIsSubmitting}: PropTypes) => {    
    const validator = useMemo(() => (new Validator()), []);
    const [project, setProject] = useState<ProjectSelection | null>(null);
    const [issueTitle, setIssueTitle] = useState<string>('');
    const [issueDescription, setIssueDescription] = useState<string>('');
    const [labels, setLabels] = useState<SelectionType[]>([]);
    const [assignees, setAssignees] = useState<SelectionType[]>([]);
    const [milestone, setMilestone] = useState<SelectionType | null>(null);
    const [showErrors, setShowErrors] = useState<boolean>(false);
    const [error, setError] = useState<string>('');
    const [issueTitleValid, setIssueTitleValid] = useState<boolean>(true);

    const {post, channelId, title} = useSelector(getCreateIssueModalContents);

    useEffect(() => {
        if (post) {
            setIssueDescription(post.message);
        } else if (channelId) {
            setIssueTitle(title.substring(0, MAX_TITLE_LENGTH));
        }
    }, []);

    const dispatch = useDispatch();

    // handle issue creation after form is populated
    const handleCreate = async (e: React.FormEvent<HTMLFormElement> | Event) => {   
        e.preventDefault();     
        if (!validator.validate() || !issueTitle || !project?.project_id) {            
            setIssueTitleValid(Boolean(issueTitle));
            setShowErrors(true);
            return;
        }

        const postId = post?.id ?? '';

        const issue: IssueBody = {
            title: issueTitle,
            description: issueDescription,
            project_id: project.project_id,
            labels: labels.map((label) => label.value),
            assignees: assignees.map((assignee) => assignee.value),
            milestone: milestone?.value,
            post_id: postId,
            channel_id: channelId,
        };

        setIsSubmitting(true)

        const created = await createIssue(issue)(dispatch);
        if (created.error) {
            const errMessage = getErrorMessage((created as {error: ErrorType}).error.message);
            setShowErrors(true);
            setIsSubmitting(false);
            setError(errMessage);
            return;
        }

        handleClose();
    };

    const handleProjectChange = (project: ProjectSelection | null) => setProject(project);

    const handleLabelsChange = (newLabels: OnChangeType) => setLabels(newLabels as SelectionType[]);

    const handleAssigneesChange = (newAssignees: OnChangeType) => setAssignees(newAssignees as SelectionType[]);

    const handleMilestoneChange = (newMilestone: OnChangeType) => setMilestone(newMilestone as SelectionType);

    const handleIssueTitleChange = (issueTitle: string) => {        
        setIssueTitle(issueTitle);
        if (issueTitle && !issueTitleValid) {
            setIssueTitleValid(true);
        }
    };

    const handleIssueDescriptionChange = (issueDescription: string) => setIssueDescription(issueDescription);

    const issueAttributeSelectors = () => {              
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
    };
    
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

    const style = getStyle(theme);

    return (
        <form
            role='form'
            onSubmit={handleCreate}
        >
            <Modal.Body
                style={style.modal}
            >
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
                {issueAttributeSelectors()}
                <Input
                    id={'description'}
                    required={false}
                    label='Issue description'
                    type='textarea'
                    value={issueDescription}
                    onChange={handleIssueDescriptionChange}
                />
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
                    saving={isSubmitting}
                    defaultMessage='Submit'
                    savingMessage='Submitting'
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

export default CreateIssueForm;
