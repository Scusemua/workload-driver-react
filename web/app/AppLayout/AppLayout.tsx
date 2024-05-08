import * as React from 'react';
import {
    Brand,
    Button,
    Flex,
    FlexItem,
    Masthead,
    MastheadBrand,
    MastheadContent,
    MastheadMain,
    Page,
    SkipToContent,
    ToggleGroup,
    ToggleGroupItem,
    Tooltip,
} from '@patternfly/react-core';
import logo from '@app/bgimages/WorkloadDriver-Logo.svg';
import { DarkModeContext } from '@app/Providers/DarkModeProvider';
import { MoonIcon, SkullCrossbonesIcon, SunIcon } from '@patternfly/react-icons';

interface IAppLayout {
    children: React.ReactNode;
}

const AppLayout: React.FunctionComponent<IAppLayout> = ({ children }) => {
    const lightModeId: string = 'theme-toggle-lightmode';
    const darkModeId: string = 'theme-toggle-darkmode';
    const lightModeButtonId: string = lightModeId + '-button';
    const darkModeButtonId: string = darkModeId + '-button';
    const { darkMode, toggleDarkMode } = React.useContext(DarkModeContext);
    const [isSelected, setIsSelected] = React.useState(darkMode ? darkModeButtonId : lightModeButtonId);

    const handleThemeToggleClick = (event) => {
        const id = event.currentTarget.id;
        setIsSelected(id);

        if ((id === lightModeButtonId && darkMode) || (id == darkModeButtonId && !darkMode)) {
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
                    <Flex direction={{ default: 'row' }}>
                        <FlexItem>
                            <div className="pf-v5-theme-dark">
                                <ToggleGroup>
                                    <ToggleGroupItem
                                        aria-label={lightModeId}
                                        id={lightModeId}
                                        buttonId={lightModeButtonId}
                                        icon={<SunIcon />}
                                        onChange={handleThemeToggleClick}
                                        isSelected={isSelected === lightModeButtonId}
                                    />
                                    <ToggleGroupItem
                                        aria-label={darkModeId}
                                        id={darkModeId}
                                        buttonId={darkModeButtonId}
                                        icon={<MoonIcon />}
                                        onChange={handleThemeToggleClick}
                                        isSelected={isSelected === darkModeButtonId}
                                    />
                                </ToggleGroup>
                            </div>
                        </FlexItem>

                        <FlexItem>
                            <Tooltip content="Cause the Cluster Gateway to panic.">
                                <Button
                                    isDanger
                                    variant="secondary"
                                    icon={<SkullCrossbonesIcon />}
                                    onClick={() => {
                                        const requestOptions = {
                                            method: 'POST',
                                        };

                                        fetch('api/panic', requestOptions);
                                    }}
                                >
                                    Induce a Panic
                                </Button>
                            </Tooltip>
                        </FlexItem>

                        <FlexItem>
                            <Tooltip content="Prompt the server to broadcast a fake error for testing/debugging purposes.">
                                <Button
                                    isDanger
                                    variant="secondary"
                                    icon={<SkullCrossbonesIcon />}
                                    onClick={() => {
                                        const requestOptions = {
                                            method: 'POST',
                                        };

                                        fetch('api/spoof-error', requestOptions);
                                    }}
                                >
                                    Generate Fake Error Message
                                </Button>
                            </Tooltip>
                        </FlexItem>
                    </Flex>
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
