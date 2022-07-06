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
