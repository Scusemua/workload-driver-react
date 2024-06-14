import * as React from 'react';
import { Alert, AlertActionCloseButton, AlertGroup, Page, SkipToContent } from '@patternfly/react-core';

import { AppHeader } from './AppHeader';
import { DashboardNotificationDrawer } from '@app/Components';
import { Dashboard } from '@app/Dashboard/Dashboard';
import { NotificationContext } from '@app/Providers';

import { Notification, WebSocketMessage } from '@app/Data/';

import useWebSocket from 'react-use-websocket';
import { v4 as uuidv4 } from 'uuid';

const AppLayout: React.FunctionComponent = () => {
    const pageId = 'primary-app-container';

    const maxDisplayedAlerts: number = 3;

    const { sendJsonMessage, lastJsonMessage } = useWebSocket('ws://localhost:8000/ws');

    const [overflowMessage, setOverflowMessage] = React.useState<string>('');
    const { alerts, setAlerts, expanded, notifications, addNewNotification, toggleExpansion } =
        React.useContext(NotificationContext);

    React.useEffect(() => {
        console.log(`Number of alerts: ${alerts.length}. Number of notifications: ${notifications.length}.`);
    }, [alerts, notifications]);

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
            } else {
                console.warn(`Received JSON message of unknown type: ${message}`);
            }
        }
    }, [lastJsonMessage]);

    return (
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
    );
};

export { AppLayout };
