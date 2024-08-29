import logo from '@app/bgimages/WorkloadDriver-Logo.svg';
import {NotificationContext} from '@app/Providers';

import {DarkModeContext} from '@app/Providers/DarkModeProvider';
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
  NotificationBadge,
  NotificationBadgeVariant,
  ToggleGroup,
  ToggleGroupItem,
  ToolbarItem,
  Tooltip,
} from '@patternfly/react-core';
import {
  CheckCircleIcon,
  ErrorCircleOIcon,
  InfoAltIcon,
  MoonIcon,
  SkullCrossbonesIcon,
  SunIcon,
  WarningTriangleIcon
} from '@patternfly/react-icons';
import * as React from 'react';
import useWebSocket, {ReadyState} from 'react-use-websocket';

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
      <CheckCircleIcon/>
    </Icon>
  ),
  [ReadyState.OPEN]: (
    <Icon isInProgress={false}>
      <CheckCircleIcon/>
    </Icon>
  ),
  [ReadyState.CLOSING]: (
    <Icon isInProgress={true}>
      <CheckCircleIcon/>
    </Icon>
  ),
  [ReadyState.CLOSED]: (
    <Icon isInProgress={false}>
      <ErrorCircleOIcon/>
    </Icon>
  ),
  [ReadyState.UNINSTANTIATED]: (
    <Icon isInProgress={false}>
      <WarningTriangleIcon/>
    </Icon>
  ),
};

type connectionStatusColorsType = {
  [key in ReadyState]: "green" | "red" | "blue" | "cyan" | "orange" | "purple" | "grey" | "gold" | undefined;
};

const connectionStatusColors: connectionStatusColorsType = {
  [ReadyState.CONNECTING]: 'orange',
  [ReadyState.OPEN]: 'green',
  [ReadyState.CLOSING]: 'orange',
  [ReadyState.CLOSED]: 'red',
  [ReadyState.UNINSTANTIATED]: 'orange',
};

export const AppHeader: React.FunctionComponent = () => {
  const lightModeId: string = 'theme-toggle-lightmode';
  const darkModeId: string = 'theme-toggle-darkmode';
  const lightModeButtonId: string = lightModeId + '-button';
  const darkModeButtonId: string = darkModeId + '-button';

  // Flag to keep track of if we fail to connect to the backend.
  // This is reset (to false) upon successful connection to backend.
  // We only show a notification about failing to connect once (the first time).
  const failedToConnect = React.useRef<boolean>(false);

  // Cache the last value of `readyState` so that, if we disconnect, we can determine if we were previously connected.
  // If so, we'll display a notification about *losing* connection.
  const [prevConnectionState, setPrevConnectionState] = React.useState<ReadyState>(ReadyState.UNINSTANTIATED);

  const {darkMode, toggleDarkMode} = React.useContext(DarkModeContext);

  const {setAlerts, expanded, notifications, toggleExpansion, addNewNotification} =
    React.useContext(NotificationContext);

  const [isSelected, setIsSelected] = React.useState(darkMode ? darkModeButtonId : lightModeButtonId);

  const {readyState} = useWebSocket('ws://localhost:8000/ws', {
    // onOpen: () => {},
    // onClose: () => {},
    shouldReconnect: () => true,
  });

  React.useEffect(() => {
    switch (readyState) {
      case ReadyState.CLOSED:
        if (!failedToConnect.current) {
          addNewNotification({
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
        addNewNotification({
          title: 'Connection Established',
          message: 'The persistent connection with the backend server has been established.',
          notificationType: 3,
          panicked: false,
        });

        // Reset this to false, as we just successfully connected.
        failedToConnect.current = false;
        break;
    }

    setPrevConnectionState(readyState);
  }, [readyState]);

  const connectionStatus = connectionStatuses[readyState];
  const connectionStatusIcon = connectionStatusIcons[readyState];
  const connectionStatusColor: "green" | "red" | "blue" | "cyan" | "orange" | "purple" | "grey" | "gold" | undefined = connectionStatusColors[readyState];

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
    notifications.filter((notification) => notification.isNotificationRead === false).length;

  const containsUnreadAlertNotification = () =>
    notifications.filter(
      (notification) =>
        notification.isNotificationRead === false &&
        (notification.variant === 'danger' || notification.variant === 'warning'),
    ).length > 0;

  const getNotificationBadgeVariant = () => {
    if (getUnreadNotificationsNumber() === 0) {
      return NotificationBadgeVariant.read;
    }
    if (containsUnreadAlertNotification()) {
      return NotificationBadgeVariant.attention;
    }
    return NotificationBadgeVariant.unread;
  };

  const notificationBadge = (
    <ToolbarItem>
      <NotificationBadge
        variant={getNotificationBadgeVariant()}
        onClick={onNotificationBadgeClick}
        aria-label="Notifications"
        count={getUnreadNotificationsNumber()}
      ></NotificationBadge>
    </ToolbarItem>
  );

  return (
    <Masthead>
      <MastheadMain>
        <MastheadBrand>
          <Brand src={logo} alt="Workload Driver Logo" heights={{default: '36px'}}/>
        </MastheadBrand>
        <MastheadContent>
          <Flex direction={{default: 'row'}}>
            <FlexItem>
              <div className="pf-v5-theme-dark">
                <ToggleGroup>
                  <ToggleGroupItem
                    aria-label={lightModeId}
                    id={lightModeId}
                    buttonId={lightModeButtonId}
                    icon={<SunIcon/>}
                    onChange={handleThemeToggleClick}
                    isSelected={isSelected === lightModeButtonId}
                  />
                  <ToggleGroupItem
                    aria-label={darkModeId}
                    id={darkModeId}
                    buttonId={darkModeButtonId}
                    icon={<MoonIcon/>}
                    onChange={handleThemeToggleClick}
                    isSelected={isSelected === darkModeButtonId}
                  />
                </ToggleGroup>
              </div>
            </FlexItem>

            <FlexItem>
              <Tooltip content="Open the notification drawer." position="bottom">
                {notificationBadge}
              </Tooltip>
            </FlexItem>

            <FlexItem>
              <Tooltip content="Cause the Cluster Gateway to panic." position="bottom">
                <Button
                  isDanger
                  variant="secondary"
                  icon={<WarningTriangleIcon/>}
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
              <Tooltip
                content="Prompt the server to broadcast a fake error for testing/debugging purposes."
                position="bottom"
              >
                <Button
                  isDanger
                  variant="secondary"
                  icon={<SkullCrossbonesIcon/>}
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
              <Tooltip
                content="Prompt the server to broadcast a bunch of fake notifications for testing/debugging purposes."
                position="bottom"
              >
                <Button
                  variant="secondary"
                  icon={<InfoAltIcon/>}
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
              <Tooltip
                content="Indicates the current connection status with the backend of the Cluster Dashboard."
                position="bottom"
              >
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
