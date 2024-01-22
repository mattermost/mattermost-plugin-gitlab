interface IssueBody {
    title: string;
    description: string;
    project_id?: number;
    labels?: (string | number)[];
    assignees?: (string | number)[];
    milestone?: (string | number);
    post_id: string;
    channel_id: string;
}

interface Issue {
    iid: number;
    web_url: string;
    project_id: number;
}

interface IssueSelection {
    value: Issue;
    label: string;
}

interface CommentBody {
    project_id?: number;
    iid?: number;
    comment: string;
    post_id: string;
    web_url?: string;
}

interface Assignee {
    id: number;
    username: string;
}

interface Label{
    name: string;
}

interface Milestone{
    id: number;
    title: string;
}

interface ProjectSelection {
    name: string;
    project_id?: number;
}

interface Project{
    path_with_namespace: string;
    id: number;
}
