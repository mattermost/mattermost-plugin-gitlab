import React, {useState} from 'react';
import {useDispatch, useSelector} from 'react-redux';
import {Modal} from 'react-bootstrap';
import {Theme} from 'mattermost-redux/types/preferences';

import FormButton from 'src/components/form_button';
import {id as pluginId} from 'src/manifest';
import {closeCreateIssueModal} from 'src/actions';
import {GlobalState} from 'src/types/global_state';
import CreateIssueForm from 'src/components/create_issue_form';

type PropTypes = {
    theme: Theme;
};

const CreateIssueModal = ({theme}: PropTypes) => {
    const [formSubmission, setFormSubmission] = useState<FormSubmission>({
        isSubmitted: false,
        isSubmitting: false,
        error: '',
    });

    const visible = useSelector((state: GlobalState) => state[`plugins-${pluginId}` as plugin].isCreateIssueModalVisible); 
    if (!visible) {
        return null;
    }

    const dispatch = useDispatch();
    const handleClose = () => {
        setFormSubmission({
            isSubmitted: false,
            isSubmitting: false,
            error: ''
        })
        dispatch(closeCreateIssueModal());
    };

    const handleCreate = async (e: React.FormEvent<HTMLFormElement> | Event) => {
        e.preventDefault();
        setFormSubmission({
            ...formSubmission,
            isSubmitted: true,
        })
    };

    const style = getStyle(theme);

    const submitError = formSubmission.error ? (
        <p className='help-text error-text'>
            <span>{formSubmission.error}</span>
        </p>
    ) : null;

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
                    <CreateIssueForm
                        handleClose={handleClose}
                        setFormSubmission={setFormSubmission}
                        formSubmission={formSubmission}
                        theme={theme}
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
                        saving={formSubmission.isSubmitting}
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
