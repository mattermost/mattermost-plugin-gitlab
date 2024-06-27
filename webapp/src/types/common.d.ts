type OnChangeType = SelectionType | SelectionType[] | null;

type SelectionType = {
    value: number | string | Issue;
    label: string;
}

type ErrorType = {
    message: string;
}

type pluginReduxStoreKey = 'plugins-com.github.manland.mattermost-plugin-gitlab'

type AttributeType = Assignee | Milestone | Label;

type FetchIssueAttributeOptionsForProject<T> = (projectID?: number) => (dispatch: Dispatch<GenericAction>) => Promise<{
    error?: ErrorType;
    data?: T[];
}>

type ReactSelectOption = {
    value: Issue;
    label: string;
}
