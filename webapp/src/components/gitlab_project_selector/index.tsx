// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useMemo, useState} from 'react';
import {useDispatch, useSelector} from 'react-redux';
import {Theme} from 'mattermost-redux/types/preferences';

import {getProjects} from 'src/actions';
import ReactSelectSetting from 'src/components/react_select_setting';
import {getYourProjects} from 'src/selectors';
import {getErrorMessage} from 'src/utils/user_utils';
import {Project, ProjectSelection} from 'src/types/gitlab_types';
import {ErrorType, SelectionType} from 'src/types/common';

type PropTypes = {
    theme: Theme;
    required: boolean;
    onChange: (project: ProjectSelection | null) => void;
    value?: string;
    addValidate: (key: string, validateField: () => boolean) => void;
    removeValidate: (key: string) => void;
};

const GitlabProjectSelector = ({theme, required, onChange, value, addValidate, removeValidate}: PropTypes) => {
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string>('');

    const myProjects = useSelector(getYourProjects);

    const dispatch = useDispatch();

    useEffect(() => {
        loadProjects();
    }, []);

    const loadProjects = async () => {
        setIsLoading(true);
        const res = (await dispatch(getProjects())) as {error?: ErrorType};
        if (res.error) {
            const errMessage = getErrorMessage(res.error?.message);
            setError(errMessage);
        } else {
            setError('');
        }

        setIsLoading(false);
    };

    const handleOnChange = (_: string, name: string) => {
        const project = myProjects.find((p: Project) => p.path_with_namespace === name);
        onChange(project ? {name, project_id: project?.id} : null);
    };

    const projectOptions = useMemo(() => {
        return myProjects.map((item: Project) => ({value: item.path_with_namespace, label: item.path_with_namespace}));
    }, [myProjects]);

    return (
        <div className={'form-group margin-bottom x3'}>
            <ReactSelectSetting
                name={'project'}
                label={'Project'}
                limitOptions={true}
                required={required}
                onChange={handleOnChange}
                options={projectOptions}
                key={'project'}
                isLoading={isLoading}
                theme={theme}
                addValidate={addValidate}
                removeValidate={removeValidate}
                value={projectOptions.find((option: SelectionType) => option.value === value)}
            />
            <div className={'help-text'}>
                {'Returns GitLab projects connected to the user account'}
            </div>
            {error && (
                <p
                    className='alert alert-danger'
                    style={{marginTop: '10px'}}
                >
                    <i
                        className='fa fa-warning'
                        title='Warning Icon'
                    />
                    <span> {error}</span>
                </p>
            )}
        </div>
    );
};

export default GitlabProjectSelector;
