import { RoundToThreeDecimalPlaces } from '@Components/Modals';
import { Alert, AlertActionCloseButton } from '@patternfly/react-core';
import { SpinnerIcon } from '@patternfly/react-icons';
import { GetPathForFetch } from '@src/Utils/path_utils';
import { GetToastContentWithHeaderAndBody } from '@src/Utils/toast_utils';
import { MAX_SAFE_INTEGER } from 'lib0/number';
import React from 'react';
import { Toast, toast } from 'react-hot-toast';
import useSWR from 'swr';

const toastIdLoggedOut: string = '__TOAST_ERROR_LOGGED_OUT__';

type AuthContext = {
    authenticated: boolean;
    setAuthenticated: (auth: boolean) => void;
    username: string | undefined;
    password: string | undefined;
    setUsername: (username: string | undefined) => void;
    setPassword: (password: string | undefined) => void;
    mutateToken: (username: string, password: string) => Promise<boolean>;
    error: any;
};

const initialState: AuthContext = {
    authenticated: false,
    setAuthenticated: () => {},
    username: undefined,
    setUsername: () => {},
    password: undefined,
    setPassword: () => {},
    mutateToken: async () => false,
    error: undefined,
};

const refreshTokenEndpoint: string = GetPathForFetch('/refresh_token');
const loginEndpoint: string = GetPathForFetch('/authenticate');

const AuthorizationContext = React.createContext<AuthContext>(initialState);

interface AuthResponse {
    token: string;
    expire: string | number;
    username: string;
    password: string;
    refreshed: boolean; // If true, then token was refreshed, rather than created for first time.
    toastId: string | undefined;
}

async function tokenFetcher(
    endpoint: RequestInfo | URL,
    username: string | undefined,
    password: string | undefined,
    currentlyAuthenticated: boolean,
    toastId: string | undefined = undefined,
): Promise<AuthResponse | void> {
    console.log(
        `Refreshing token. Endpoint: "${endpoint}". Username: "${username}". Password: "${password}". ToastID: ${toastId}. Currently authenticated: ${currentlyAuthenticated}.`,
    );

    const abortController: AbortController = new AbortController();
    const signal: AbortSignal = abortController.signal;
    const timeout: number = 30000;

    if (!username || !password) {
        return;
    }

    const requestBody = {
        username: username,
        password: password,
    };

    if (currentlyAuthenticated) {
        requestBody['token'] = localStorage.getItem('token') || '';
    }

    const init: RequestInit = {
        method: 'POST',
        headers: {
            Authorization: 'Bearer ' + localStorage.getItem('token'),
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestBody),
        signal: signal,
    };

    setTimeout(() => {
        abortController.abort(`The request timed-out after ${timeout} milliseconds.`);
    }, timeout);

    if (endpoint == loginEndpoint) {
        console.log(`Body of LOGIN request:\n${JSON.stringify(requestBody, null, 2)}`);
    }

    const response: Response = await fetch(endpoint, init);
    const responseJSON: AuthResponse | Error = await response.json();

    if (response.status !== 200) {
        console.log(`Authenticate failed. Could not log in:\n${JSON.stringify(responseJSON, null, 2)}`);
        throw new Error(`HTTP ${response.status} ${response.statusText}: ${(responseJSON as Error).message}`);
    } else {
        console.log(`Fetched JWT token:\n${JSON.stringify(responseJSON, null, 2)}`);
    }

    console.log(`tokenFetcher toastId: ${toastId}`);

    const authResponse: AuthResponse = responseJSON as AuthResponse;
    authResponse.username = username;
    authResponse.password = password;
    authResponse.refreshed = endpoint == refreshTokenEndpoint;
    authResponse.toastId = toastId || '';

    return authResponse;
}

function isNumber(n) {
    return !isNaN(parseFloat(n)) && !isNaN(n - 0);
}

const AuthProvider = (props: { children }) => {
    const [authenticated, updateAuthenticatedStatus] = React.useState<boolean>(false);
    const [username, updateUsername] = React.useState<string | undefined>();
    const [password, updatePassword] = React.useState<string | undefined>();
    const [doLogin, setDoLogin] = React.useState<boolean>(false);

    const shouldFetch = () => {
        if (authenticated) {
            return refreshTokenEndpoint;
        } else if (doLogin) {
            return loginEndpoint;
        }

        return null;
    };

    const onSuccess = (data: AuthResponse | void) => {
        if (data && data.token && data.expire) {
            console.log(`Refreshed token: ${data.token}. Expires at: ${data.expire}.`);
            localStorage.setItem('token', data.token);
            localStorage.setItem('token-expiration', (data.expire as number).toString());

            updateUsername(data.username);
            updatePassword(data.password);

            updateAuthenticatedStatus(true);

            const toastId: string | undefined = data['toastId'];
            const refreshed: boolean = data['refreshed'];

            console.log(`onSuccess toastId: ${toastId}`);

            if (refreshed) {
                toast.custom(
                    GetToastContentWithHeaderAndBody(
                        'Authentication Status Refreshed',
                        'Your log-in session has been automatically extended.',
                        'success',
                        () => toast.dismiss(toastId),
                    ),
                    { id: toastId },
                );
            } else {
                toast.custom(
                    GetToastContentWithHeaderAndBody(
                        'Authentication Successful',
                        `Successfully logged-in as user "${data['username']}"`,
                        'success',
                        () => toast.dismiss(toastId),
                    ),
                    { id: toastId },
                );
            }
        }
    };

    const { error } = useSWR(
        shouldFetch,
        (input: RequestInfo | URL) => {
            setDoLogin(false);
            return tokenFetcher(input, username, password, authenticated);
        },
        {
            revalidateOnFocus: false,
            revalidateOnMount: false,
            revalidateOnReconnect: false,
            revalidateIfStale: false,
            onSuccess: onSuccess,
            refreshInterval: (latestData) => {
                if (latestData && latestData.expire) {
                    let expire: string | number = latestData.expire;

                    if (!isNumber(expire)) {
                        expire = Date.parse(expire as string);
                    }

                    console.log(`Token is set to expire at ${expire}.`);

                    const expireIn: number = (expire as number) - Date.now();

                    console.log(
                        `Will automatically refresh JWT token in ${RoundToThreeDecimalPlaces(expireIn / 1000.0)} seconds`,
                    );

                    return expireIn * 0.9;
                }

                return MAX_SAFE_INTEGER;
            },
        },
    );

    const doMutate = async (user: string, passwd: string) => {
        console.log(
            `Manually refreshing token now with username "${user}". Current authenticated status: ${authenticated}`,
        );

        const toastId: string = toast.custom((t: Toast) => (
            <Alert
                isInline
                variant={'info'}
                title={'Logging in...'}
                onTimeout={() => toast.dismiss(t.id)}
                customIcon={<SpinnerIcon className={'loading-icon-spin-pulse'} />}
                actionClose={<AlertActionCloseButton onClose={() => toast.dismiss(t.id)} />}
            />
        ));

        console.log(`doMutate toastId: ${toastId}`);

        let response: AuthResponse | void = undefined;
        try {
            response = await tokenFetcher(
                authenticated ? refreshTokenEndpoint : loginEndpoint,
                user,
                passwd,
                authenticated,
                toastId,
            );
        } catch (err) {
            toast.custom(
                (t: Toast) =>
                    GetToastContentWithHeaderAndBody('Login Attempt Failed', (err as Error).message, 'danger', () =>
                        toast.dismiss(t.id),
                    ),
                { id: toastId },
            );

            throw err;
        }

        onSuccess(response);

        return true;
    };
    return (
        <AuthorizationContext.Provider
            value={{
                authenticated: authenticated,
                setAuthenticated: (nextAuthStatus: boolean) => {
                    // If the user was authenticated and is now being set to unauthenticated, then display an error.
                    if (authenticated && !nextAuthStatus) {
                        toast.custom(
                            () =>
                                GetToastContentWithHeaderAndBody(
                                    'Logged Out',
                                    "You've been logged-out. Please reauthenticate to continue using the Cluster Dashboard.",
                                    'danger',
                                    () => toast.dismiss(toastIdLoggedOut),
                                ),
                            { id: toastIdLoggedOut },
                        );
                    }

                    updateAuthenticatedStatus(nextAuthStatus);
                },
                username: username,
                setUsername: (user: string | undefined) => {
                    updateUsername(user);
                },
                password: password,
                setPassword: (password: string | undefined) => {
                    updatePassword(password);
                },
                mutateToken: doMutate,
                error: error,
            }}
        >
            {props.children}
        </AuthorizationContext.Provider>
    );
};

export { AuthProvider, AuthorizationContext, AuthContext };
