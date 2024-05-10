import * as React from 'react';
import { Page, SkipToContent } from '@patternfly/react-core';

import { AppHeader } from './AppHeader';

interface IAppLayout {
    children: React.ReactNode;
}

const AppLayout: React.FunctionComponent<IAppLayout> = ({ children }) => {
    const pageId = 'primary-app-container';

    const PageSkipToContent = (
        <SkipToContent
            onClick={(event) => {
                event.preventDefault();
                const primaryContentContainer = document.getElementById(pageId);
                primaryContentContainer && primaryContentContainer.focus();
            }}
            href={`#${pageId}`}
        >
            Skip to Content
        </SkipToContent>
    );
    return (
        <Page mainContainerId={pageId} header={<AppHeader />} skipToContent={PageSkipToContent}>
            {children}
        </Page>
    );
};

export { AppLayout };
