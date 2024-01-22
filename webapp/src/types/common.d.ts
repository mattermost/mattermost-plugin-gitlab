type OnChangeType = SelectionType | SelectionType[] | null;

type SelectionType = {
    value: number | string;
    label: string;
}

type ErrorType = {
    message: string;
}

type pluginReduxStoreKey = 'plugins-com.github.manland.mattermost-plugin-gitlab'

type FetchIssueAttributeOptionsForProject = (projectID?: number) => (dispatch: Dispatch<GenericAction>) => Promise<{
    error?: ErrorType;
    data?: Assignee[] | Milestone[] | Label[];
}>
