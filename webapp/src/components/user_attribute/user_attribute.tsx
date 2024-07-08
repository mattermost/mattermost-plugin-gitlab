import React, {useEffect, CSSProperties} from 'react';

interface UserAttributeProps {
    id: string;
    username?: string;
    gitlabURL?: string;
    actions: {
        getGitlabUser: (id: string) => void;
    };
}

const UserAttribute = ({id, username, gitlabURL, actions}: UserAttributeProps) => {
    useEffect(() => {
        actions.getGitlabUser(id);
    }, [id, actions]);

    if (!username || !gitlabURL) {
        return null;
    }

    return (
        <div style={style.container}>
            <a
                href={`${gitlabURL}/${username}`}
                target='_blank'
                rel='noopener noreferrer'
            >
                <i className='fa fa-gitlab'/>{' ' + username}
            </a>
        </div>
    );
};

const style: {container: CSSProperties} = {
    container: {
        margin: '5px 0',
    },
};

export default UserAttribute;
