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
