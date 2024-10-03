import { DashboardNotificationDrawer } from '@app/Components';
import { Dashboard } from '@app/Dashboard/Dashboard';

import { Notification, WebSocketMessage } from '@app/Data/';
import { DarkModeContext, NotificationContext, useNodes } from '@app/Providers';
import { AlertGroup, Page, SkipToContent } from '@patternfly/react-core';
import * as React from 'react';
import { Toaster } from 'react-hot-toast';

import useWebSocket from 'react-use-websocket';
import { v4 as uuidv4 } from 'uuid';

import { AppHeader } from './AppHeader';

const AppLayout: React.FunctionComponent = () => {
    const pageId = 'primary-app-container';

    const maxDisplayedAlerts: number = 3;

    const { sendJsonMessage, lastJsonMessage } = useWebSocket('ws://localhost:8000/ws');

    const [overflowMessage, setOverflowMessage] = React.useState<string>('');
    const { alerts, setAlerts, expanded, notifications, addNewNotification, toggleExpansion } =
        React.useContext(NotificationContext);
    const { refreshNodes } = useNodes();

    const { darkMode } = React.useContext(DarkModeContext);

    // React.useEffect(() => {
    //     console.log(`Number of alerts: ${alerts.length}. Number of notifications: ${notifications.length}.`);
    // }, [alerts, notifications]);

    React.useEffect(() => {
        setOverflowMessage(buildOverflowMessage());
    }, [maxDisplayedAlerts, notifications, alerts]);

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

    const buildOverflowMessage = () => {
        const overflow = alerts.length - maxDisplayedAlerts;
        if (overflow > 0 && maxDisplayedAlerts > 0) {
            return `View ${overflow} more notification(s) in notification drawer`;
        }
        return '';
    };

    React.useEffect(() => {
        sendJsonMessage({
            op: 'register',
            msg_id: uuidv4(),
        });
    }, [sendJsonMessage]);

    React.useEffect(() => {
        if (lastJsonMessage !== null) {
            const message: WebSocketMessage = lastJsonMessage as WebSocketMessage;

            if (message.op == 'notification') {
                const notification: Notification = message.payload as Notification;
                console.log(
                    `Received "${notification.title}" notification (typ=${notification.notificationType}, panicked=${notification.panicked}) via WebSocket: ${notification.message}`,
                );

                addNewNotification(notification);

                // If we received a notification that a Local Daemon has either connected to the Cluster or the Cluster
                // has lost connection to a Local Daemon, then we should automatically refresh the nodes so that the UI updates accordingly.
                if (
                    notification.title == 'Local Daemon Connected' ||
                    notification.title == 'Local Daemon Connectivity Error'
                ) {
                    refreshNodes();
                }
            } else {
                console.warn(`Received JSON message of unknown type: ${JSON.stringify(message)}`);
            }
        }
    }, [addNewNotification, lastJsonMessage, refreshNodes]);

    /**
     * Return the default style applied to all Toast notifications.
     *
     * We return a dark mode style if the page is set to dark mode.
     */
    const getDefaultToastStyle = () => {
        if (darkMode) {
            return {
                zIndex: 9999,
                borderRadius: '10px',
                background: '#333',
                color: '#fff',
            };
        } else {
            return {
                zIndex: 9999,
                borderRadius: '10px',
            };
        }
    };

    return (
        <React.Fragment>
            <Toaster
                position="bottom-right"
                containerStyle={{
                    zIndex: 9999,
                }}
                toastOptions={{
                    className: 'react-hot-toast',
                    style: getDefaultToastStyle(),
                }}
            />
            <Page
                mainContainerId={pageId}
                header={<AppHeader />}
                skipToContent={PageSkipToContent}
                isNotificationDrawerExpanded={expanded}
                notificationDrawer={<DashboardNotificationDrawer />}
            >
                <Dashboard />
                <AlertGroup
                    isToast
                    isLiveRegion
                    onOverflowClick={() => {
                        setAlerts([]);
                        toggleExpansion(true);
                    }}
                    overflowMessage={overflowMessage}
                >
                    {alerts.slice(0, maxDisplayedAlerts)}
                </AlertGroup>
            </Page>
        </React.Fragment>
    );
};

export { AppLayout };
