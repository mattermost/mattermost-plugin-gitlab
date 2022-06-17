type OnChangeType = SelectionType | SelectionType[] | null;

type SelectionType = {
    value: number | string;
    label: string;
}

type ErrorType = {
    message: string;
}

type plugin = "plugin"
