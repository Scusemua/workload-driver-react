import {
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
    MastheadToggle,
    ToggleGroup,
    ToggleGroupItem,
    ToolbarItem,
    Tooltip,
} from '@patternfly/react-core';
import {
    AttentionBellIcon,
    BarsIcon,
    CheckCircleIcon,
    ClockIcon,
    ErrorCircleOIcon,
    InfoAltIcon,
    MoonIcon,
    SunIcon,
    WarningTriangleIcon,
} from '@patternfly/react-icons';
import { AuthorizationContext } from '@Providers/AuthProvider';
import { useClusterAge } from '@Providers/ClusterAgeProvider';
import { DarkModeContext } from '@Providers/DarkModeProvider';
import { useClusterDeploymentMode } from '@Providers/DeploymentModeProvider';
import { useKernels } from '@Providers/KernelProvider';
import { useNodes } from '@Providers/NodeProvider';
import { NotificationContext } from '@Providers/NotificationProvider';
import { useClusterSchedulingPolicy } from '@Providers/SchedulingPolicyProvider';
import logo from '@src/app/bgimages/WorkloadDriver-Logo.svg';
import { GetPathForFetch, JoinPaths } from '@src/Utils/path_utils';
import * as React from 'react';
import { useContext } from 'react';
import useWebSocket, { ReadyState } from 'react-use-websocket';
import { QueryMessageModal } from 'src/Components/Modals';
import { FormatSecondsShort } from 'src/Utils/utils';

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

type statusColor = {
    [key in ReadyState]: 'green' | 'red' | 'blue' | 'cyan' | 'orange' | 'purple' | 'grey' | 'gold' | undefined;
};

const connectionStatusColors: statusColor = {
    [ReadyState.CONNECTING]: 'orange',
    [ReadyState.OPEN]: 'green',
    [ReadyState.CLOSING]: 'orange',
    [ReadyState.CLOSED]: 'red',
    [ReadyState.UNINSTANTIATED]: 'orange',
};

interface AppHeaderProps {
    isLoggedIn: boolean;
    onMastheadToggleClicked: () => void;
}

const toastIdFailedToConnect: string = '__TOAST_ERROR_FAILED_TO_CONNECT__';
const toastIdConnectionEstablished: string = '__TOAST_CONNECTION_ESTABLISHED__';
const toastIdConnectionLost: string = '__TOAST_WARNING_CONNECTION_LOST__';

export const AppHeader: React.FunctionComponent<AppHeaderProps> = (props: AppHeaderProps) => {
    const lightModeId: string = 'theme-toggle-lightmode';
    const darkModeId: string = 'theme-toggle-darkmode';
    const lightModeButtonId: string = lightModeId + '-button';
    const darkModeButtonId: string = darkModeId + '-button';

    const [queryMessageModalOpen, setQueryMessageModalOpen] = React.useState<boolean>(false);

    const { clusterAge } = useClusterAge();
    const { schedulingPolicy } = useClusterSchedulingPolicy();
    const { deploymentMode } = useClusterDeploymentMode();
    const { refreshNodes } = useNodes();
    const { refreshKernels } = useKernels(false);

    const { authenticated } = useContext(AuthorizationContext);

    const [currentClusterAge, setCurrentClusterAge] = React.useState<string>('N/A');

    React.useEffect(() => {
        const intervalId = setInterval(() => {
            if (clusterAge !== undefined && clusterAge > 0) {
                setCurrentClusterAge(FormatSecondsShort((Date.now() - ((clusterAge as number) || 0)) / 1000.0));
            }
        }, 1000); // Update every 1 second.

        return () => clearInterval(intervalId); // Cleanup on unmount.
    }, [clusterAge]);

    // Flag to keep track of if we fail to connect to the backend.
    // This is reset (to false) upon successful connection to backend.
    // We only show a notification about failing to connect once (the first time).
    const failedToConnect = React.useRef<boolean>(false);

    // Cache the last value of `readyState` so that, if we disconnect, we can determine if we were previously connected.
    // If so, we'll display a notification about *losing* connection.
    const [prevConnectionState, setPrevConnectionState] = React.useState<ReadyState>(ReadyState.UNINSTANTIATED);

    const { darkMode, toggleDarkMode } = React.useContext(DarkModeContext);

    const { setAlerts, expanded, notifications, toggleExpansion, addNewNotification } =
        React.useContext(NotificationContext);

    const [isSelected, setIsSelected] = React.useState(darkMode ? darkModeButtonId : lightModeButtonId);

    const websocketUrl: string = JoinPaths(process.env.PUBLIC_PATH || '/', 'websocket', 'general');
    const { readyState } = useWebSocket(
        websocketUrl,
        {
            shouldReconnect: () => authenticated,
            reconnectAttempts: 50,
            onError: (evt) => {
                console.error(`WebSocket encountered error: ${JSON.stringify(evt)}. WebSocket URL: ${websocketUrl}`);
            },
            onClose: (evt) => {
                console.warn(`WebSocket connection has closed: ${JSON.stringify(evt)}. WebSocket URL: ${websocketUrl}`);
            },
            onOpen: (evt) => {
                console.debug(
                    `WebSocket connection has been established: ${JSON.stringify(evt)}. WebSocket URL: ${websocketUrl}`,
                );
            },
            share: true,
        },
        authenticated,
    );

    const handleConnectionStateChange = React.useCallback(() => {
        if (!authenticated) {
            return;
        }

        switch (readyState) {
            case ReadyState.CLOSED:
                if (!failedToConnect.current) {
                    addNewNotification({
                        id: toastIdFailedToConnect,
                        title: 'Failed to Connect to Backend',
                        message: 'The persistent connection with the backend server could not be established lost.',
                        notificationType: 1,
                        panicked: false,
                    });
                    console.error('Failed to connect to backend');

                    // Take note that we failed to connect.
                    // This will prevent us from posting the same notification.
                    failedToConnect.current = true;
                } else if (prevConnectionState == ReadyState.OPEN) {
                    addNewNotification({
                        id: toastIdConnectionLost,
                        title: 'Connection Lost to Backend',
                        message: 'The persistent connection with the backend server has been lost.',
                        notificationType: 1,
                        panicked: false,
                    });
                    console.error('Lost connection to backend');

                    // Don't set the value of 'failedToConnect' yet.
                    // We want to display a 'failed to connect' notification if we fail to reconnect.
                }
                break;
            case ReadyState.OPEN:
                console.log('Connected to backend');

                if (prevConnectionState !== ReadyState.OPEN) {
                    addNewNotification({
                        id: toastIdConnectionEstablished,
                        title: 'Connection Established',
                        message: 'The persistent connection with the backend server has been established.',
                        notificationType: 3,
                        panicked: false,
                    });
                }

                // If we've just connected, then let's refresh our kernels and our nodes, in case they've
                // changed since we were last connected.
                refreshKernels()
                    .then(() => {})
                    .catch((err: Error) => console.log(`Kernel refresh failed: ${err}`));
                refreshNodes(false); // Pass false to omit the separate toast notification about refreshing nodes.

                // Reset this to false, as we just successfully connected.
                failedToConnect.current = false;
                break;
        }

        setPrevConnectionState(readyState);
        // If I add the dependencies 'refreshKernels, refreshNodes, addNewNotification' here,
        // then it sends a million requests when the connection with the backend is not available.
    }, [authenticated, readyState, prevConnectionState]);

    React.useEffect(() => {
        handleConnectionStateChange();
    }, [prevConnectionState, readyState, authenticated, handleConnectionStateChange]);

    const connectionStatus = connectionStatuses[readyState];
    const connectionStatusIcon = connectionStatusIcons[readyState];
    const connectionStatusColor: 'green' | 'red' | 'blue' | 'cyan' | 'orange' | 'purple' | 'grey' | 'gold' | undefined =
        connectionStatusColors[readyState];

    const handleThemeToggleClick = (event) => {
        const id = event.currentTarget.id;
        setIsSelected(id);

        if ((id === lightModeButtonId && darkMode) || (id == darkModeButtonId && !darkMode)) {
            toggleDarkMode();
        }
    };

    const onNotificationBadgeClick = () => {
        setAlerts([]);
        toggleExpansion(!expanded);
    };

    const getUnreadNotificationsNumber = () =>
        notifications.filter((notification) => !notification.isNotificationRead).length;

    const containsUnreadErrorNotifications = () =>
        notifications.filter((notification) => !notification.isNotificationRead && notification.variant === 'danger')
            .length > 0;

    const containsUnreadWarningNotifications = () =>
        notifications.filter((notification) => !notification.isNotificationRead && notification.variant === 'warning')
            .length > 0;

    const getNotificationBadgeIconClassName = (): string => {
        if (getUnreadNotificationsNumber() === 0) {
            return 'notification-badge-icon-no-notifications';
        }

        return '';
    };

    const getNotificationBadgeClassName = (): string => {
        if (getUnreadNotificationsNumber() === 0) {
            return 'notification-badge-no-notifications';
        }

        if (containsUnreadWarningNotifications()) {
            return 'pf-m-warning';
        }

        if (containsUnreadErrorNotifications()) {
            return 'pf-m-danger';
        }

        return 'pf-m-info';
    };

    // const getNotificationBadgeButtonOptions = ():BadgeCountObject => {
    //   const numUnreadNotifications: number = getUnreadNotificationsNumber();
    //
    //   return {
    //     isRead: numUnreadNotifications === 0,
    //     count: numUnreadNotifications,
    //     className: (numUnreadNotifications === 0) ? "custom-badge-read" : "custom-badge-unread"
    //   }
    // }

    const notificationBadge = (
        <ToolbarItem>
            <Button
                id={'notification-badge-button'}
                onClick={onNotificationBadgeClick}
                variant={'primary'}
                aria-label="Notifications"
                icon={
                    <AttentionBellIcon
                        id={'notification-badge-button-icon'}
                        className={getNotificationBadgeIconClassName()}
                    />
                }
                className={getNotificationBadgeClassName()}
            >
                {getUnreadNotificationsNumber()}
            </Button>
        </ToolbarItem>
    );

    return (
        <Masthead>
            <MastheadMain>
                <MastheadToggle>
                    <Button
                        variant="plain"
                        onClick={() => props.onMastheadToggleClicked()}
                        aria-label="Global navigation"
                    >
                        <BarsIcon />
                    </Button>
                </MastheadToggle>
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

                        {props.isLoggedIn && (
                            <FlexItem>
                                <Tooltip content="Open the notification drawer." position="bottom">
                                    {notificationBadge}
                                </Tooltip>
                            </FlexItem>
                        )}

                        {props.isLoggedIn && (
                            <FlexItem>
                                <Tooltip content="Cause the Cluster Gateway to panic." position="bottom">
                                    <Button
                                        isDanger
                                        key={'cause-gateway-panic-button'}
                                        variant="secondary"
                                        icon={<WarningTriangleIcon />}
                                        onClick={() => {
                                            const requestOptions = {
                                                method: 'POST',
                                                Headers: {
                                                    Authorization: 'Bearer ' + localStorage.getItem('token'),
                                                },
                                            };

                                            fetch(GetPathForFetch('api/panic'), requestOptions).then(() => {});
                                        }}
                                    >
                                        Panic
                                    </Button>
                                </Tooltip>
                            </FlexItem>
                        )}

                        {props.isLoggedIn && (
                            <FlexItem>
                                <Tooltip
                                    content={'Query the status of a particular Jupyter ZMQ message.'}
                                    position={'bottom'}
                                >
                                    <Button
                                        key={'open-query-message-modal-button'}
                                        variant={'secondary'}
                                        icon={<InfoAltIcon />}
                                        onClick={() => setQueryMessageModalOpen(true)}
                                    >
                                        Query Message
                                    </Button>
                                </Tooltip>
                            </FlexItem>
                        )}

                        <FlexItem>
                            <Tooltip content="Indicates whether we're presently authenticated." position="bottom">
                                <Label
                                    isCompact
                                    color={authenticated ? 'green' : 'orange'}
                                    icon={authenticated ? <CheckCircleIcon /> : <WarningTriangleIcon />}
                                >
                                    {authenticated ? 'Authenticated (Logged In)' : 'Unauthenticated (Logged Out)'}
                                </Label>
                            </Tooltip>
                        </FlexItem>

                        {props.isLoggedIn && (
                            <FlexItem>
                                <Tooltip
                                    content="Indicates the current connection status with the backend of the Cluster Dashboard."
                                    position="bottom"
                                >
                                    <Label color={connectionStatusColor} icon={connectionStatusIcon} isCompact>
                                        {connectionStatus}
                                    </Label>
                                </Tooltip>
                            </FlexItem>
                        )}

                        {props.isLoggedIn && (
                            <FlexItem>
                                <Tooltip content={'Age of the cluster'} position="bottom">
                                    <Label isCompact color={'purple'} icon={<ClockIcon />}>
                                        <b>Cluster Age:</b> {currentClusterAge}
                                    </Label>
                                </Tooltip>
                            </FlexItem>
                        )}

                        {props.isLoggedIn && (
                            <FlexItem>
                                <Tooltip
                                    content={'Configured scheduling policy employed by the cluster'}
                                    position="bottom"
                                >
                                    <Label isCompact color={'cyan'} icon={<ClockIcon />}>
                                        <b>Scheduling Policy:</b> {schedulingPolicy}
                                    </Label>
                                </Tooltip>
                            </FlexItem>
                        )}

                        {props.isLoggedIn && (
                            <FlexItem>
                                <Tooltip content={'Configured deployment mode of the cluster'} position="bottom">
                                    <Label isCompact color={'gold'} icon={<ClockIcon />}>
                                        <b>Deployment Mode:</b> {deploymentMode}
                                    </Label>
                                </Tooltip>
                            </FlexItem>
                        )}
                    </Flex>
                </MastheadContent>
            </MastheadMain>

            {props.isLoggedIn && (
                <QueryMessageModal isOpen={queryMessageModalOpen} onClose={() => setQueryMessageModalOpen(false)} />
            )}
        </Masthead>
    );
};
