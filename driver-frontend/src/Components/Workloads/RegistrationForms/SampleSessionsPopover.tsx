import { Popover } from '@patternfly/react-core';
import HelpIcon from '@patternfly/react-icons/dist/esm/icons/help-icon';
import styles from '@patternfly/react-styles/css/components/Form/form';
import React from 'react';

const SampleSessionsPopover: React.FunctionComponent = () => {
    return (
        <Popover
            aria-label="sample-sessions-percentage-header"
            headerContent={<div>Sample Sessions %</div>}
            bodyContent={
                <div>
                    SampleSessionsPercent is the percent of sessions from a CSV workload for which we will actually
                    process events. If SampleSessionsPercent is set to 1.0, then all sessions will be processed.
                    SampleSessionsPercent must be strictly greater than 0.
                </div>
            }
        >
            <button
                type="button"
                aria-label="Set the Sample Sessions %."
                onClick={(e) => e.preventDefault()}
                className={styles.formGroupLabelHelp}
            >
                <HelpIcon />
            </button>
        </Popover>
    );
};

export default SampleSessionsPopover;
