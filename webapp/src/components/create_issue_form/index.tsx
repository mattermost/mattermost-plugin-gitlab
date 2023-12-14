import React, {useEffect, useMemo, useState} from 'react';
import {useDispatch, useSelector} from 'react-redux';
import {Theme} from 'mattermost-redux/types/preferences';
import {getPost} from 'mattermost-redux/selectors/entities/posts';

import {getErrorMessage} from 'src/utils/user_utils';
import GitlabLabelSelector from 'src/components/gitlab_label_selector';
import GitlabAssigneeSelector from 'src/components/gitlab_assignee_selector';
import GitlabMilestoneSelector from 'src/components/gitlab_milestone_selector';
import GitlabProjectSelector from 'src/components/gitlab_project_selector';
import Validator from 'src/components/validator';
import Input from 'src/components/input';
import {id as pluginId} from 'src/manifest';
import {createIssue} from 'src/actions';
import {GlobalState} from 'src/types/global_state';
import {usePrevious} from 'src/hooks/use_previous';

const MAX_TITLE_LENGTH = 255;

type PropTypes = {
    theme: Theme;
    handleClose: () => void;
    setFormSubmission: React.Dispatch<React.SetStateAction<FormSubmission>>;
    formSubmission: FormSubmission;
};

const CreateIssueForm = ({theme, handleClose, formSubmission, setFormSubmission}: PropTypes) => {    
    const validator = useMemo(() => (new Validator()), []);
    const [project, setProject] = useState<ProjectSelection | null>(null);
    const [issueTitle, setIssueTitle] = useState<string>('');
    const [issueDescription, setIssueDescription] = useState<string>('');
    const [labels, setLabels] = useState<SelectionType[]>([]);
    const [assignees, setAssignees] = useState<SelectionType[]>([]);
    const [milestone, setMilestone] = useState<SelectionType | null>(null);
    const [showErrors, setShowErrors] = useState<boolean>(false);
    const [issueTitleValid, setIssueTitleValid] = useState<boolean>(true);

    const {post, channelId, title} = useSelector((state: GlobalState) => {
        const {postId, title, channelId} = state[`plugins-${pluginId}` as plugin].createIssueModal;
        
        const post = postId ? getPost(state, postId) : null;
        return {
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
    const handleCreate = async () => {        
        if (!validator.validate() || !issueTitle) {            
            setIssueTitleValid(Boolean(issueTitle));
            setShowErrors(true);
            setFormSubmission({
                ...formSubmission,
                isSubmitted: false,
            })
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

        setFormSubmission({
            ...formSubmission,
            isSubmitting: true
        })

        const created = await createIssue(issue)(dispatch);
        if (created.error) {
            const errMessage = getErrorMessage((created as {error: ErrorType}).error.message);
            setShowErrors(true);
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

    return (
        <>
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
        </>
    );
}

export default CreateIssueForm;
