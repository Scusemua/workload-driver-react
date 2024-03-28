import * as React from 'react';
import {
    Brand,
    Masthead,
    MastheadBrand,
    MastheadContent,
    MastheadMain,
    Page,
    SkipToContent,
    ToggleGroup,
    ToggleGroupItem,
} from '@patternfly/react-core';
import logo from '@app/bgimages/WorkloadDriver-Logo.svg';
import { DarkModeContext } from '@app/Providers/DarkModeProvider';
import { MoonIcon, SunIcon } from '@patternfly/react-icons';

interface IAppLayout {
    children: React.ReactNode;
}

const AppLayout: React.FunctionComponent<IAppLayout> = ({ children }) => {
    const lightModeToggleId: string = 'theme-toggle-lightmode';
    const darkModeToggleId: string = 'theme-toggle-darkmode';
    const { darkMode, toggleDarkMode } = React.useContext(DarkModeContext);
    const [isSelected, setIsSelected] = React.useState(darkMode ? darkModeToggleId : lightModeToggleId);

    const handleThemeToggleClick = (event) => {
        const id = event.currentTarget.id;
        setIsSelected(id);

        if ((id === lightModeToggleId && darkMode) || (id == darkModeToggleId && !darkMode)) {
            toggleDarkMode();
        }
    };

    const Header = (
        <Masthead>
            <MastheadMain>
                <MastheadBrand>
                    <Brand src={logo} alt="Workload Driver Logo" heights={{ default: '36px' }} />
                </MastheadBrand>
                <MastheadContent>
                    <div className="pf-v5-theme-dark">
                        <ToggleGroup>
                            <ToggleGroupItem
                                aria-label="theme-toggle-lightmode"
                                id="theme-toggle-lightmode"
                                buttonId="theme-toggle-lightmode"
                                icon={<SunIcon />}
                                onChange={handleThemeToggleClick}
                                isSelected={isSelected === 'theme-toggle-lightmode'}
                            />
                            <ToggleGroupItem
                                aria-label="theme-toggle-darkmode"
                                id="theme-toggle-darkmode"
                                buttonId="theme-toggle-darkmode"
                                icon={<MoonIcon />}
                                onChange={handleThemeToggleClick}
                                isSelected={isSelected === 'theme-toggle-darkmode'}
                            />
                        </ToggleGroup>
                    </div>
                </MastheadContent>
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
