import { Dashboard } from '@App/Dashboard';
import { DashboardLoginPage } from '@App/DashboardLoginPage';
import { DashboardNotificationDrawer } from '@Components/DashboardNotificationDrawer';
import { AlertGroup, Page, SkipToContent } from '@patternfly/react-core';
import { AuthorizationContext, AuthProvider } from '@Providers/AuthProvider';

import { Notification, WebSocketMessage } from '@src/Data/';
import { DarkModeContext, NotificationContext, useNodes } from '@src/Providers';
import * as React from 'react';
import { ToastBar, Toaster } from 'react-hot-toast';

import useWebSocket from 'react-use-websocket';
import { v4 as uuidv4 } from 'uuid';

import { AppHeader } from './AppHeader';

const maxDisplayedAlerts: number = 3;

const AppLayout: React.FunctionComponent = () => {
    const pageId = 'primary-app-container';

    const { authenticated, setAuthenticated } = React.useContext(AuthorizationContext);

    const { sendJsonMessage, lastJsonMessage } = useWebSocket('ws://localhost:8000/ws');

    const [overflowMessage, setOverflowMessage] = React.useState<string>('');
    const { alerts, setAlerts, expanded, notifications, addNewNotification, toggleExpansion } =
        React.useContext(NotificationContext);
    const { refreshNodes } = useNodes();

    const { darkMode } = React.useContext(DarkModeContext);

    React.useEffect(() => {
        const overflow: number = alerts.length - maxDisplayedAlerts;
        if (overflow > 0 && maxDisplayedAlerts > 0) {
            setOverflowMessage(`View ${overflow} more notification(s) in notification drawer`);
        }
        setOverflowMessage('');
    }, [notifications, alerts]);

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

    React.useEffect(() => {
        sendJsonMessage({
            op: 'register',
            msg_id: uuidv4(),
        });
    }, [sendJsonMessage]);

    React.useEffect(() => {
        if (!authenticated) {
            return;
        }

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
                    refreshNodes(false); // Pass false to omit the separate toast notification about refreshing nodes.
                }
            } else {
                console.warn(`Received JSON message of unknown type: ${JSON.stringify(message)}`);
            }
        }
    }, [lastJsonMessage]);

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

    const onSuccessfulLogin = (token: string, expiration: string) => {
        console.log(`Authenticated successfully: ${token}. Token will expire at: ${expiration}.`);
        localStorage.setItem("token", token);
        localStorage.setItem("token-expiration", expiration);
        setAuthenticated(true);
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
                >
                    {(t) => (
                        <ToastBar
                            toast={t}
                            style={{
                                ...t.style,
                                animation: t.visible ? 'custom-enter 1s ease' : 'custom-exit 1s ease',
                            }}
                        />
                    )}
                </Toaster>
                <Page
                    mainContainerId={pageId}
                    header={<AppHeader isLoggedIn={authenticated} />}
                    skipToContent={PageSkipToContent}
                    isNotificationDrawerExpanded={expanded}
                    notificationDrawer={authenticated && <DashboardNotificationDrawer />}
                >
                    {!authenticated && <DashboardLoginPage onSuccessfulLogin={onSuccessfulLogin} />}
                    {authenticated && <Dashboard />}
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
