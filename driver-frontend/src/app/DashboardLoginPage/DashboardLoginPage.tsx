import { Ds2Icon } from '@Icons/Ds2Icon';
import { ListItem, ListVariant, LoginFooterItem, LoginForm, LoginPage } from '@patternfly/react-core';
import { GithubIcon } from '@patternfly/react-icons';
import ExclamationCircleIcon from '@patternfly/react-icons/dist/esm/icons/exclamation-circle-icon';
import logo_greyscale from '@src/app/bgimages/icon_greyscale.svg';
import logo from '@src/app/bgimages/WorkloadDriver-Logo.svg';
import React from 'react';

interface DashboardLoginPageProps {
    onSuccessfulLogin: () => void;
}

export const DashboardLoginPage: React.FunctionComponent<DashboardLoginPageProps> = (
    props: DashboardLoginPageProps,
) => {
    const [showHelperText, setShowHelperText] = React.useState(false);
    const [username, setUsername] = React.useState('');
    const [isValidUsername, setIsValidUsername] = React.useState(true);
    const [password, setPassword] = React.useState('');
    const [isValidPassword, setIsValidPassword] = React.useState(true);

    const handleUsernameChange = (_event: React.FormEvent<HTMLInputElement>, value: string) => {
        setUsername(value);
    };

    const handlePasswordChange = (_event: React.FormEvent<HTMLInputElement>, value: string) => {
        setPassword(value);
    };

    const onLoginButtonClick = async (event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
        event.preventDefault();

        const request = new Request('api/authenticate', {
            method: 'POST',
            body: JSON.stringify({ username, password }),
            headers: new Headers({ 'Content-Type': 'application/json' }),
        });
        const response = await fetch(request);
        if (response.status != 200) {
            setIsValidPassword(false);
            setIsValidUsername(false);
            setShowHelperText(true);
        } else {
            setIsValidPassword(true);
            setIsValidUsername(true);
            setShowHelperText(false);
            props.onSuccessfulLogin();
        }
    };

    const listItem = (
        <React.Fragment>
            <ListItem icon={<GithubIcon />}>
                <LoginFooterItem href="https://github.com/Scusemua/workload-driver-react">
                    Dashboard GitHub Page
                </LoginFooterItem>
            </ListItem>
            <ListItem icon={<Ds2Icon scale={1.5} />}>
                <LoginFooterItem href="https://ds2-lab.github.io/">
                    UVA D<span className="lowerc">S</span>
                    <sup>2</sup> Research Lab
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
            isValidPassword={isValidPassword}
            rememberMeLabel="Keep me logged in for 30 days."
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
            textContent="Distributed Jupyter Notebook Cluster -- Admin Dashboard & Workload Orchestrator"
            loginTitle="Log in to access the Dashboard"
            loginSubtitle="Enter the configured admin credentials"
        >
            {loginForm}
        </LoginPage>
    );
};
