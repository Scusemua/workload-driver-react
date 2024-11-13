import { IAppRoute, IAppRouteGroup, routes } from '@App/routes';
import { DashboardNotificationDrawer } from '@Components/DashboardNotificationDrawer';
import {
    AlertGroup,
    Nav,
    NavExpandable,
    NavItem,
    NavList,
    Page,
    PageSidebar,
    PageSidebarBody,
    SkipToContent,
} from '@patternfly/react-core';
import { AuthorizationContext } from '@Providers/AuthProvider';

import { Notification, WebSocketMessage } from '@src/Data/';
import { DarkModeContext, NotificationContext, useNodes } from '@src/Providers';
import { JoinPaths } from '@src/Utils/path_utils';
import { UnixDurationToString } from '@src/Utils/utils';
import * as React from 'react';
import { toast, ToastBar, Toaster } from 'react-hot-toast';
import { NavLink, useLocation } from 'react-router-dom';

import useWebSocket from 'react-use-websocket';
import { v4 as uuidv4 } from 'uuid';

import { AppHeader } from './AppHeader';

const maxDisplayedAlerts: number = 3;

interface IAppLayout {
    children: React.ReactNode;
}

const AppLayout: React.FunctionComponent<IAppLayout> = ({ children }) => {
    const pageId = 'primary-app-container';

    const [sidebarOpen, setSidebarOpen] = React.useState<boolean>(false);

    const firstRender = React.useRef<boolean>(true);

    const { authenticated, setAuthenticated } = React.useContext(AuthorizationContext);

    const location = useLocation();

    React.useEffect(() => {
        if (!firstRender) {
            console.log('Not first render.');
            return;
        }

        firstRender.current = true;

        // If the user is already authenticated, then don't bother with this.
        if (authenticated) {
            console.log("We're already authenticated.");
            return;
        }

        const authToken: string = localStorage.getItem('token') || '';
        const expiresAtStr: string = localStorage.getItem('token-expiration') || '';

        if (authToken === '') {
            console.debug('Could not recover valid auth token. User will have to log in.');
            return;
        }

        const testAuth: RequestInit = {
            method: 'GET',
            headers: {
                Authorization: 'Bearer ' + authToken,
            },
        };

        fetch('api/config', testAuth)
            .then((resp: Response) => {
                if (resp.status === 200) {
                    // Just make sure the token is not JUST about to expire.
                    // If that's the case, then we'll just make the user log in again.
                    if (!expiresAtStr || expiresAtStr === '') {
                        console.warn(
                            `Recovered valid auth token "${authToken}", but could not recover token's expiration (got string "${expiresAtStr}")... discarding.`,
                        );
                        return; // Not authenticated.
                    }

                    const expiresAt: number = Date.parse(expiresAtStr);
                    const expiresIn: number = expiresAt - Date.now();

                    if (expiresIn <= 60000) {
                        console.warn('Recovered valid auth token, but it expires within 1 minute. Discarding.');

                        // Clear the token and its expiration time from the local storage before returning.
                        localStorage.removeItem('token');
                        localStorage.removeItem('token-expiration');
                        return;
                    }

                    console.log(
                        `We're already authenticated. Token doesn't expire for another ${UnixDurationToString(expiresIn)} seconds at ${expiresAtStr}.`,
                    );
                    setAuthenticated(true);

                    toast.success('Restored existing authenticated user session.', { style: { maxWidth: '500px' } });
                } else if (resp.status == 401) {
                    console.log(
                        `Got response while testing auth: ${resp.status} ${resp.statusText}. User will have to log in.`,
                    );
                } else {
                    console.error(
                        `Unexpected response while testing authentication: ${resp.status} ${resp.statusText} - ${JSON.stringify(resp)}`,
                    );
                    // Assume we're not authenticated.
                }
            })
            .catch((err: Error) => {
                console.error(`Error while testing auth: ${err}`);
                // Assume we're not authenticated.
            });
    }, [firstRender, authenticated, setAuthenticated]);

    const websocketUrl: string = JoinPaths(process.env.PUBLIC_PATH || '/', 'websocket', 'general');
    const { sendJsonMessage, lastJsonMessage } = useWebSocket(
        websocketUrl,
        {
            onOpen: () => {
                console.log(`Successfully connected to Backend Server via WebSocket @ ${websocketUrl}`);
            },
            onError: (event) => {
                console.error(`WebSocket error (addr="${websocketUrl}"). Event: ${JSON.stringify(event)}`);
            },
            share: true,
        },
        authenticated,
    );

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
                if (primaryContentContainer) {
                    primaryContentContainer.focus();
                }
            }}
            href={`#${pageId}`}
        >
            Skip to Content
        </SkipToContent>
    );

    React.useEffect(() => {
        // Don't send any WebSocket messages until we've authenticated.
        if (!authenticated) {
            return;
        }

        sendJsonMessage({
            op: 'register',
            msg_id: uuidv4(),
        });
    }, [sendJsonMessage, authenticated]);

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

    const renderNavItem = (route: IAppRoute, index: number) => (
        <NavItem
            key={`${route.label}-${index}`}
            id={`${route.label}-${index}`}
            isActive={route.path === location.pathname}
        >
            <NavLink to={route.path}>{route.label}</NavLink>
        </NavItem>
    );

    const renderNavGroup = (group: IAppRouteGroup, groupIndex: number) => (
        <NavExpandable
            key={`${group.label}-${groupIndex}`}
            id={`${group.label}-${groupIndex}`}
            title={group.label}
            isActive={group.routes.some((route) => route.path === location.pathname)}
        >
            {group.routes.map((route, idx) => route.label && renderNavItem(route, idx))}
        </NavExpandable>
    );

    const Navigation = (
        <Nav id="nav-primary-simple">
            <NavList id="nav-list-simple">
                {routes.map(
                    (route, idx) =>
                        route.label && (!route.routes ? renderNavItem(route, idx) : renderNavGroup(route, idx)),
                )}
            </NavList>
        </Nav>
    );

    const Sidebar = (
        <PageSidebar>
            <PageSidebarBody>{Navigation}</PageSidebarBody>
        </PageSidebar>
    );

    const onMastheadToggleClicked = () => {
        console.log('onMastheadToggleClicked called');
        setSidebarOpen((curr) => !curr);
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
                header={
                    <AppHeader isLoggedIn={authenticated} onMastheadToggleClicked={() => onMastheadToggleClicked()} />
                }
                skipToContent={PageSkipToContent}
                sidebar={sidebarOpen && Sidebar}
                isNotificationDrawerExpanded={expanded}
                notificationDrawer={authenticated && <DashboardNotificationDrawer />}
            >
                {/*{!authenticated && <DashboardLoginPage />}*/}
                {/*{authenticated && <Dashboard />}*/}
                {authenticated && (
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
                )}
                {children}
            </Page>
        </React.Fragment>
    );
};

export { AppLayout };
