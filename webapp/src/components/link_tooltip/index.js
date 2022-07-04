import {connect} from 'react-redux';

import {id} from '../../manifest';

import {LinkTooltip} from './link_tooltip.jsx';

const mapStateToProps = (state) => {
    return {
        connected: state[`plugins-${id}`].connected,
    };
};

export default connect(mapStateToProps, null)(LinkTooltip);
