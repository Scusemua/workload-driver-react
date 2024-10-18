import { RoundToThreeDecimalPlaces } from '@Components/Modals';
import { MAX_SAFE_INTEGER } from 'lib0/number';
import React from 'react';
import useSWR, { KeyedMutator, useSWRConfig } from 'swr';

type AuthContext = {
    authenticated: boolean;
    setAuthenticated: (auth: boolean) => void;
    username: string | undefined;
    password: string | undefined;
    setUsername: (username: string | undefined) => void;
    setPassword: (password: string | undefined) => void;
    mutateToken: KeyedMutator<any> | undefined;
    error: any;
};

const initialState: AuthContext = {
    authenticated: false,
    setAuthenticated: () => {},
    username: undefined,
    setUsername: () => {},
    password: undefined,
    setPassword: () => {},
    mutateToken: undefined,
    error: undefined,
};

const refreshTokenEndpoint: string = '/refresh_token';
const loginEndpoint: string = '/authenticate';

const AuthorizationContext = React.createContext<AuthContext>(initialState);

const tokenFetcher = async (
    endpoint: RequestInfo | URL,
    username: string | undefined,
    password: string | undefined,
    currentlyAuthenticated: boolean,
) => {
    console.log(`Refreshing token. Endpoint: "${endpoint}". Username: "${username}". Password: "${password}". Currently authenticated: ${currentlyAuthenticated}.`);

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

    console.log(`Fetched JWT token:\n${JSON.stringify(responseJSON, null, 2)}`);

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

    const { mutate } = useSWRConfig();

    const doMutate = async () => {
        console.log(`Manually refreshing token now. Currently authenticated: ${authenticated}`);
        if (authenticated) {
            return await mutate(refreshTokenEndpoint);
        } else {
            setDoLogin(true);
        }
    };

    // React.useEffect(() => {
    //   if (authenticated) {
    //     const currentToken: string = localStorage.getItem('token');
    //     const expireAt: string = localStorage.getItem('token-expiration');
    //
    //     const ttl: number = (Number.parseInt(expireAt) - Date.now()) + 2500; // Add 2.5 sec for a bit of a buffer.
    //
    //     if (ttl < 0) {
    //       return;
    //     }
    //
    //     const intervalId = setInterval(() => {
    //       if (localStorage.getItem('token') == currentToken) {
    //         updateAuthenticatedStatus(false);
    //       }
    //     }, ttl);
    //
    //     return () => clearInterval(intervalId);
    //   }
    //
    //   return;
    // }, [authenticated])

    return (
        <AuthorizationContext.Provider
            value={{
                authenticated: authenticated,
                setAuthenticated: (auth: boolean) => {
                    updateAuthenticatedStatus(auth);
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
