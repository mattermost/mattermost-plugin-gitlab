// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {useCallback, useEffect, useState} from 'react';
import {useDispatch, useSelector} from 'react-redux';
import {Theme} from 'mattermost-redux/types/preferences';

import {getProjects} from 'src/actions';
import {id as pluginId} from 'src/manifest';
import ReactSelectSetting from 'src/components/react_select_setting';
import {GlobalState} from 'src/types/global_state';

type PropTypes = {
    theme: Theme;
    required: boolean;
    onChange: (project: ProjectSelection) => void;
    value?: string;
    addValidate: (key: string, validateField: () => boolean) => void;
    removeValidate: (key: string) => void;
};

const GitlabProjectSelector = (props: PropTypes) => { 
    const [isLoading, setIsLoading] = useState(false);

    const {yourProjects} = useSelector((state: GlobalState) => {
        return {
            yourProjects: state[`plugins-${pluginId}` as plugin].yourProjects,
        };
    });

    const dispatch = useDispatch();

    useEffect(() => {
      loadProjects();
    }, []);

    const loadProjects = useCallback(async () => {
        setIsLoading(true);
        await dispatch(getProjects());
        setIsLoading(false);
    }, []);

    const onChange = (_: string, name: string) => {
        const project = yourProjects.find((p: Project) => p.path_with_namespace === name);
        props.onChange({name, project_id: project?.id});
    }

    const projectOptions = yourProjects.map((item: Project) => ({value: item.path_with_namespace, label: item.path_with_namespace}));
    
    return (
        <div className={'form-group margin-bottom x3'}>
            <ReactSelectSetting
                name={'project'}
                label={'Project'}
                limitOptions={true}
                required={true}
                onChange={onChange}
                options={projectOptions}
                key={'project'}
                isLoading={isLoading}
                theme={props.theme}
                addValidate={props.addValidate}
                removeValidate={props.removeValidate}
                value={projectOptions.find((option: SelectionType) => option.value === props.value)}
            />
            <div className={'help-text'}>
                {'Returns GitLab projects connected to the user account'}
            </div>
        </div>
    );
}

export default GitlabProjectSelector;
