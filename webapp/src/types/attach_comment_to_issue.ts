export interface Issue {
    iid: number;
    web_url: string;
    project_id: number;
}

export interface IssueSelection {
    value: Issue;
    label: string;
}
