import {useCallback} from "react";
import {useDispatch} from "react-redux";

export const useOptions = (projectName: string, getOptions: GetOptions, returnFields: string[], errorMessage: string, projectID?: number) => {
    const dispatch = useDispatch();

    const loadOptions = useCallback(async () => {
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
            value: option[returnFields[0]],
            label: option[returnFields[1]],
        }));
    }, [projectID, projectName, errorMessage, returnFields]);
    return loadOptions;
}
