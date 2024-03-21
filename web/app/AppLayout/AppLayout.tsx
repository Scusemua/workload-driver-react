import * as React from 'react';
import { Brand, Masthead, MastheadBrand, MastheadMain, Page, SkipToContent } from '@patternfly/react-core';
import logo from '@app/bgimages/WorkloadDriver-Logo.svg';

interface IAppLayout {
    children: React.ReactNode;
}

const AppLayout: React.FunctionComponent<IAppLayout> = ({ children }) => {
    const Header = (
        <Masthead>
            <MastheadMain>
                <MastheadBrand>
                    <Brand src={logo} alt="Workload Driver Logo" heights={{ default: '36px' }} />
                </MastheadBrand>
            </MastheadMain>
        </Masthead>
    );

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
        <Page mainContainerId={pageId} header={Header} skipToContent={PageSkipToContent}>
            {children}
        </Page>
    );
};

export { AppLayout };
