import { RoundToThreeDecimalPlaces } from '@Components/Modals';
import { GetPathForFetch } from '@src/Utils/path_utils';
import { GetToastContentWithHeaderAndBody } from '@src/Utils/toast_utils';
import { MAX_SAFE_INTEGER } from 'lib0/number';
import React from 'react';
import { Toast, toast } from 'react-hot-toast';
import useSWR from 'swr';

type AuthContext = {
    authenticated: boolean;
    setAuthenticated: (auth: boolean) => void;
    username: string | undefined;
    password: string | undefined;
    setUsername: (username: string | undefined) => void;
    setPassword: (password: string | undefined) => void;
    mutateToken: (username: string, password: string) => Promise<void>;
    error: any;
};

const initialState: AuthContext = {
    authenticated: false,
    setAuthenticated: () => {},
    username: undefined,
    setUsername: () => {},
    password: undefined,
    setPassword: () => {},
    mutateToken: async () => {},
    error: undefined,
};

const refreshTokenEndpoint: string = GetPathForFetch('/refresh_token');
const loginEndpoint: string = GetPathForFetch('/authenticate');

const AuthorizationContext = React.createContext<AuthContext>(initialState);

const tokenFetcher = async (
    endpoint: RequestInfo | URL,
    username: string | undefined,
    password: string | undefined,
    currentlyAuthenticated: boolean,
) => {
    console.log(
        `Refreshing token. Endpoint: "${endpoint}". Username: "${username}". Password: "${password}". Currently authenticated: ${currentlyAuthenticated}.`,
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
    const responseJSON = await response.json();

    if (response.status !== 200) {
        console.log(`Authenticate failed. Could not log in:\n${JSON.stringify(responseJSON, null, 2)}`);
        throw new Error(`HTTP ${response.status} ${response.statusText}: ${responseJSON.message}`);
    } else {
        console.log(`Fetched JWT token:\n${JSON.stringify(responseJSON, null, 2)}`);
    }

    response['username'] = username;
    response['password'] = password;

    return responseJSON;
};

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

    const onSuccess = (data: any) => {
        if (data && data['token'] && data['expire']) {
            console.log(`Refreshed token: ${data['token']}. Expires at: ${data['expire']}.`);
            localStorage.setItem('token', data['token']);
            localStorage.setItem('token-expiration', data['expire']);

            updateAuthenticatedStatus(true);
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
                if (latestData && latestData['expire']) {
                    const expire: string = latestData['expire'];
                    const expireUnix: number = Date.parse(expire);

                    console.log(`Token is set to expire at ${expireUnix}.`);

                    const expireIn: number = expireUnix - Date.now();

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

        let response: Response | undefined = undefined;
        try {
            response = await tokenFetcher(
                authenticated ? refreshTokenEndpoint : loginEndpoint,
                user,
                passwd,
                authenticated,
            );
        } catch (err) {
            toast.custom((t: Toast) =>
                GetToastContentWithHeaderAndBody('Login Attempt Failed', (err as Error).message, 'danger', () =>
                    toast.dismiss(t.id),
                ),
            );

            throw err;
        }

        onSuccess(response);
    };

    return (
        <AuthorizationContext.Provider
            value={{
                authenticated: authenticated,
                setAuthenticated: (nextAuthStatus: boolean) => {
                    // If the user was authenticated and is now being set to unauthenticated, then display an error.
                    if (authenticated && !nextAuthStatus) {
                        toast.error((t: Toast) =>
                            GetToastContentWithHeaderAndBody(
                                'Logged Out',
                                "You've been logged-out. Please reauthenticate to continue using the Cluster Dashboard.",
                                'danger',
                                () => toast.dismiss(t.id),
                            ),
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
