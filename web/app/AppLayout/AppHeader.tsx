import * as React from 'react';
import {
    Alert,
    Brand,
    Button,
    Flex,
    FlexItem,
    Icon,
    Label,
    LabelProps,
    Masthead,
    MastheadBrand,
    MastheadContent,
    MastheadMain,
    ToggleGroup,
    ToggleGroupItem,
    Tooltip,
} from '@patternfly/react-core';
import logo from '@app/bgimages/WorkloadDriver-Logo.svg';
import { InfoAltIcon, MoonIcon, SkullCrossbonesIcon, SunIcon } from '@patternfly/react-icons';

import { DarkModeContext } from '@app/Providers/DarkModeProvider';
import { ErrorCircleOIcon, WarningTriangleIcon } from '@patternfly/react-icons';
import useWebSocket, { ReadyState } from 'react-use-websocket';
import { CheckCircleIcon } from '@patternfly/react-icons';

import toast from 'react-hot-toast';
import { NotificationContext } from '@app/Providers';

const connectionStatuses = {
    [ReadyState.CONNECTING]: 'Connecting to Backend ...',
    [ReadyState.OPEN]: 'Connected to Backend ',
    [ReadyState.CLOSING]: 'Disconnecting from Backend ...',
    [ReadyState.CLOSED]: 'Disconnected from Backend ',
    [ReadyState.UNINSTANTIATED]: 'Backend Connection Uninstantiated',
};

const connectionStatusIcons = {
    [ReadyState.CONNECTING]: (
        <Icon isInProgress={true}>
            <CheckCircleIcon />
        </Icon>
    ),
    [ReadyState.OPEN]: (
        <Icon isInProgress={false}>
            <CheckCircleIcon />
        </Icon>
    ),
    [ReadyState.CLOSING]: (
        <Icon isInProgress={true}>
            <CheckCircleIcon />
        </Icon>
    ),
    [ReadyState.CLOSED]: (
        <Icon isInProgress={false}>
            <ErrorCircleOIcon />
        </Icon>
    ),
    [ReadyState.UNINSTANTIATED]: (
        <Icon isInProgress={false}>
            <WarningTriangleIcon />
        </Icon>
    ),
};

const connectionStatusColors: any = {
    /* Map<ReadyState, LabelProps['color']> */ [ReadyState.CONNECTING]: 'yellow',
    [ReadyState.OPEN]: 'green',
    [ReadyState.CLOSING]: 'yellow',
    [ReadyState.CLOSED]: 'red',
    [ReadyState.UNINSTANTIATED]: 'yellow',
};

export const AppHeader: React.FunctionComponent = () => {
    const lightModeId: string = 'theme-toggle-lightmode';
    const darkModeId: string = 'theme-toggle-darkmode';
    const lightModeButtonId: string = lightModeId + '-button';
    const darkModeButtonId: string = darkModeId + '-button';

    const { darkMode, toggleDarkMode } = React.useContext(DarkModeContext);

    const { addNewNotification } = React.useContext(NotificationContext);

    const [isSelected, setIsSelected] = React.useState(darkMode ? darkModeButtonId : lightModeButtonId);

    const { readyState } = useWebSocket('ws://localhost:8000/ws', {
        onOpen: () => console.log('Connected to backend'),
        onClose: () => console.error('Lost connection to backend'),
        shouldReconnect: () => true,
    });

    React.useEffect(() => {
        switch (readyState) {
            case ReadyState.CLOSED:
                addNewNotification({
                    title: 'Connection Lost to Backend',
                    message: 'The persistent connection with the backend server has been lost.',
                    notificationType: 1,
                    panicked: false,
                });
            // toast.custom(
            //     <Alert variant="warning" title="Connection Lost to Backend">
            //         The persistent connection with the backend server has been lost.
            //     </Alert>,
            // );
            case ReadyState.OPEN:
                addNewNotification({
                    title: 'Connection Established',
                    message: 'The persistent connection with the backend server has been established.',
                    notificationType: 3,
                    panicked: false,
                });
            // toast.custom(
            //     <Alert variant="success" title="Connection Established">
            //         The persistent connection with the backend server has been established.
            //     </Alert>,
            // );
        }
    }, [readyState]);

    const connectionStatus = connectionStatuses[readyState];
    const connectionStatusIcon = connectionStatusIcons[readyState];
    const connectionStatusColor = connectionStatusColors[readyState];

    const handleThemeToggleClick = (event) => {
        const id = event.currentTarget.id;
        setIsSelected(id);

        if ((id === lightModeButtonId && darkMode) || (id == darkModeButtonId && !darkMode)) {
            toggleDarkMode();
        }
    };

    return (
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
                                    icon={<WarningTriangleIcon />}
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
                                        console.log('Requesting fake error message from backend.');

                                        const requestOptions = {
                                            method: 'POST',
                                        };

                                        fetch('api/spoof-error', requestOptions);
                                    }}
                                >
                                    Spoof Error Message
                                </Button>
                            </Tooltip>
                        </FlexItem>

                        <FlexItem>
                            <Tooltip content="Prompt the server to broadcast a bunch of fake notifications for testing/debugging purposes.">
                                <Button
                                    variant="secondary"
                                    icon={<InfoAltIcon />}
                                    onClick={() => {
                                        console.log('Requesting spoofed notifications from backend.');

                                        const requestOptions = {
                                            method: 'POST',
                                        };

                                        fetch('api/spoof-notifications', requestOptions);
                                    }}
                                >
                                    Spoof Notifications
                                </Button>
                            </Tooltip>
                        </FlexItem>

                        <FlexItem>
                            <Tooltip content="Indicates the current connection status with the backend of the Cluster Dashboard.">
                                <Label color={connectionStatusColor} icon={connectionStatusIcon}>
                                    {connectionStatus}
                                </Label>
                            </Tooltip>
                        </FlexItem>
                    </Flex>
                </MastheadContent>
            </MastheadMain>
        </Masthead>
    );
};
