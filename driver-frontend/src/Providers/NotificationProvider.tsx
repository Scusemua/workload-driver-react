import { Notification } from '@src/Data';
import { Alert, AlertActionCloseButton, AlertProps } from '@patternfly/react-core';
import React from 'react';
import { v4 as uuidv4 } from 'uuid';

const notificationVariants: ('danger' | 'warning' | 'info' | 'success')[] = ['danger', 'warning', 'info', 'success'];

type NotificationContext = {
    alerts: React.ReactElement<AlertProps>[];
    notifications: NotificationProps[];
    expanded: boolean;
    setAlerts: React.Dispatch<
        React.SetStateAction<React.ReactElement<AlertProps, string | React.JSXElementConstructor<any>>[]>
    >;
    setNotifications: React.Dispatch<React.SetStateAction<NotificationProps[]>>;
    addNewNotification: (notification: Notification) => void;
    toggleExpansion: (val: boolean) => void;
};

interface NotificationProps {
    title: string;
    variant: 'danger' | 'warning' | 'info' | 'success';
    key: React.Key;
    timestamp: string;
    description: string;
    isNotificationRead: boolean;
}

const initialState: NotificationContext = {
    alerts: [],
    notifications: [],
    expanded: false,
    setAlerts: () => {},
    setNotifications: () => {},
    addNewNotification: () => {},
    toggleExpansion: () => {},
};

const NotificationContext = React.createContext<NotificationContext>(initialState);

const getTimeCreated = () => {
    const dateCreated = new Date();
    return (
        dateCreated.toDateString() +
        ' at ' +
        ('00' + dateCreated.getHours().toString()).slice(-2) +
        ':' +
        ('00' + dateCreated.getMinutes().toString()).slice(-2)
    );
};

const NotificationProvider = (props) => {
    const alertTimeout: number = 8000;

    const [alerts, setAlerts] = React.useState<React.ReactElement<AlertProps>[]>([]);
    const [notifications, setNotifications] = React.useState<NotificationProps[]>([]);
    const [expanded, setExpanded] = React.useState<boolean>(false);

    const toggleExpansion = (val: boolean) => {
        setExpanded(val);
    };

    const addNewNotification = (notification: Notification) => {
        const title: string = notification.title;
        const description: string = notification.message;
        const key: string = uuidv4();
        const timestamp: string = getTimeCreated();
        let variant: 'danger' | 'warning' | 'info' | 'success' = notificationVariants[notification.notificationType];

        if (variant === undefined) {
            variant = 'danger';
        }

        console.log(
            `Adding new notification: ${title}: ${description} (key=${key}, timestamp=${timestamp}, variant=${variant})`,
        );

        setNotifications((prevNotifications) => [
            { title, variant, key, timestamp, description, isNotificationRead: false },
            ...prevNotifications,
        ]);

        if (!expanded) {
            setAlerts((prevAlerts) => [
                <Alert
                    variant={variant}
                    title={title}
                    timeout={alertTimeout}
                    onTimeout={() =>
                        setAlerts((prevAlerts) => prevAlerts.filter((alert) => alert.props.id !== key.toString()))
                    }
                    isLiveRegion
                    actionClose={
                        <AlertActionCloseButton
                            title={title}
                            variantLabel={`${variant} alert`}
                            onClose={() =>
                                setAlerts((prevAlerts) =>
                                    prevAlerts.filter((alert) => alert.props.id !== key.toString()),
                                )
                            }
                        />
                    }
                    key={key}
                    id={key.toString()}
                >
                    <p>{description}</p>
                </Alert>,
                ...prevAlerts,
            ]);
        }
    };

    return (
        <NotificationContext.Provider
            value={{
                alerts,
                notifications,
                expanded,
                setAlerts,
                setNotifications,
                addNewNotification,
                toggleExpansion,
            }}
        >
            {props.children}
        </NotificationContext.Provider>
    );
};

export { NotificationContext, NotificationProvider, NotificationProps };
