import * as React from 'react';
import {
    Alert,
    Brand,
    Button,
    Flex,
    FlexItem,
    Icon,
    Label,
    Masthead,
    MastheadBrand,
    MastheadContent,
    MastheadMain,
    ToggleGroup,
    ToggleGroupItem,
    Tooltip,
} from '@patternfly/react-core';
import logo from '@app/bgimages/WorkloadDriver-Logo.svg';
import { MoonIcon, SkullCrossbonesIcon, SunIcon } from '@patternfly/react-icons';

import { v4 as uuidv4 } from 'uuid';
import { DarkModeContext } from '@app/Providers/DarkModeProvider';
import { ErrorCircleOIcon, WarningTriangleIcon } from '@patternfly/react-icons';
import useWebSocket, { ReadyState } from 'react-use-websocket';
import { CheckCircleIcon } from '@patternfly/react-icons';

import { ErrorMessage, WebSocketMessage } from '@app/Data/WebSocket';
import toast from 'react-hot-toast';

export const AppHeader: React.FunctionComponent = () => {
    const lightModeId: string = 'theme-toggle-lightmode';
    const darkModeId: string = 'theme-toggle-darkmode';
    const lightModeButtonId: string = lightModeId + '-button';
    const darkModeButtonId: string = darkModeId + '-button';

    const { darkMode, toggleDarkMode } = React.useContext(DarkModeContext);

    const [isSelected, setIsSelected] = React.useState(darkMode ? darkModeButtonId : lightModeButtonId);

    const { sendJsonMessage, lastJsonMessage, lastMessage, readyState } = useWebSocket('ws://localhost:8000/ws');

    const connectionStatus = {
        [ReadyState.CONNECTING]: 'Connecting to Backend ...',
        [ReadyState.OPEN]: 'Connected to Backend ',
        [ReadyState.CLOSING]: 'Disconnecting from Backend ...',
        [ReadyState.CLOSED]: 'Disconnected from Backend ',
        [ReadyState.UNINSTANTIATED]: 'Backend Connection Uninstantiated',
    }[readyState];

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
    }[readyState];

    const connectionStatusColors = {
        [ReadyState.CONNECTING]: 'yellow',
        [ReadyState.OPEN]: 'green',
        [ReadyState.CLOSING]: 'yellow',
        [ReadyState.CLOSED]: 'red',
        [ReadyState.UNINSTANTIATED]: 'yellow',
    }[readyState];

    React.useEffect(() => {
        sendJsonMessage({
            op: 'register',
            msg_id: uuidv4(),
        });
    }, [sendJsonMessage]);

    React.useEffect(() => {
        if (lastMessage !== null) {
            console.log(`Received general WebSocket binary message: ${lastMessage}`);
        }
    }, [lastMessage]);

    React.useEffect(() => {
        if (lastJsonMessage !== null) {
            console.log(`Received general WebSocket JSON message: ${lastJsonMessage}`);
            const message: WebSocketMessage = lastJsonMessage as WebSocketMessage;

            if (message.op === 'error') {
                const errorMessage: ErrorMessage = message.payload as ErrorMessage;
                toast.custom(
                    <Alert variant="danger" title={errorMessage.errorName} ouiaId="DangerAlert">
                        {errorMessage.errorMessage}
                    </Alert>,
                );
            }
        }
    }, [lastJsonMessage]);

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
                                        console.log('Requesting fake error message from backend.');

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
                        <FlexItem>
                            <Tooltip content="Indicates the current connection status with the backend of the Cluster Dashboard.">
                                <Label color={connectionStatusColors} icon={connectionStatusIcons}>
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
