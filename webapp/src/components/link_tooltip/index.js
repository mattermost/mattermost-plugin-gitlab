import {connect} from 'react-redux';

import manifest from '../../manifest';

import {LinkTooltip} from './link_tooltip.jsx';

const mapStateToProps = (state) => {
    const {id} = manifest;
    return {
        connected: state[`plugins-${id}`].connected,
        connectedGitlabUrl: state[`plugins-${id}`].gitlabURL,
    };
};

export default connect(mapStateToProps, null)(LinkTooltip);
