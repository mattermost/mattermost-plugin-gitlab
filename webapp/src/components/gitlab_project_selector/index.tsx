// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useEffect, useState} from 'react';
import {useDispatch, useSelector} from 'react-redux';
import {Theme} from 'mattermost-redux/types/preferences';

import {getProjects} from 'src/actions';
import {id as pluginId} from 'src/manifest';
import ReactSelectSetting from 'src/components/react_select_setting';
import {GlobalState} from 'src/types/global_state';

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

    const {yourProjects} = useSelector((state: GlobalState) => ({
        yourProjects: state[`plugins-${pluginId}` as pluginReduxStoreKey].yourProjects,
    }));

    const dispatch = useDispatch();

    useEffect(() => {
        loadProjects();
    }, []);

    const loadProjects = async () => {
        setIsLoading(true);
        await dispatch(getProjects());
        setIsLoading(false);
    };

    const handleOnChange = (_: string, name: string) => {
        const project = yourProjects.find((p: Project) => p.path_with_namespace === name);
        onChange(project ? {name, project_id: project?.id} : null);
    };

    const projectOptions = yourProjects.map((item: Project) => ({value: item.path_with_namespace, label: item.path_with_namespace}));

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
        </div>
    );
};

export default GitlabProjectSelector;
