import React, {PureComponent} from 'react';
import {Modal} from 'react-bootstrap';
import {Theme} from 'mattermost-redux/types/preferences';

import GitlabLabelSelector from '../../gitlab_label_selector';
import GitlabAssigneeSelector from '../../gitlab_assignee_selector';
import GitlabMilestoneSelector from '../../gitlab_milestone_selector';
import GitlabProjectSelector from '../../gitlab_project_selector';
import Validator from '../../validator';
import FormButton from '../../form_button';
import Input from '../../input';
import {getErrorMessage} from '../../../utils/user_utils';
import {Post} from 'mattermost-redux/types/posts';
import {LabelSelection} from 'src/types/gitlab_label_selector';
import {AssigneeSelection} from 'src/types/gitlab_assignee_selector';
import {MilestoneSelection} from 'src/types/gitlab_milestone_selector';
import {ProjectSelection as Project } from 'src/types/gitlab_project_selector';

const MAX_TITLE_LENGTH = 256;

interface PropTypes {
    post: Post | null,
    title: string,
    channelId: string,
    theme: Theme,
    visible: boolean,
    actions: {
        close: () => {
            type: string;
        };
        create: (payload: any) => Promise<{
            error: any;
            data?: undefined;
        } | {
            data: any;
            error?: undefined;
        }>;
    }
};

interface StateTypes {
    submitting: boolean;
    error: string | null;
    project: Project | null;
    issueTitle: string;
    issueDescription: string;
    labels: LabelSelection[];
    assignees: AssigneeSelection[];
    milestone: null | MilestoneSelection;
    showErrors: boolean;
    issueTitleValid: boolean;
}

const initialState = {
    submitting: false,
    error: null,
    project: null,
    issueTitle: '',
    issueDescription: '',
    labels: [],
    assignees: [],
    milestone: null,
    showErrors: false,
    issueTitleValid: true,
};

export default class CreateIssueModal extends PureComponent<PropTypes, StateTypes> {
    validator: Validator

    constructor(props: PropTypes) {
        super(props);
        this.state = initialState;
        this.validator = new Validator();
    }

    componentDidUpdate(prevProps: PropTypes) {
        if (this.props.post && !prevProps.post) {
            this.setState({issueDescription: this.props.post.message});
        } else if (this.props.channelId && (this.props.channelId !== prevProps.channelId || this.props.title !== prevProps.title)) {
            const title = this.props.title.substring(0, MAX_TITLE_LENGTH);
            this.setState({issueTitle: title});
        }
    }

    // handle issue creation after form is populated
    handleCreate = async (e: React.FormEvent<HTMLFormElement> | Event) => {
        e.preventDefault();

        if (!this.validator.validate() || !this.state.issueTitle) {
            this.setState({
                issueTitleValid: Boolean(this.state.issueTitle),
                showErrors: true,
            });
            return;
        }

        const {post} = this.props;
        const postId = post?.id ?? '';

        const issue = {
            title: this.state.issueTitle,
            description: this.state.issueDescription,
            project_id: this.state.project?.project_id,
            labels: this.state.labels.map((label) => label.value),
            assignees: this.state.assignees.map((assignee) => assignee.value),
            milestone: this.state.milestone?.value,
            post_id: postId,
            channel_id: this.props.channelId,
        };

        this.setState({submitting: true});

        const created = await this.props.actions.create(issue);
        if (created.error) {
            const errMessage = getErrorMessage(created.error.message);
            this.setState({
                error: errMessage,
                showErrors: true,
                submitting: false,
            });
            return;
        }  

        this.handleClose();
    };

    handleClose = () => {this.setState(initialState, this.props.actions.close);};

    handleProjectChange = (project: Project) => this.setState({project});

    handleLabelsChange = (labels: LabelSelection[]) => this.setState({labels});

    handleAssigneesChange = (assignees: AssigneeSelection[]) => this.setState({assignees});

    handleMilestoneChange = (milestone: MilestoneSelection) => this.setState({milestone});

    handleIssueTitleChange = (issueTitle: string) => {
        this.setState({issueTitle});
        if (issueTitle && !this.state.issueTitleValid) {
            this.setState({issueTitleValid: true});
        }
    };

    handleIssueDescriptionChange = (issueDescription: string) => this.setState({issueDescription});

    renderIssueAttributeSelectors = () => {
        if (!this.state.project) {
            return null;
        }

        const dropdownProps = {
            projectID: this.state.project.project_id,
            projectName: this.state.project.name,
            theme: this.props.theme,
        }

        const labelProps = {
            selectedLabels: this.state.labels,
            onChange: this.handleLabelsChange,
        }

        const assigneeProps = {
            selectedAssignees: this.state.assignees,
            onChange: this.handleAssigneesChange,
        }

        const milestoneProps = {
            selectedMilestone: this.state.milestone,
            onChange: this.handleMilestoneChange,
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
    }

    render() {
        if (!this.props.visible) {
            return null;
        }

        const theme = this.props.theme;
        const {error, submitting} = this.state;
        const style = getStyle(theme);

        const requiredMsg = 'This field is required.';
        const issueTitleValidationError = (this.state.showErrors && !this.state.issueTitleValid) ? (
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
                    onChange={this.handleProjectChange}
                    value={this.state.project?.name}
                    required={true}
                    theme={theme}
                    addValidate={this.validator.addComponent}
                    removeValidate={this.validator.removeComponent}
                />

                <Input
                    id={'title'}
                    label='Issue title'
                    type='input'
                    required={true}
                    disabled={false}
                    maxLength={MAX_TITLE_LENGTH}
                    value={this.state.issueTitle}
                    onChange={this.handleIssueTitleChange}
                />
                {issueTitleValidationError}

                {this.renderIssueAttributeSelectors()}

                <Input
                    id={'description'}
                    required={false}
                    label='Issue description'
                    type='textarea'
                    value={this.state.issueDescription}
                    onChange={this.handleIssueDescriptionChange}
                />
            </div>
        );

        return (
            <Modal
                dialogClassName='modal--scroll'
                show={true}
                onHide={this.handleClose}
                onExited={this.handleClose}
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
                    onSubmit={this.handleCreate}
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
                            onClick={this.handleClose}
                        />
                        <FormButton
                            btnClass='btn btn-primary'
                            saving={submitting}
                            defaultMessage='Submit'
                            savingMessage='Submitting'
                        >
                            {'Submit'}
                        </FormButton>
                    </Modal.Footer>
                </form>
            </Modal>
        );
    }
}

const getStyle = (theme: Theme) => ({
    modal: {
        padding: '2em 2em 3em',
        color: theme.centerChannelColor,
        backgroundColor: theme.centerChannelBg,
    },
});
