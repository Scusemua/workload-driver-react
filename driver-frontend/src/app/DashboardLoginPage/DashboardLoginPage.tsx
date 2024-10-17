import { Ds2Icon } from '@Icons/Ds2Icon';
import { ListItem, ListVariant, LoginFooterItem, LoginForm, LoginPage } from '@patternfly/react-core';
import { ExternalLinkAltIcon, GithubIcon } from '@patternfly/react-icons';
import ExclamationCircleIcon from '@patternfly/react-icons/dist/esm/icons/exclamation-circle-icon';
import { AuthorizationContext } from '@Providers/AuthProvider';
import logo_greyscale from '@src/app/bgimages/icon_greyscale.svg';
import logo from '@src/app/bgimages/WorkloadDriver-Logo.svg';
import React from 'react';

interface DashboardLoginPageProps {
    onSuccessfulLogin: (token: string, expiration: string) => void;
}

export const DashboardLoginPage: React.FunctionComponent<DashboardLoginPageProps> = (
    props: DashboardLoginPageProps,
) => {
    const [showHelperText, setShowHelperText] = React.useState(false);
    // const [username, setUsername] = React.useState('');
    const [isValidUsername, setIsValidUsername] = React.useState(true);
    // const [password, setPassword] = React.useState('');
    const [isValidPassword, setIsValidPassword] = React.useState(true);

    const { username, setUsername, password, setPassword, mutateToken, error } = React.useContext(AuthorizationContext);

    React.useEffect(() => {
        if (error) {
            setIsValidPassword(false);
            setIsValidUsername(false);
            setShowHelperText(true);
        } else {
            setIsValidPassword(true);
            setIsValidUsername(true);
            setShowHelperText(false);
        }
    }, [error]);

    const handleUsernameChange = (_event: React.FormEvent<HTMLInputElement>, value: string) => {
        setUsername(value);
    };

    const handlePasswordChange = (_event: React.FormEvent<HTMLInputElement>, value: string) => {
        setPassword(value);
    };

    const onLoginButtonClick = async (event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
        event.preventDefault();

        // const req: RequestInit = {
        //     method: 'POST',
        //     headers: {
        //         'Content-Type': 'application/json',
        //     },
        //     body: JSON.stringify({ username: username, password: password }),
        // };

        if (mutateToken) {
            await mutateToken();
        }

        // const response = await fetch('authenticate', req);
        // if (response.status != 200) {
        //     setIsValidPassword(false);
        //     setIsValidUsername(false);
        //     setShowHelperText(true);
        // } else {
        //     setIsValidPassword(true);
        //     setIsValidUsername(true);
        //     setShowHelperText(false);
        //
        //     const body = await response.json();
        //     const token: string = body['token'];
        //     const expiration: string = body['expire'];
        //     props.onSuccessfulLogin(token, expiration);
        // }
    };

    // icon={<Ds2Icon scale={1.5} />}
    const listItem = (
        <React.Fragment>
            <ListItem icon={<GithubIcon />}>
                <LoginFooterItem href="https://github.com/Scusemua/workload-driver-react">
                    Source Code
                </LoginFooterItem>
            </ListItem>
            <ListItem icon={<ExternalLinkAltIcon/>}>
                <LoginFooterItem href="https://ds2-lab.github.io/">
                    D<span className="lowerc">S</span>
                    <sup>2</sup> Research Lab @ UVA
                </LoginFooterItem>
            </ListItem>
        </React.Fragment>
    );

    const loginForm = (
        <LoginForm
            showHelperText={showHelperText}
            helperText="Invalid login credentials."
            helperTextIcon={<ExclamationCircleIcon />}
            usernameLabel="Username"
            usernameValue={username}
            onChangeUsername={handleUsernameChange}
            isValidUsername={isValidUsername}
            passwordLabel="Password"
            passwordValue={password}
            onChangePassword={handlePasswordChange}
            isShowPasswordEnabled
            isValidPassword={isValidPassword}
            onLoginButtonClick={onLoginButtonClick}
            loginButtonLabel="Log in"
        />
    );

    return (
        <LoginPage
            footerListVariants={ListVariant.inline}
            brandImgSrc={logo}
            brandImgAlt="Distributed Dashboard Logo"
            backgroundImgSrc={logo_greyscale}
            footerListItems={listItem}
            textContent="Distributed Notebook Cluster | Admin Dashboard & Workload Orchestrator"
            loginTitle="Log in to access the Dashboard"
            loginSubtitle="Enter the configured admin credentials"
        >
            {loginForm}
        </LoginPage>
    );
};
