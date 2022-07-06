type OnChangeType = SelectionType | SelectionType[] | null;

type SelectionType = {
    value: number | string;
    label: string;
}

type ErrorType = {
    message: string;
}

type plugin = 'plugins-com.github.manland.mattermost-plugin-gitlab'

type GetOptions = (projectID?: number) => (dispatch: Dispatch<GenericAction>) => Promise<{
    error?: ErrorType;
    data?: Assignee[] | Milestone[] | Label[];
}>
