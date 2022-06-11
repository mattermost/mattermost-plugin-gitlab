// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';
import {Modal} from 'react-bootstrap';

import FormButton from 'components/form_button';
import Input from 'components/input';
import Validator from 'components/validator';

import GitlabIssueSelector from 'components/gitlab_issue_selector';
import {getErrorMessage} from 'utils/user_utils';

const initialState = {
    submitting: false,
    issueValue: null,
    error: null,
};

export default class AttachCommentToIssueModal extends PureComponent {
    static propTypes = {
        close: PropTypes.func.isRequired,
        create: PropTypes.func.isRequired,
        post: PropTypes.object,
        theme: PropTypes.object.isRequired,
        visible: PropTypes.bool.isRequired,
    };

    constructor(props) {
        super(props);
        this.state = initialState;
        this.validator = new Validator();
    }

    handleCreate = async (e) => {
        console.log(this.props.post);
        e.preventDefault();

        if (!this.validator.validate()) {
            return;
        }

        const issue = {
            // project_id: this.state.issueValue.project_id,
            project_id: 36353273,
            // iid: this.state.issueValue.iid,
            iid: 31,
            comment: this.props.post.message,
            post_id: this.props.post.id,
            // web_url: this.state.issueValue.web_url,
            web_url: "https://gitlab.com/raghavaggarwal2308/Calculator/-/issues/31",
        };

        this.setState({submitting: true});

        const created = await this.props.create(issue);
        if (created.error) {
            const errMessage = getErrorMessage(created.error.message);
            this.setState({error: errMessage, submitting: false});
            return;
        }

        this.handleClose(e);
    };

    handleClose = (e) => {
        e.preventDefault();
        const {close} = this.props;
        this.setState(initialState, close);
    };

    handleIssueValueChange = (newValue) => {
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
                    id={'issue'}
                    onChange={this.handleIssueValueChange}
                    required={true}
                    theme={theme}
                    error={error}
                    value={this.state.issueValue}
                    addValidate={this.validator.addComponent}
                    removeValidate={this.validator.removeComponent}
                />
                <Input
                    label='Message Attached to GitLab Issue'
                    type='textarea'
                    isDisabled={true}
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
                        ref='modalBody'
                    >
                        {component}
                    </Modal.Body>
                    <Modal.Footer>
                        <FormButton
                            type='button'
                            btnClass='btn-link'
                            defaultMessage='Cancel'
                            onClick={this.handleClose}
                        />
                        <FormButton
                            type='submit'
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

const getStyle = (theme) => ({
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
