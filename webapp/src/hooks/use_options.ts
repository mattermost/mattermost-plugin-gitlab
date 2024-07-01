import {useDispatch} from 'react-redux';

type useOptionsProps = {
    projectName: string;
    getOptions: FetchIssueAttributeOptionsForProject<AttributeType>;
    returnType: [string, string];
    errorMessage: string;
    projectID?: number;
}

export const useOptions = ({projectName, getOptions, returnType, errorMessage, projectID}: useOptionsProps) => {
    const dispatch = useDispatch();

    const loadOptions = async () => {
        if (!projectName) {
            return [];
        }

        const options = await getOptions(projectID)(dispatch);
        if (options?.error) {
            throw new Error(errorMessage);
        }

        if (!options || !options.data) {
            return [];
        }

        return options.data.map((option: any) => ({
            value: option[returnType[0]],
            label: option[returnType[1]],
        }));
    };
    return loadOptions;
};
