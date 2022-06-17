// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import {Modal} from 'react-bootstrap';
import {Theme} from 'mattermost-redux/types/preferences';
import {Post} from 'mattermost-redux/types/posts';

import FormButton from '../../form_button';
import Input from '../../input';
import Validator from '../../validator';

import GitlabIssueSelector from '../../gitlab_issue_selector';
import {getErrorMessage} from '../../../utils/user_utils';

const initialState = {
    submitting: false,
    issueValue: null,
    error: '',
};

interface PropTypes {  
    post: Post,
    theme: Theme,
    visible: boolean,
    actions: {
        close: () => {
            type: string;
        };
        create: (payload: CommentBody) => Promise<{
            error?: ErrorType;
        }>;
    }
}

interface StateTypes {
    submitting: boolean,
    issueValue: Issue | null,
    error: string,
}

export default class AttachCommentToIssueModal extends PureComponent<PropTypes, StateTypes> {
    validator: Validator;

    constructor(props: PropTypes) {
        super(props);
        this.state = initialState;
        this.validator = new Validator();
    }

    handleCreate = async (e: React.FormEvent<HTMLFormElement> | Event) => {
        e.preventDefault();

        if (!this.validator.validate()) {
            return;
        }

        const comment = {
            project_id: this.state.issueValue?.project_id,
            iid: this.state.issueValue?.iid,
            comment: this.props.post.message,
            post_id: this.props.post.id,
            web_url: this.state.issueValue?.web_url,
        };

        this.setState({submitting: true});

        const created = await this.props.actions.create(comment);
        if (created.error) {
            const errMessage = getErrorMessage(created.error.message);
            this.setState({error: errMessage, submitting: false});
            return;
        }

        this.handleClose();
    };

    handleClose = () => {
        this.setState(initialState, this.props.actions.close);
    };

    handleIssueValueChange = (newValue: Issue | null) => {
        this.setState({
            issueValue: newValue,
        });
    };

    render() {
        const {visible, theme} = this.props;
        const {error, submitting} = this.state;
        const style = getStyle(theme);

        if (!visible) {
            return null;
        }

        const component = (
            <div>
                <GitlabIssueSelector
                    name={'issue'}
                    onChange={this.handleIssueValueChange}
                    required={true}
                    theme={theme}
                    error={error}
                    value={this.state.issueValue}
                    addValidate={this.validator.addComponent}
                    removeValidate={this.validator.removeComponent}
                />
                <Input
                    id={'comment'}
                    label='Message Attached to GitLab Issue'
                    type='textarea'
                    required={false}
                    value={this.props.post.message}
                    disabled={false}
                    readOnly={true}
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
                        {'Attach Message to GitLab Issue'}
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
                        <FormButton
                            btnClass='btn-link'
                            defaultMessage='Cancel'
                            onClick={this.handleClose}
                        />
                        <FormButton
                            btnClass='btn btn-primary'
                            saving={submitting}
                            defaultMessage='Attach'
                            savingMessage='Attaching'
                        >
                            {'Attach'}
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
    descriptionArea: {
        height: 'auto',
        width: '100%',
        color: '#000',
    },
});
